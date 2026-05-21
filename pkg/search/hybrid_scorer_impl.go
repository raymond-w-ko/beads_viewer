package search

import "fmt"

type hybridScorer struct {
	weights Weights
	cache   MetricsCache
}

// NewHybridScorer creates a scorer with the given weights and metrics cache.
func NewHybridScorer(weights Weights, cache MetricsCache) HybridScorer {
	normalized := defaultHybridWeights()
	if err := weights.validateComponents(); err == nil {
		normalized = weights.Normalize()
	}
	if err := normalized.Validate(); err != nil {
		normalized = defaultHybridWeights()
	}
	return &hybridScorer{
		weights: normalized,
		cache:   cache,
	}
}

func (s *hybridScorer) Score(issueID string, textScore float64) (HybridScore, error) {
	if issueID == "" {
		return HybridScore{}, fmt.Errorf("issueID is required")
	}

	if s.cache == nil {
		return HybridScore{
			IssueID:    issueID,
			FinalScore: textScore,
			TextScore:  textScore,
		}, nil
	}

	metrics, found := s.cache.Get(issueID)
	if !found {
		return HybridScore{
			IssueID:    issueID,
			FinalScore: textScore,
			TextScore:  textScore,
		}, nil
	}

	// Skip normalizations for zero-weight components to save computation
	var statusScore, priorityScore, impactScore, recencyScore float64
	if s.weights.Status > 0 {
		statusScore = normalizeStatus(metrics.Status)
	}
	if s.weights.Priority > 0 {
		priorityScore = normalizePriority(metrics.Priority)
	}
	if s.weights.Impact > 0 {
		impactScore = normalizeImpact(metrics.BlockerCount, s.cache.MaxBlockerCount())
	}
	if s.weights.Recency > 0 {
		recencyScore = normalizeRecency(metrics.UpdatedAt)
	}

	final := s.weights.TextRelevance*textScore +
		s.weights.PageRank*metrics.PageRank +
		s.weights.Status*statusScore +
		s.weights.Impact*impactScore +
		s.weights.Priority*priorityScore +
		s.weights.Recency*recencyScore

	return HybridScore{
		IssueID:    issueID,
		FinalScore: final,
		TextScore:  textScore,
		ComponentScores: map[string]float64{
			"pagerank": metrics.PageRank,
			"status":   statusScore,
			"impact":   impactScore,
			"priority": priorityScore,
			"recency":  recencyScore,
		},
	}, nil
}

func defaultHybridWeights() Weights {
	if preset, err := GetPreset(PresetDefault); err == nil {
		return preset
	}
	return Weights{TextRelevance: 1.0}
}

func (s *hybridScorer) Configure(weights Weights) error {
	// Validate raw weights first (catches negative values before normalization can mask them)
	if err := weights.Validate(); err != nil {
		return err
	}
	s.weights = weights.Normalize()
	return nil
}

func (s *hybridScorer) GetWeights() Weights {
	return s.weights
}
