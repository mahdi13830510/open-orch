package normalizer

import (
	"regexp"
	"strings"
)

// MaxLen caps a normalized feature slug. DNS labels are 63 chars max; we also
// allow it to fit inside a hostname like  "<slug>-pr<num>.<base-domain>".
const MaxLen = 40

// Known git prefixes we strip when deriving a feature identity from a branch.
var prefixes = []string{
	"feat/", "feature/", "fix/", "bug/", "bugfix/", "chore/",
	"hotfix/", "release/", "refactor/", "docs/", "test/", "ci/", "perf/",
}

// invalidChars: anything that isn't [a-z0-9-] gets squashed to a hyphen.
var invalidChars = regexp.MustCompile(`[^a-z0-9-]+`)
var multiHyphen = regexp.MustCompile(`-+`)

// Branch turns a raw git branch into a DNS-safe feature slug.
//
// Rules:
//   - lowercase
//   - strip well-known git prefixes
//   - replace slashes with hyphens
//   - replace any invalid DNS char with hyphen
//   - collapse runs of hyphens
//   - trim leading/trailing hyphens
//   - enforce max length (truncate cleanly at hyphen if possible)
func Branch(raw string) string {
	s := strings.ToLower(strings.TrimSpace(raw))
	if s == "" {
		return ""
	}

	// Strip known prefixes (only the first one we hit).
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			s = strings.TrimPrefix(s, p)
			break
		}
	}

	// Slashes become hyphens.
	s = strings.ReplaceAll(s, "/", "-")

	// Squash anything not DNS-safe.
	s = invalidChars.ReplaceAllString(s, "-")

	// Collapse repeated hyphens.
	s = multiHyphen.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")

	// Enforce length; prefer cutting at a hyphen.
	if len(s) > MaxLen {
		cut := s[:MaxLen]
		if i := strings.LastIndex(cut, "-"); i > MaxLen/2 {
			cut = cut[:i]
		}
		s = strings.Trim(cut, "-")
	}
	return s
}

// HostnameLabel makes a string safe to use as a hostname label, with the
// same general rules. Useful for "<feature>-pr<num>" labels too.
func HostnameLabel(parts ...string) string {
	joined := strings.Join(parts, "-")
	joined = strings.ToLower(joined)
	joined = invalidChars.ReplaceAllString(joined, "-")
	joined = multiHyphen.ReplaceAllString(joined, "-")
	joined = strings.Trim(joined, "-")
	if len(joined) > 63 {
		joined = strings.Trim(joined[:63], "-")
	}
	return joined
}
