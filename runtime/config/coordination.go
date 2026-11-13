package config

import (
	"errors"
	"fmt"
	"strings"
)

// Runtime coordination backend selection. The coordination domain owns the
// distributed source-owner coordinator, the cross-instance mailbox backend and
// the instance registry wiring. The default memory/file posture is unchanged:
// nothing here is enabled unless an operator explicitly configures redis.
const (
	CoordinationBackendMemory = "memory"
	CoordinationBackendFile   = "file"
	CoordinationBackendRedis  = "redis"

	InstanceRegistryDisabled = "disabled"
	InstanceRegistryRedis    = "redis"
)

// RuntimeCoordinationConfig is the `runtime.coordination.*` domain. Every
// field must keep additive + nullable + default semantics: absent keys fall
// back to the single-instance defaults, and enabling redis is the only way a
// cross_machine deployment guarantee may be advertised.
type RuntimeCoordinationConfig struct {
	Backend          string                         `json:"backend"`
	FileDir          string                         `json:"file_dir"`
	Redis            RuntimeCoordinationRedisConfig `json:"redis"`
	MailboxBackend   string                         `json:"mailbox_backend"`
	InstanceRegistry string                         `json:"instance_registry"`
}

// RuntimeCoordinationRedisConfig carries the injected Redis wiring. Durations
// use the house _ms int convention so strict parsing stays lossless.
type RuntimeCoordinationRedisConfig struct {
	Addr              string `json:"addr"`
	Username          string `json:"username"`
	Password          string `json:"password"`
	DB                int    `json:"db"`
	DialTimeoutMS     int    `json:"dial_timeout_ms"`
	OpTimeoutMS       int    `json:"op_timeout_ms"`
	KeyPrefix         string `json:"key_prefix"`
	LeaseTTLMS        int    `json:"lease_ttl_ms"`
	InflightTimeoutMS int    `json:"inflight_timeout_ms"`
}

// DefaultRuntimeCoordinationConfig returns the fail-closed single-instance
// posture: no redis, no registry, mailbox on memory.
func DefaultRuntimeCoordinationConfig() RuntimeCoordinationConfig {
	return RuntimeCoordinationConfig{
		Backend:          CoordinationBackendMemory,
		FileDir:          "",
		MailboxBackend:   MailboxBackendMemory,
		InstanceRegistry: InstanceRegistryDisabled,
		Redis: RuntimeCoordinationRedisConfig{
			Addr:              "",
			Username:          "",
			Password:          "",
			DB:                0,
			DialTimeoutMS:     2000,
			OpTimeoutMS:       1000,
			KeyPrefix:         "baymax",
			LeaseTTLMS:        30000,
			InflightTimeoutMS: 60000,
		},
	}
}

// ValidateRuntimeCoordinationConfig enforces fail-fast rules. Invalid
// configuration must never start or survive a hot reload (atomic rollback
// happens at the manager layer by rejecting the candidate snapshot).
func ValidateRuntimeCoordinationConfig(cfg RuntimeCoordinationConfig) error {
	backend := strings.ToLower(strings.TrimSpace(cfg.Backend))
	switch backend {
	case CoordinationBackendMemory:
	case CoordinationBackendFile:
		if strings.TrimSpace(cfg.FileDir) == "" {
			return errors.New("runtime.coordination.file_dir is required when runtime.coordination.backend=file")
		}
	case CoordinationBackendRedis:
		if strings.TrimSpace(cfg.Redis.Addr) == "" {
			return errors.New("runtime.coordination.redis.addr is required when runtime.coordination.backend=redis")
		}
		if cfg.Redis.LeaseTTLMS <= 0 {
			return errors.New("runtime.coordination.redis.lease_ttl_ms must be > 0")
		}
		if cfg.Redis.InflightTimeoutMS <= 0 {
			return errors.New("runtime.coordination.redis.inflight_timeout_ms must be > 0")
		}
		if cfg.Redis.OpTimeoutMS <= 0 {
			return errors.New("runtime.coordination.redis.op_timeout_ms must be > 0")
		}
		if cfg.Redis.OpTimeoutMS >= cfg.Redis.LeaseTTLMS {
			return errors.New("runtime.coordination.redis.op_timeout_ms must be < runtime.coordination.redis.lease_ttl_ms")
		}
	default:
		return fmt.Errorf(
			"runtime.coordination.backend must be one of [%s,%s,%s], got %q",
			CoordinationBackendMemory,
			CoordinationBackendFile,
			CoordinationBackendRedis,
			cfg.Backend,
		)
	}

	mailboxBackend := strings.ToLower(strings.TrimSpace(cfg.MailboxBackend))
	switch mailboxBackend {
	case MailboxBackendMemory, MailboxBackendFile:
	case MailboxBackendRedis:
		if backend != CoordinationBackendRedis {
			return errors.New("runtime.coordination.mailbox_backend=redis requires runtime.coordination.backend=redis")
		}
	default:
		return fmt.Errorf(
			"runtime.coordination.mailbox_backend must be one of [%s,%s,%s], got %q",
			MailboxBackendMemory,
			MailboxBackendFile,
			MailboxBackendRedis,
			cfg.MailboxBackend,
		)
	}

	registry := strings.ToLower(strings.TrimSpace(cfg.InstanceRegistry))
	switch registry {
	case InstanceRegistryDisabled:
	case InstanceRegistryRedis:
		if backend != CoordinationBackendRedis {
			return errors.New("runtime.coordination.instance_registry=redis requires runtime.coordination.backend=redis")
		}
	default:
		return fmt.Errorf(
			"runtime.coordination.instance_registry must be one of [%s,%s], got %q",
			InstanceRegistryDisabled,
			InstanceRegistryRedis,
			cfg.InstanceRegistry,
		)
	}
	return nil
}
