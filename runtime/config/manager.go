package config

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	runtimediag "github.com/FelixSeptem/baymax/runtime/diagnostics"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

type ManagerOptions struct {
	FilePath        string
	EnvPrefix       string
	EnableHotReload bool
}

type Snapshot struct {
	Config   Config         `json:"config"`
	LoadedAt time.Time      `json:"loaded_at"`
	Source   SnapshotSource `json:"source"`
}

type SnapshotSource struct {
	FilePath  string `json:"file_path,omitempty"`
	EnvPrefix string `json:"env_prefix"`
}

type Manager struct {
	filePath  string
	envPrefix string

	snap atomic.Value // *Snapshot
	diag *runtimediag.Store

	watchStarted atomic.Bool
	stopOnce     sync.Once
	stopCh       chan struct{}
}

func NewManager(opts ManagerOptions) (*Manager, error) {
	loadOpts := LoadOptions{FilePath: opts.FilePath, EnvPrefix: opts.EnvPrefix}
	cfg, _, err := loadWithSnapshot(loadOpts)
	if err != nil {
		return nil, err
	}

	m := &Manager{
		filePath:  strings.TrimSpace(opts.FilePath),
		envPrefix: strings.TrimSpace(opts.EnvPrefix),
		diag: runtimediag.NewStore(
			cfg.Diagnostics.MaxCallRecords,
			cfg.Diagnostics.MaxRunRecords,
			cfg.Diagnostics.MaxReloadErrors,
			cfg.Diagnostics.MaxSkillRecords,
		),
		stopCh: make(chan struct{}),
	}
	if m.envPrefix == "" {
		m.envPrefix = DefaultEnvPrefix
	}
	m.snap.Store(&Snapshot{
		Config:   cfg,
		LoadedAt: time.Now(),
		Source: SnapshotSource{
			FilePath:  m.filePath,
			EnvPrefix: m.envPrefix,
		},
	})
	if opts.EnableHotReload || cfg.Reload.Enabled {
		if err := m.Watch(context.Background()); err != nil {
			return nil, err
		}
	}
	return m, nil
}

func (m *Manager) Close() error {
	m.stopOnce.Do(func() { close(m.stopCh) })
	return nil
}

func (m *Manager) Watch(ctx context.Context) error {
	if strings.TrimSpace(m.filePath) == "" {
		return fmt.Errorf("hot reload requires a config file path")
	}
	if !m.watchStarted.CompareAndSwap(false, true) {
		return nil
	}

	w := viper.New()
	w.SetConfigFile(m.filePath)
	w.SetConfigType("yaml")
	if err := w.ReadInConfig(); err != nil {
		return fmt.Errorf("watch runtime config: %w", err)
	}

	events := make(chan struct{}, 1)
	w.OnConfigChange(func(_ fsnotify.Event) {
		select {
		case events <- struct{}{}:
		default:
		}
	})
	w.WatchConfig()

	go m.watchLoop(ctx, events)
	return nil
}

func (m *Manager) watchLoop(ctx context.Context, events <-chan struct{}) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopCh:
			return
		case <-events:
			debounce := m.EffectiveConfig().Reload.Debounce
			if debounce <= 0 {
				debounce = 100 * time.Millisecond
			}
			timer := time.NewTimer(debounce)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-m.stopCh:
				timer.Stop()
				return
			case <-timer.C:
			}
			m.reload()
		}
	}
}

func (m *Manager) reload() {
	cfg, _, err := loadWithSnapshot(LoadOptions{FilePath: m.filePath, EnvPrefix: m.envPrefix})
	if err != nil {
		m.diag.AddReload(runtimediag.ReloadRecord{Time: time.Now(), Success: false, Error: err.Error()})
		return
	}
	m.snap.Store(&Snapshot{
		Config:   cfg,
		LoadedAt: time.Now(),
		Source: SnapshotSource{
			FilePath:  m.filePath,
			EnvPrefix: m.envPrefix,
		},
	})
	m.diag.Resize(
		cfg.Diagnostics.MaxCallRecords,
		cfg.Diagnostics.MaxRunRecords,
		cfg.Diagnostics.MaxReloadErrors,
		cfg.Diagnostics.MaxSkillRecords,
	)
	m.diag.AddReload(runtimediag.ReloadRecord{Time: time.Now(), Success: true})
}

func (m *Manager) EffectiveConfig() Config {
	s := m.snapshot()
	if s == nil {
		return DefaultConfig()
	}
	return s.Config
}

func (m *Manager) CurrentSnapshot() Snapshot {
	s := m.snapshot()
	if s == nil {
		return Snapshot{Config: DefaultConfig(), LoadedAt: time.Now()}
	}
	return *s
}

func (m *Manager) EffectiveConfigSanitized() map[string]any {
	s := m.snapshot()
	if s == nil {
		raw, _ := toMap(DefaultConfig())
		return runtimediag.SanitizeMap(raw)
	}
	raw, err := toMap(s.Config)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}
	return runtimediag.SanitizeMap(raw)
}

func (m *Manager) ResolvePolicy(profile string, override *types.MCPRuntimePolicy) (types.MCPRuntimePolicy, error) {
	return ResolveMCPPolicyWithConfig(m.EffectiveConfig(), profile, override)
}

func (m *Manager) RecordCall(rec runtimediag.CallRecord) {
	m.diag.AddCall(rec)
}

func (m *Manager) RecordRun(rec runtimediag.RunRecord) {
	m.diag.AddRun(rec)
}

func (m *Manager) RecordReload(rec runtimediag.ReloadRecord) {
	m.diag.AddReload(rec)
}

func (m *Manager) RecordSkill(rec runtimediag.SkillRecord) {
	m.diag.AddSkill(rec)
}

func (m *Manager) RecentCalls(n int) []runtimediag.CallRecord {
	return m.diag.RecentCalls(n)
}

func (m *Manager) RecentRuns(n int) []runtimediag.RunRecord {
	return m.diag.RecentRuns(n)
}

func (m *Manager) RecentReloads(n int) []runtimediag.ReloadRecord {
	return m.diag.RecentReloads(n)
}

func (m *Manager) RecentSkills(n int) []runtimediag.SkillRecord {
	return m.diag.RecentSkills(n)
}

func (m *Manager) snapshot() *Snapshot {
	v := m.snap.Load()
	if v == nil {
		return nil
	}
	s, _ := v.(*Snapshot)
	return s
}
