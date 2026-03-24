package xfetch

import (
	"math"
	"testing"
	"time"
)

func TestXFetchShouldRefresh_ZeroDuration(t *testing.T) {
	now := time.Now()
	lastCompute := now.Add(-time.Second)

	// Zero duration should never trigger refresh
	if ShouldRefresh(lastCompute, 0, 1.0, now) {
		t.Error("expected no refresh with zero duration")
	}

	// Negative duration should never trigger refresh
	if ShouldRefresh(lastCompute, -time.Second, 1.0, now) {
		t.Error("expected no refresh with negative duration")
	}
}

func TestXFetchShouldRefresh_InvalidBeta(t *testing.T) {
	now := time.Now()
	lastCompute := now.Add(-time.Hour)
	computeDuration := time.Minute

	tests := []struct {
		name string
		beta float64
	}{
		{name: "zero", beta: 0},
		{name: "negative", beta: -1},
		{name: "nan", beta: math.NaN()},
		{name: "positive inf", beta: math.Inf(1)},
		{name: "negative inf", beta: math.Inf(-1)},
	}

	for _, tt := range tests {
		if ShouldRefresh(lastCompute, computeDuration, tt.beta, now) {
			t.Fatalf("expected no refresh for invalid beta %s", tt.name)
		}
	}
}

func TestXFetchShouldRefresh_Deterministic(t *testing.T) {
	// With a fixed "now" far in the future, should always refresh
	lastCompute := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	computeDuration := time.Hour
	farFuture := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)

	// Run multiple times - all should trigger refresh since now >> lastCompute + duration
	refreshCount := 0
	for i := 0; i < 100; i++ {
		if ShouldRefresh(lastCompute, computeDuration, 1.0, farFuture) {
			refreshCount++
		}
	}

	// All should refresh since we're 10 years past the compute time
	if refreshCount < 95 {
		t.Errorf("expected nearly all refreshes for far future time, got %d/100", refreshCount)
	}
}

func TestXFetchShouldRefresh_NeverRefreshImmediately(t *testing.T) {
	// Immediately after compute, should almost never refresh
	now := time.Now()
	lastCompute := now
	computeDuration := time.Hour

	// Run multiple times
	refreshCount := 0
	for i := 0; i < 100; i++ {
		if ShouldRefresh(lastCompute, computeDuration, 1.0, now) {
			refreshCount++
		}
	}

	// Almost none should refresh immediately (random chance is very low)
	// The threshold is duration * beta * -ln(rand), which is always positive
	// So now must be > lastCompute + positive_threshold, which is impossible when now == lastCompute
	if refreshCount > 10 {
		t.Errorf("expected almost no immediate refreshes, got %d/100", refreshCount)
	}
}

func TestXFetchShouldRefresh_ProbabilisticDistribution(t *testing.T) {
	// Test that refresh probability increases as time passes
	// At time = lastCompute + computeDuration, probability should be around 63% (1 - 1/e)
	// because P(refresh) = P(-ln(rand) < 1) = P(rand > 1/e) = 1 - 1/e ≈ 0.632

	computeDuration := time.Hour
	lastCompute := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// Test at t = lastCompute + computeDuration (elapsed = 1.0 * duration)
	testTime := lastCompute.Add(computeDuration)

	const samples = 1000
	refreshCount := 0
	for i := 0; i < samples; i++ {
		if ShouldRefresh(lastCompute, computeDuration, 1.0, testTime) {
			refreshCount++
		}
	}

	// Expected: ~63.2% (1 - 1/e)
	// With 1000 samples, we expect 632 ± ~30 (3 standard deviations)
	// Using wider bounds for test stability: 550-720
	expectedCenter := 0.632 * float64(samples)
	tolerance := 0.1 * float64(samples) // 10% tolerance

	if math.Abs(float64(refreshCount)-expectedCenter) > tolerance {
		t.Errorf("at t=duration, expected ~%.0f refreshes (1-1/e), got %d", expectedCenter, refreshCount)
	}
}

func TestXFetchShouldRefresh_BetaScaling(t *testing.T) {
	// Higher beta should mean less aggressive refresh (higher threshold)
	computeDuration := time.Hour
	lastCompute := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	testTime := lastCompute.Add(computeDuration)

	const samples = 500

	// Beta = 0.5 (more aggressive)
	refreshLowBeta := 0
	for i := 0; i < samples; i++ {
		if ShouldRefresh(lastCompute, computeDuration, 0.5, testTime) {
			refreshLowBeta++
		}
	}

	// Beta = 2.0 (less aggressive)
	refreshHighBeta := 0
	for i := 0; i < samples; i++ {
		if ShouldRefresh(lastCompute, computeDuration, 2.0, testTime) {
			refreshHighBeta++
		}
	}

	// Lower beta should produce more refreshes
	if refreshHighBeta >= refreshLowBeta {
		t.Errorf("expected lower beta to produce more refreshes: beta=0.5 got %d, beta=2.0 got %d",
			refreshLowBeta, refreshHighBeta)
	}
}

func TestShouldRefreshWithDefault(t *testing.T) {
	// Just verify the wrapper works
	// Use a time far in the past to guarantee refresh
	oldTime := time.Now().Add(-365 * 24 * time.Hour)
	computeDuration := time.Minute

	// Should almost certainly refresh
	refreshCount := 0
	for i := 0; i < 10; i++ {
		if ShouldRefreshWithDefault(oldTime, computeDuration) {
			refreshCount++
		}
	}

	if refreshCount < 8 {
		t.Errorf("expected most calls to refresh for old cache, got %d/10", refreshCount)
	}
}
