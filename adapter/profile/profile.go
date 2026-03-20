package profile

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const (
	ProfileV1Alpha0 = "v1alpha0"
	ProfileV1Alpha1 = "v1alpha1"
	CurrentProfile  = ProfileV1Alpha1
)

const (
	CodeUnknownProfileVersion = "adapter.contract-profile.unknown-version"
	CodeProfileOutOfWindow    = "adapter.contract-profile.out-of-window"
)

var (
	profilePattern = regexp.MustCompile(`^v([0-9]+)alpha([0-9]+)$`)
	knownProfiles  = []string{
		ProfileV1Alpha0,
		ProfileV1Alpha1,
	}
)

type Version struct {
	raw   string
	major int
	alpha int
}

func (v Version) String() string {
	return v.raw
}

type Window struct {
	Current       Version
	AllowPrevious bool
}

type Error struct {
	Code    string
	Profile string
	Message string
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if strings.TrimSpace(e.Profile) != "" {
		return fmt.Sprintf("[%s] %s: %s", e.Code, e.Profile, e.Message)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func Parse(raw string) (Version, error) {
	candidate := strings.ToLower(strings.TrimSpace(raw))
	matches := profilePattern.FindStringSubmatch(candidate)
	if len(matches) != 3 {
		return Version{}, &Error{
			Code:    CodeUnknownProfileVersion,
			Profile: candidate,
			Message: "profile must match v<major>alpha<minor>",
		}
	}
	major, err := strconv.Atoi(matches[1])
	if err != nil {
		return Version{}, &Error{
			Code:    CodeUnknownProfileVersion,
			Profile: candidate,
			Message: "invalid major version number",
		}
	}
	alpha, err := strconv.Atoi(matches[2])
	if err != nil {
		return Version{}, &Error{
			Code:    CodeUnknownProfileVersion,
			Profile: candidate,
			Message: "invalid alpha version number",
		}
	}
	if !isRecognized(candidate) {
		return Version{}, &Error{
			Code:    CodeUnknownProfileVersion,
			Profile: candidate,
			Message: "profile is not recognized",
		}
	}
	return Version{
		raw:   candidate,
		major: major,
		alpha: alpha,
	}, nil
}

func DefaultWindow() Window {
	current, _ := Parse(CurrentProfile)
	return Window{
		Current:       current,
		AllowPrevious: true,
	}
}

func NewWindow(currentProfile string, allowPrevious bool) (Window, error) {
	current, err := Parse(currentProfile)
	if err != nil {
		return Window{}, err
	}
	return Window{
		Current:       current,
		AllowPrevious: allowPrevious,
	}, nil
}

func ValidateCompatibility(profileVersion string, window Window) (Version, error) {
	profile, err := Parse(profileVersion)
	if err != nil {
		return Version{}, err
	}
	if profile.raw == window.Current.raw {
		return profile, nil
	}
	if window.AllowPrevious {
		if previous, ok := Previous(window.Current); ok && profile.raw == previous.raw {
			return profile, nil
		}
	}
	return Version{}, &Error{
		Code:    CodeProfileOutOfWindow,
		Profile: profile.raw,
		Message: "profile is outside runtime compatibility window",
	}
}

func Previous(current Version) (Version, bool) {
	for idx, raw := range knownProfiles {
		if raw != current.raw {
			continue
		}
		if idx <= 0 {
			return Version{}, false
		}
		previous, err := Parse(knownProfiles[idx-1])
		if err != nil {
			return Version{}, false
		}
		return previous, true
	}
	return Version{}, false
}

func RecognizedProfiles() []string {
	out := make([]string, 0, len(knownProfiles))
	out = append(out, knownProfiles...)
	return out
}

func isRecognized(profile string) bool {
	for _, known := range knownProfiles {
		if profile == known {
			return true
		}
	}
	return false
}
