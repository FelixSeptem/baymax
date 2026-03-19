package manifest

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var semverPattern = regexp.MustCompile(`^v?(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-([0-9A-Za-z.-]+))?(?:\+[0-9A-Za-z.-]+)?$`)

type semVersion struct {
	major int
	minor int
	patch int
	pre   []preToken
}

type preToken struct {
	raw       string
	numeric   bool
	numValue  int
	lexicText string
}

type semverComparator struct {
	op      string
	version semVersion
}

func evaluateSemverRange(expr, runtimeVersion string) (bool, error) {
	ver, err := parseSemver(runtimeVersion)
	if err != nil {
		return false, fmt.Errorf("invalid runtime version: %w", err)
	}
	comparators, err := parseSemverRange(expr)
	if err != nil {
		return false, err
	}
	for _, cmp := range comparators {
		ok := cmp.match(ver)
		if !ok {
			return false, nil
		}
	}
	return true, nil
}

func parseSemverRange(expr string) ([]semverComparator, error) {
	raw := strings.TrimSpace(expr)
	if raw == "" {
		return nil, fmt.Errorf("empty semver range expression")
	}
	normalized := strings.ReplaceAll(raw, ",", " ")
	tokens := strings.Fields(normalized)
	comparators := make([]semverComparator, 0, len(tokens))
	for _, token := range tokens {
		cmp, err := parseComparator(token)
		if err != nil {
			return nil, err
		}
		comparators = append(comparators, cmp)
	}
	if len(comparators) == 0 {
		return nil, fmt.Errorf("empty semver range expression")
	}
	return comparators, nil
}

func parseComparator(token string) (semverComparator, error) {
	raw := strings.TrimSpace(token)
	if raw == "" {
		return semverComparator{}, fmt.Errorf("invalid comparator: %q", token)
	}
	op := ""
	versionRaw := raw
	switch {
	case strings.HasPrefix(raw, ">="):
		op = ">="
		versionRaw = strings.TrimSpace(raw[2:])
	case strings.HasPrefix(raw, "<="):
		op = "<="
		versionRaw = strings.TrimSpace(raw[2:])
	case strings.HasPrefix(raw, "=="):
		op = "="
		versionRaw = strings.TrimSpace(raw[2:])
	case strings.HasPrefix(raw, ">"):
		op = ">"
		versionRaw = strings.TrimSpace(raw[1:])
	case strings.HasPrefix(raw, "<"):
		op = "<"
		versionRaw = strings.TrimSpace(raw[1:])
	case strings.HasPrefix(raw, "="):
		op = "="
		versionRaw = strings.TrimSpace(raw[1:])
	default:
		op = "="
	}
	if versionRaw == "" {
		return semverComparator{}, fmt.Errorf("invalid comparator: %q", token)
	}
	version, err := parseSemver(versionRaw)
	if err != nil {
		return semverComparator{}, fmt.Errorf("invalid comparator %q: %w", token, err)
	}
	return semverComparator{op: op, version: version}, nil
}

func parseSemver(input string) (semVersion, error) {
	match := semverPattern.FindStringSubmatch(strings.TrimSpace(input))
	if len(match) != 5 {
		return semVersion{}, fmt.Errorf("invalid semver %q", input)
	}
	major, err := strconv.Atoi(match[1])
	if err != nil {
		return semVersion{}, fmt.Errorf("invalid major version: %w", err)
	}
	minor, err := strconv.Atoi(match[2])
	if err != nil {
		return semVersion{}, fmt.Errorf("invalid minor version: %w", err)
	}
	patch, err := strconv.Atoi(match[3])
	if err != nil {
		return semVersion{}, fmt.Errorf("invalid patch version: %w", err)
	}
	pre, err := parsePreRelease(match[4])
	if err != nil {
		return semVersion{}, err
	}
	return semVersion{
		major: major,
		minor: minor,
		patch: patch,
		pre:   pre,
	}, nil
}

func parsePreRelease(raw string) ([]preToken, error) {
	in := strings.TrimSpace(raw)
	if in == "" {
		return nil, nil
	}
	parts := strings.Split(in, ".")
	out := make([]preToken, 0, len(parts))
	for _, part := range parts {
		segment := strings.TrimSpace(part)
		if segment == "" {
			return nil, fmt.Errorf("invalid pre-release %q", raw)
		}
		if isAllDigits(segment) {
			num, err := strconv.Atoi(segment)
			if err != nil {
				return nil, fmt.Errorf("invalid pre-release numeric %q", segment)
			}
			out = append(out, preToken{raw: segment, numeric: true, numValue: num})
			continue
		}
		out = append(out, preToken{raw: segment, lexicText: segment})
	}
	return out, nil
}

func (c semverComparator) match(v semVersion) bool {
	diff := compareSemver(v, c.version)
	switch c.op {
	case ">":
		return diff > 0
	case ">=":
		return diff >= 0
	case "<":
		return diff < 0
	case "<=":
		return diff <= 0
	case "=":
		return diff == 0
	default:
		return false
	}
}

func compareSemver(a, b semVersion) int {
	if a.major != b.major {
		if a.major > b.major {
			return 1
		}
		return -1
	}
	if a.minor != b.minor {
		if a.minor > b.minor {
			return 1
		}
		return -1
	}
	if a.patch != b.patch {
		if a.patch > b.patch {
			return 1
		}
		return -1
	}

	// A stable release has higher precedence than pre-release.
	if len(a.pre) == 0 && len(b.pre) == 0 {
		return 0
	}
	if len(a.pre) == 0 {
		return 1
	}
	if len(b.pre) == 0 {
		return -1
	}

	max := len(a.pre)
	if len(b.pre) > max {
		max = len(b.pre)
	}
	for i := 0; i < max; i++ {
		if i >= len(a.pre) {
			return -1
		}
		if i >= len(b.pre) {
			return 1
		}
		ap := a.pre[i]
		bp := b.pre[i]
		if ap.numeric && bp.numeric {
			if ap.numValue != bp.numValue {
				if ap.numValue > bp.numValue {
					return 1
				}
				return -1
			}
			continue
		}
		if ap.numeric != bp.numeric {
			if ap.numeric {
				return -1
			}
			return 1
		}
		if ap.lexicText == bp.lexicText {
			continue
		}
		if ap.lexicText > bp.lexicText {
			return 1
		}
		return -1
	}
	return 0
}

func isAllDigits(in string) bool {
	if in == "" {
		return false
	}
	for _, ch := range in {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}
