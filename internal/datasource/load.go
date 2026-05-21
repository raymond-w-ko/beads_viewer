package datasource

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Dicklesworthstone/beads_viewer/pkg/loader"
	"github.com/Dicklesworthstone/beads_viewer/pkg/model"
)

// LoadIssues performs smart multi-source detection and loading.
// It discovers all available sources (SQLite, JSONL), validates them, selects
// the freshest valid source, and loads issues from it. SQLite is preferred over
// JSONL when both exist at comparable freshness, since SQLite reflects the most
// recent state (including status changes from br operations).
//
// Falls back to legacy JSONL-only loading via loader.LoadIssues if smart
// detection finds no valid sources.
func LoadIssues(repoPath string) ([]model.Issue, error) {
	if source, ok, err := ExplicitBeadsDBSource(); err != nil {
		return nil, err
	} else if ok {
		return LoadFromSource(source)
	}

	beadsDir, err := loader.GetBeadsDir(repoPath)
	if err != nil {
		return nil, err
	}

	issues, smartErr := loadSmart(beadsDir, repoPath)
	if smartErr == nil {
		return issues, nil
	}

	// Fall back to legacy JSONL-only loading
	return loader.LoadIssues(repoPath)
}

// LoadIssuesFromDir performs smart source detection within a known beads directory.
// This is useful when the caller already knows the .beads path.
func LoadIssuesFromDir(beadsDir string) ([]model.Issue, error) {
	issues, smartErr := loadSmart(beadsDir, "")
	if smartErr == nil {
		return issues, nil
	}

	// Fall back to JSONL
	jsonlPath, err := loader.FindJSONLPath(beadsDir)
	if err != nil {
		return nil, err
	}
	return loader.LoadIssuesFromFile(jsonlPath)
}

// ExplicitBeadsDBSource returns the direct source named by BEADS_DB when it
// points at a concrete source file. Directory values return ok=false so callers
// can use normal source discovery within that directory.
func ExplicitBeadsDBSource() (DataSource, bool, error) {
	return SourceFromFile(os.Getenv(loader.BeadsDBEnvVar))
}

// SourceFromFile returns a DataSource for a concrete source file path. Directory
// paths and empty values return ok=false so callers can fall back to discovery.
func SourceFromFile(dbPath string) (DataSource, bool, error) {
	if strings.TrimSpace(dbPath) == "" {
		return DataSource{}, false, nil
	}

	info, err := os.Stat(dbPath)
	if err == nil && info.IsDir() {
		return DataSource{}, false, nil
	}
	if err != nil && !os.IsNotExist(err) {
		return DataSource{}, true, fmt.Errorf("stat %s: %w", dbPath, err)
	}

	sourceType, priority, ok := explicitBeadsDBFileType(dbPath)
	if !ok {
		if err == nil {
			return DataSource{}, true, fmt.Errorf("unsupported BEADS_DB file type: %s", dbPath)
		}
		return DataSource{}, false, nil
	}

	source := DataSource{
		Type:     sourceType,
		Path:     dbPath,
		Priority: priority,
	}
	if err == nil {
		source.ModTime = info.ModTime()
		source.Size = info.Size()
	}
	return source, true, nil
}

func explicitBeadsDBFileType(dbPath string) (SourceType, int, bool) {
	switch strings.ToLower(filepath.Ext(dbPath)) {
	case ".db", ".sqlite", ".sqlite3":
		return SourceTypeSQLite, PrioritySQLite, true
	case ".jsonl":
		return SourceTypeJSONLLocal, PriorityJSONLLocal, true
	default:
		return "", 0, false
	}
}

// loadSmart discovers sources, validates, selects the best, and loads from it.
func loadSmart(beadsDir, repoPath string) ([]model.Issue, error) {
	sources, err := DiscoverSources(DiscoveryOptions{
		BeadsDir:               beadsDir,
		RepoPath:               repoPath,
		ValidateAfterDiscovery: true,
		IncludeInvalid:         false,
	})
	if err != nil {
		return nil, err
	}
	if len(sources) == 0 {
		return nil, fmt.Errorf("no valid sources discovered")
	}

	best, err := SelectBestSource(sources)
	if err != nil {
		return nil, err
	}

	return LoadFromSource(best)
}

// LoadFromSource loads issues from a specific DataSource via the IssueReader
// interface, dispatching to the appropriate backend based on source type.
func LoadFromSource(source DataSource) ([]model.Issue, error) {
	reader, err := NewReader(source)
	if err != nil {
		return nil, fmt.Errorf("failed to open source %s: %w", source.Path, err)
	}
	defer reader.Close()
	return reader.LoadIssues()
}
