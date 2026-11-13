package config

import (
	"strings"
	"testing"
)

func TestDefaultCoordinationConfigIsFailClosedSingleInstance(t *testing.T) {
	cfg := DefaultConfig()
	coordination := cfg.Runtime.Coordination
	if coordination.Backend != CoordinationBackendMemory {
		t.Fatalf("default backend = %q", coordination.Backend)
	}
	if coordination.MailboxBackend != MailboxBackendMemory {
		t.Fatalf("default mailbox backend = %q", coordination.MailboxBackend)
	}
	if coordination.InstanceRegistry != InstanceRegistryDisabled {
		t.Fatalf("default instance registry = %q", coordination.InstanceRegistry)
	}
	if coordination.Redis.Addr != "" {
		t.Fatalf("default redis addr = %q", coordination.Redis.Addr)
	}
	if err := ValidateRuntimeCoordinationConfig(coordination); err != nil {
		t.Fatalf("default posture must validate: %v", err)
	}
}

func TestCoordinationConfigDefaultsSurviveStrictLoad(t *testing.T) {
	cfg := DefaultConfig()
	// The strict loader must keep every default value intact when no
	// runtime.coordination.* key is present in the source document.
	if cfg.Runtime.Coordination.Redis.LeaseTTLMS != 30000 || cfg.Runtime.Coordination.Redis.OpTimeoutMS != 1000 {
		t.Fatalf("redis defaults drifted: %+v", cfg.Runtime.Coordination.Redis)
	}
	if cfg.Runtime.Coordination.Redis.DialTimeoutMS != 2000 || cfg.Runtime.Coordination.Redis.InflightTimeoutMS != 60000 {
		t.Fatalf("redis timeout defaults drifted: %+v", cfg.Runtime.Coordination.Redis)
	}
	if cfg.Runtime.Coordination.Redis.KeyPrefix != "baymax" || cfg.Runtime.Coordination.Redis.DB != 0 {
		t.Fatalf("redis identity defaults drifted: %+v", cfg.Runtime.Coordination.Redis)
	}
}

func TestValidateCoordinationConfigPositiveRedisPosture(t *testing.T) {
	coordination := DefaultRuntimeCoordinationConfig()
	coordination.Backend = CoordinationBackendRedis
	coordination.MailboxBackend = MailboxBackendRedis
	coordination.InstanceRegistry = InstanceRegistryRedis
	coordination.Redis.Addr = "127.0.0.1:6379"
	if err := ValidateRuntimeCoordinationConfig(coordination); err != nil {
		t.Fatalf("valid redis posture = %v", err)
	}
}

func TestValidateCoordinationConfigFailFastRules(t *testing.T) {
	base := DefaultRuntimeCoordinationConfig()

	cases := []struct {
		name    string
		mutate  func(*RuntimeCoordinationConfig)
		wantErr string
	}{
		{
			name:    "redis backend requires addr",
			mutate:  func(c *RuntimeCoordinationConfig) { c.Backend = CoordinationBackendRedis },
			wantErr: "redis.addr is required",
		},
		{
			name: "file backend requires dir",
			mutate: func(c *RuntimeCoordinationConfig) {
				c.Backend = CoordinationBackendFile
			},
			wantErr: "file_dir is required",
		},
		{
			name: "unknown backend",
			mutate: func(c *RuntimeCoordinationConfig) {
				c.Backend = "etcd"
			},
			wantErr: "runtime.coordination.backend must be one of",
		},
		{
			name: "op timeout must be below lease ttl",
			mutate: func(c *RuntimeCoordinationConfig) {
				c.Backend = CoordinationBackendRedis
				c.Redis.Addr = "127.0.0.1:6379"
				c.Redis.OpTimeoutMS = c.Redis.LeaseTTLMS
			},
			wantErr: "op_timeout_ms must be <",
		},
		{
			name: "redis mailbox backend requires redis coordinator",
			mutate: func(c *RuntimeCoordinationConfig) {
				c.MailboxBackend = MailboxBackendRedis
			},
			wantErr: "mailbox_backend=redis requires",
		},
		{
			name: "redis instance registry requires redis coordinator",
			mutate: func(c *RuntimeCoordinationConfig) {
				c.InstanceRegistry = InstanceRegistryRedis
			},
			wantErr: "instance_registry=redis requires",
		},
		{
			name: "unknown instance registry mode",
			mutate: func(c *RuntimeCoordinationConfig) {
				c.InstanceRegistry = "consul"
			},
			wantErr: "runtime.coordination.instance_registry must be one of",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			candidate := base
			tc.mutate(&candidate)
			err := ValidateRuntimeCoordinationConfig(candidate)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("validation error = %v, want substring %q", err, tc.wantErr)
			}
		})
	}
}

func TestConfigValidateRejectsInvalidCoordinationDomain(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Runtime.Coordination.Backend = "zookeeper"
	if err := Validate(cfg); err == nil {
		t.Fatal("invalid coordination backend accepted by config validation")
	}
}
