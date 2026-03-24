package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	TimeoutResolutionSourceProfile = "profile"
	TimeoutResolutionSourceDomain  = "domain"
	TimeoutResolutionSourceRequest = "request"
)

type TimeoutResolutionInput struct {
	RequestedProfile string
	DomainTimeout    time.Duration
	RequestTimeout   time.Duration
}

type TimeoutResolutionResult struct {
	EffectiveProfile string
	EffectiveTimeout time.Duration
	Source           string
	Trace            string
}

type TimeoutResolutionTrace struct {
	Version            string `json:"version"`
	Profile            string `json:"profile"`
	ProfileTimeoutMs   int64  `json:"profile_timeout_ms"`
	DomainTimeoutMs    int64  `json:"domain_timeout_ms"`
	RequestTimeoutMs   int64  `json:"request_timeout_ms"`
	SelectedSource     string `json:"selected_source"`
	EffectiveTimeoutMs int64  `json:"effective_timeout_ms"`
}

func ResolveOperationTimeout(cfg Config, in TimeoutResolutionInput) (TimeoutResolutionResult, error) {
	profile, err := ResolveOperationProfile(cfg, in.RequestedProfile)
	if err != nil {
		return TimeoutResolutionResult{}, err
	}
	profileTimeout, err := timeoutForOperationProfile(cfg.Runtime.OperationProfiles, profile)
	if err != nil {
		return TimeoutResolutionResult{}, err
	}
	effective := profileTimeout
	source := TimeoutResolutionSourceProfile
	if in.DomainTimeout > 0 {
		effective = in.DomainTimeout
		source = TimeoutResolutionSourceDomain
	}
	if in.RequestTimeout > 0 {
		effective = in.RequestTimeout
		source = TimeoutResolutionSourceRequest
	}
	if effective <= 0 {
		return TimeoutResolutionResult{}, errors.New("resolved timeout must be > 0")
	}
	trace := TimeoutResolutionTrace{
		Version:            "v1",
		Profile:            profile,
		ProfileTimeoutMs:   profileTimeout.Milliseconds(),
		DomainTimeoutMs:    in.DomainTimeout.Milliseconds(),
		RequestTimeoutMs:   in.RequestTimeout.Milliseconds(),
		SelectedSource:     source,
		EffectiveTimeoutMs: effective.Milliseconds(),
	}
	traceBlob, err := json.Marshal(trace)
	if err != nil {
		return TimeoutResolutionResult{}, fmt.Errorf("marshal timeout resolution trace: %w", err)
	}
	return TimeoutResolutionResult{
		EffectiveProfile: profile,
		EffectiveTimeout: effective,
		Source:           source,
		Trace:            string(traceBlob),
	}, nil
}

func ResolveOperationProfile(cfg Config, requested string) (string, error) {
	profile := strings.ToLower(strings.TrimSpace(requested))
	if profile == "" {
		profile = strings.ToLower(strings.TrimSpace(cfg.Runtime.OperationProfiles.DefaultProfile))
	}
	switch profile {
	case OperationProfileLegacy, OperationProfileInteractive, OperationProfileBackground, OperationProfileBatch:
		return profile, nil
	default:
		return "", fmt.Errorf(
			"operation profile must be one of [%s,%s,%s,%s], got %q",
			OperationProfileLegacy,
			OperationProfileInteractive,
			OperationProfileBackground,
			OperationProfileBatch,
			requested,
		)
	}
}

func timeoutForOperationProfile(cfg RuntimeOperationProfilesConfig, profile string) (time.Duration, error) {
	switch strings.ToLower(strings.TrimSpace(profile)) {
	case OperationProfileLegacy:
		if cfg.Legacy.Timeout <= 0 {
			return 0, errors.New("runtime.operation_profiles.legacy.timeout must be > 0")
		}
		return cfg.Legacy.Timeout, nil
	case OperationProfileInteractive:
		if cfg.Interactive.Timeout <= 0 {
			return 0, errors.New("runtime.operation_profiles.interactive.timeout must be > 0")
		}
		return cfg.Interactive.Timeout, nil
	case OperationProfileBackground:
		if cfg.Background.Timeout <= 0 {
			return 0, errors.New("runtime.operation_profiles.background.timeout must be > 0")
		}
		return cfg.Background.Timeout, nil
	case OperationProfileBatch:
		if cfg.Batch.Timeout <= 0 {
			return 0, errors.New("runtime.operation_profiles.batch.timeout must be > 0")
		}
		return cfg.Batch.Timeout, nil
	default:
		return 0, fmt.Errorf("unsupported operation profile %q", profile)
	}
}
