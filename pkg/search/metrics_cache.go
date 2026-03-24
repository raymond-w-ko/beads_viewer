package search

import "time"

// IssueMetrics contains the graph-derived metrics for hybrid scoring.
type IssueMetrics struct {
	IssueID      string    `json:"issue_id"`
	PageRank     float64   `json:"pagerank"`      // 0.0-1.0, from graph analysis
	Status       string    `json:"status"`        // open|in_progress|blocked|closed
	Priority     int       `json:"priority"`      // 0-4 (P0=0, P4=4)
	BlockerCount int       `json:"blocker_count"` // How many issues this blocks
	UpdatedAt    time.Time `json:"updated_at"`    // For recency calculation
}

// MetricsCache provides fast access to issue metrics for hybrid scoring.
type MetricsCache interface {
	// Get returns metrics for an issue, computing/loading if needed.
	Get(issueID string) (IssueMetrics, bool)

	// GetBatch returns metrics for multiple issues efficiently.
	GetBatch(issueIDs []string) map[string]IssueMetrics

	// Refresh recomputes the cache from source data.
	Refresh() error

	// DataHash returns the hash of source data for cache validation.
	DataHash() string

	// MaxBlockerCount returns the maximum blocker count for normalization.
	MaxBlockerCount() int
}

// MetricsLoader abstracts the source of metrics (graph analysis or direct DB).
type MetricsLoader interface {
	LoadMetrics() (map[string]IssueMetrics, error)
	ComputeDataHash() (string, error)
}

// MetricsLoaderAtomic extends MetricsLoader with atomic load-and-hash capability.
// Loaders implementing this interface can return metrics and their data hash together,
// preventing TOCTOU race conditions between loading metrics and computing hash.
type MetricsLoaderAtomic interface {
	MetricsLoader
	// LoadMetricsWithHash returns metrics and their corresponding data hash atomically.
	// The hash is computed from the same data used to generate the metrics, ensuring
	// consistency even if the underlying data changes between calls.
	LoadMetricsWithHash() (map[string]IssueMetrics, string, error)
}
