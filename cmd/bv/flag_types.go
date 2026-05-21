package main

import (
	"fmt"
	"strconv"
	"strings"
)

// percentOrFraction is a pflag.Value that accepts either:
//   - an integer 0-100 (interpreted as percent), e.g. "20"
//   - a float 0.0-1.0 (interpreted as fraction), e.g. "0.20"
//
// Internally stored as int 0-100 (percent) so existing consumers reading the
// canonical integer downstream are unchanged.
//
// Why: prior to this change, `--related-min-relevance` accepted only int
// 0-100, while sibling `--relations-threshold` accepted only float 0.0-1.0.
// An agent or user inferring a uniform convention from one flag would silently
// fail on the other (e.g. "0.5" parsed as int "0"). Accepting both forms with
// a structured error mentioning each removes that footgun.
type percentOrFraction struct {
	val int // canonical: 0-100 percent
	// flagName is the user-facing flag (e.g. "related-min-relevance"); used
	// only in error messages so they reference the exact flag the user typed.
	flagName string
}

// newPercentOrFraction constructs a flag value with a percent-int default.
// Both `defaultPercent` and any user-supplied value go through the same
// validation; an out-of-range default is reported to the caller.
func newPercentOrFraction(flagName string, defaultPercent int) (*percentOrFraction, error) {
	if defaultPercent < 0 || defaultPercent > 100 {
		return nil, fmt.Errorf("percentOrFraction: default %d out of range (expected 0-100) for --%s", defaultPercent, flagName)
	}
	return &percentOrFraction{
		val:      defaultPercent,
		flagName: flagName,
	}, nil
}

// Value returns the int 0-100 percent representation. Use this, not the
// String() representation, for downstream consumers expecting an int.
func (p *percentOrFraction) Value() int {
	if p == nil {
		return 0
	}
	return p.val
}

// String returns the canonical percent-int string. pflag uses this for
// --help rendering, so the rendered default looks like a familiar integer.
func (p *percentOrFraction) String() string {
	if p == nil {
		return ""
	}
	return strconv.Itoa(p.val)
}

// Type satisfies pflag.Value's Type() method; affects --help formatting.
func (p *percentOrFraction) Type() string {
	return "percent_or_fraction"
}

// Set parses the user-supplied string. Tries float-with-decimal-point first,
// falls back to int. Both forms are validated against their respective range.
func (p *percentOrFraction) Set(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return fmt.Errorf("--%s: empty value (expected int 0-100 or float 0.0-1.0)", p.flagName)
	}
	// A "." anywhere in the input signals fractional intent; this is the
	// reliable disambiguator. "0" is unambiguously percent (= fraction 0.00),
	// "100" is unambiguously percent (= fraction 1.0), "50" is percent 50%.
	if strings.ContainsRune(s, '.') {
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return fmt.Errorf("--%s: %q is not a number (expected int 0-100 percent OR float 0.0-1.0 fraction)", p.flagName, s)
		}
		if f < 0.0 || f > 1.0 {
			return fmt.Errorf("--%s: float %g out of range (expected 0.0-1.0 fraction; for percent use int 0-100)", p.flagName, f)
		}
		// Round to nearest int percent so 0.235 becomes 24%, 0.999 becomes 100%.
		p.val = int(f*100 + 0.5)
		return nil
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return fmt.Errorf("--%s: %q is not an integer (expected int 0-100 percent OR float 0.0-1.0 fraction)", p.flagName, s)
	}
	if n < 0 || n > 100 {
		return fmt.Errorf("--%s: int %d out of range (expected 0-100 percent; for fraction use float 0.0-1.0)", p.flagName, n)
	}
	p.val = n
	return nil
}
