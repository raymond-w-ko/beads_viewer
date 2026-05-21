package main

import (
	"strings"
	"testing"
)

func newTestPercentOrFraction(t *testing.T, flagName string, defaultPercent int) *percentOrFraction {
	t.Helper()
	p, err := newPercentOrFraction(flagName, defaultPercent)
	if err != nil {
		t.Fatalf("newPercentOrFraction(%q, %d) err = %v", flagName, defaultPercent, err)
	}
	return p
}

func TestPercentOrFraction_Default(t *testing.T) {
	p := newTestPercentOrFraction(t, "test", 20)
	if got := p.Value(); got != 20 {
		t.Fatalf("default Value() = %d, want 20", got)
	}
	if got := p.String(); got != "20" {
		t.Fatalf("default String() = %q, want %q", got, "20")
	}
}

func TestPercentOrFraction_AcceptsInt(t *testing.T) {
	p := newTestPercentOrFraction(t, "test", 20)
	if err := p.Set("50"); err != nil {
		t.Fatalf("Set(\"50\") err = %v", err)
	}
	if got := p.Value(); got != 50 {
		t.Fatalf("after Set(50), Value() = %d, want 50", got)
	}
}

func TestPercentOrFraction_AcceptsFraction(t *testing.T) {
	p := newTestPercentOrFraction(t, "test", 20)
	if err := p.Set("0.5"); err != nil {
		t.Fatalf("Set(\"0.5\") err = %v", err)
	}
	// 0.5 fraction becomes 50 percent (round half-up).
	if got := p.Value(); got != 50 {
		t.Fatalf("after Set(0.5), Value() = %d, want 50", got)
	}
}

func TestPercentOrFraction_FractionEdges(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"0.0", 0},
		{"1.0", 100},
		{"0.235", 24}, // round half-up: 23.5 becomes 24
		{"0.999", 100},
		{"0.001", 0},
	}
	for _, tc := range cases {
		p := newTestPercentOrFraction(t, "t", 20)
		if err := p.Set(tc.in); err != nil {
			t.Errorf("Set(%q) err = %v", tc.in, err)
			continue
		}
		if got := p.Value(); got != tc.want {
			t.Errorf("Set(%q) Value()=%d, want %d", tc.in, got, tc.want)
		}
	}
}

func TestPercentOrFraction_RejectsOutOfRangeInt(t *testing.T) {
	p := newTestPercentOrFraction(t, "test", 20)
	cases := []string{"-1", "101", "200", "1000"}
	for _, in := range cases {
		err := p.Set(in)
		if err == nil {
			t.Errorf("Set(%q) expected error, got nil", in)
			continue
		}
		// Error message must mention both forms so the user sees the alternative.
		msg := err.Error()
		if !strings.Contains(msg, "0-100") || !strings.Contains(msg, "0.0-1.0") {
			t.Errorf("Set(%q) err = %q; expected mention of both 0-100 and 0.0-1.0", in, msg)
		}
	}
}

func TestPercentOrFraction_RejectsOutOfRangeFraction(t *testing.T) {
	p := newTestPercentOrFraction(t, "test", 20)
	cases := []string{"-0.1", "1.1", "2.0", "-0.5"}
	for _, in := range cases {
		err := p.Set(in)
		if err == nil {
			t.Errorf("Set(%q) expected error, got nil", in)
			continue
		}
		msg := err.Error()
		if !strings.Contains(msg, "0.0-1.0") {
			t.Errorf("Set(%q) err = %q; expected mention of 0.0-1.0", in, msg)
		}
	}
}

func TestPercentOrFraction_RejectsNonNumeric(t *testing.T) {
	p := newTestPercentOrFraction(t, "test", 20)
	cases := []string{"abc", "0.5x", "1e2y", ""}
	for _, in := range cases {
		err := p.Set(in)
		if err == nil {
			t.Errorf("Set(%q) expected error, got nil", in)
		}
	}
}

func TestPercentOrFraction_ErrorMessageNamesFlag(t *testing.T) {
	p := newTestPercentOrFraction(t, "related-min-relevance", 20)
	err := p.Set("foo")
	if err == nil {
		t.Fatalf("expected error for 'foo'")
	}
	if !strings.Contains(err.Error(), "related-min-relevance") {
		t.Errorf("error %q should name the flag", err.Error())
	}
}

func TestPercentOrFraction_TypeForHelpRendering(t *testing.T) {
	p := newTestPercentOrFraction(t, "test", 20)
	if got := p.Type(); got != "percent_or_fraction" {
		t.Errorf("Type() = %q, want percent_or_fraction", got)
	}
}

func TestPercentOrFraction_RejectsInvalidDefault(t *testing.T) {
	for _, defaultPercent := range []int{-1, 101} {
		if _, err := newPercentOrFraction("test", defaultPercent); err == nil {
			t.Fatalf("newPercentOrFraction default %d expected error, got nil", defaultPercent)
		}
	}
}
