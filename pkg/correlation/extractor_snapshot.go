package correlation

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"
)

// snapshotCommit holds the metadata and touched path for a single commit that
// modified the followed beads file.
type snapshotCommit struct {
	info commitInfo
	// path is the beads-file path *at this commit* (new name on a rename).
	path string
	// parentPath is the path the file had in the parent commit. It differs from
	// path only across a rename; empty when the file did not exist in the parent
	// (an addition).
	parentPath string
	// hadParent is false for the file's first appearance (no "-" side to diff).
	hadParent bool
}

// extractViaSnapshots reconstructs bead lifecycle events without asking git to
// produce a textual patch (`-p`) of the followed beads blob.
//
// Rationale (#160 follow-up / #161): `git log -p --follow -- <beads.jsonl>` runs
// git's Myers diff over the entire multi-MB JSONL blob at every commit, which is
// O(blob-size x commits) and dominates `--robot-triage` on large repos (measured
// ~13 min on a 93 MB blob across 200 commits). The parser only needs the set of
// added/removed `+{...}`/`-{...}` JSONL *record lines* per commit. Because the
// beads exporter writes one whole JSON record per line, that set is exactly the
// per-commit line-level set difference between the file's blob and its parent's
// blob — which we can compute from cheap blob reads (reading a blob is ~50x
// cheaper than diffing it) plus a hash-set comparison in Go. The synthesized
// per-commit diff is fed to the *unchanged* parseDiff, so event semantics
// (created/claimed/closed/reopened/modified) are identical to the `-p` path.
//
// The commit list, ordering, rename-following, and Since/Until/Limit filtering
// are obtained from a metadata-only `git log --follow --name-status` (no `-p`),
// which is effectively free.
func (e *Extractor) extractViaSnapshots(opts ExtractOptions) ([]BeadEvent, error) {
	commits, err := e.snapshotCommits(opts)
	if err != nil {
		return nil, err
	}
	if len(commits) == 0 {
		return nil, nil
	}

	// For each commit diff the followed file's blob at the commit against its
	// blob in the commit's first parent (the same two sides git's `-p --follow`
	// compares). Both blobs are batched through a single `git cat-file --batch`
	// so we pay one fork for the whole pass. We intentionally read the true
	// SHA^ blob rather than carrying the previous --follow entry forward: the
	// chronological predecessor in the --follow list is not always the commit's
	// first parent (merges / history rewrites), and using it produces spurious
	// events. Reading SHA^ keeps the output byte-identical to the legacy path
	// (verified by the differential test).
	specs := make([]string, 0, len(commits)*2)
	for _, c := range commits {
		specs = append(specs, c.info.SHA+":"+c.path)
		if c.hadParent {
			specs = append(specs, c.info.SHA+"^:"+c.parentPath)
		}
	}
	blobs, err := e.readBlobs(specs)
	if err != nil {
		return nil, err
	}

	var events []BeadEvent
	for i := range commits {
		c := commits[i]
		newSet := recordLineSet(blobs[c.info.SHA+":"+c.path])
		var oldSet map[string]int
		if c.hadParent {
			oldSet = recordLineSet(blobs[c.info.SHA+"^:"+c.parentPath])
		}

		diffText := synthesizeRecordDiffSets(oldSet, newSet)
		if len(diffText) > 0 {
			events = append(events, e.parseDiff(diffText, c.info, opts.BeadID)...)
		}
	}

	// snapshotCommits returns newest-first (git log order); match the legacy
	// Extract contract which sorts chronologically before returning.
	reverseEvents(events)
	return events, nil
}

// snapshotCommits returns, newest-first, the commits that touched the followed
// beads file together with the path the file had at the commit and at its
// parent. It uses a metadata-only `git log --follow --name-status` (no `-p`).
func (e *Extractor) snapshotCommits(opts ExtractOptions) ([]snapshotCommit, error) {
	primary := e.primaryBeadsFile()

	args := []string{
		"log",
		"--follow",
		"--name-status",
		"--format=" + gitLogHeaderFormat,
	}
	args = appendHistoryFilters(args, opts)
	args = append(args, "--", primary)

	cmd := exec.Command("git", withNoColorGit(args)...)
	cmd.Dir = e.repoPath
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("git log --name-status failed: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("git log --name-status failed: %w", err)
	}

	return parseSnapshotLog(out)
}

// appendHistoryFilters appends --since/--until/-n filters (the same ones the
// legacy buildGitLogArgs honored) before the pathspec separator.
func appendHistoryFilters(args []string, opts ExtractOptions) []string {
	if opts.Since != nil {
		args = append(args, "--since="+opts.Since.Format(time.RFC3339))
	}
	if opts.Until != nil {
		args = append(args, "--until="+opts.Until.Format(time.RFC3339))
	}
	if opts.Limit > 0 {
		args = append(args, fmt.Sprintf("-n%d", opts.Limit))
	}
	return args
}

// parseSnapshotLog parses the newline-delimited output of
// `git log --follow --name-status --format=<header>`.
//
// Per commit the stream is: a header line (NUL-separated %H..%s, terminated by
// '\n'), followed by tab-separated name-status line(s) such as "M\t<path>" or
// "R100\t<old>\t<new>", then a blank line before the next commit. We split the
// stream on the commit-header marker (commitPattern), reusing the same boundary
// detection as the streaming patch parser, then read the name-status lines that
// follow each header.
func parseSnapshotLog(out []byte) ([]snapshotCommit, error) {
	var commits []snapshotCommit

	locs := commitPattern.FindAllIndex(out, -1)
	for i, loc := range locs {
		start := loc[0]
		end := len(out)
		if i+1 < len(locs) {
			end = locs[i+1][0]
		}
		chunk := out[start:end]

		// The header line ends at the first '\n'; bytes before it are the
		// NUL-delimited %H..%s header.
		nl := bytes.IndexByte(chunk, '\n')
		if nl < 0 {
			continue
		}
		info, err := parseCommitInfo(string(chunk[:nl]))
		if err != nil {
			return nil, fmt.Errorf("parsing snapshot commit header: %w", err)
		}

		// Remaining lines are the tab-separated name-status entries. With
		// --follow against a single pathspec there is exactly one entry per
		// commit, but we scan defensively and take the first usable one.
		sc := snapshotCommit{info: info}
		if parseNameStatusLines(chunk[nl+1:], &sc) {
			commits = append(commits, sc)
		}
	}

	return commits, nil
}

// parseNameStatusLines reads the tab-separated name-status lines for one commit
// and fills sc with the status and path(s) for the followed file. Returns false
// when no usable name-status entry is present.
func parseNameStatusLines(payload []byte, sc *snapshotCommit) bool {
	scanner := bufio.NewScanner(bytes.NewReader(payload))
	scanner.Buffer(make([]byte, 64*1024), gitLogMaxScanTokenSize)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 2 {
			continue
		}
		status := parts[0]
		switch {
		case (strings.HasPrefix(status, "R") || strings.HasPrefix(status, "C")) && len(parts) >= 3:
			// Rename/copy: "R100"\t<old>\t<new>. Diff the new path's content
			// against the source path as it existed in the parent.
			sc.parentPath = parts[1]
			sc.path = parts[2]
			sc.hadParent = true
			return true
		case strings.HasPrefix(status, "A"):
			// Addition: file did not exist in the parent.
			sc.path = parts[1]
			sc.hadParent = false
			return true
		default:
			// Modify/Type-change/etc.: "M"\t<path>.
			sc.path = parts[1]
			sc.parentPath = parts[1]
			sc.hadParent = true
			return true
		}
	}
	return false
}

// readBlobs reads the requested <rev>:<path> blob specs via a single
// `git cat-file --batch` process and returns their contents keyed by spec.
// Missing blobs map to nil (treated as empty by the caller).
func (e *Extractor) readBlobs(specs []string) (map[string][]byte, error) {
	result := make(map[string][]byte, len(specs))

	// De-duplicate specs (the same blob can recur across adjacent commits).
	seen := make(map[string]struct{}, len(specs))
	unique := make([]string, 0, len(specs))
	for _, s := range specs {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		unique = append(unique, s)
	}
	sort.Strings(unique)

	cmd := exec.Command("git", "cat-file", "--batch")
	cmd.Dir = e.repoPath
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("git cat-file stdin: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("git cat-file stdout: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("starting git cat-file: %w", err)
	}

	// Writer goroutine: feed all specs then close stdin.
	writeErr := make(chan error, 1)
	go func() {
		w := bufio.NewWriter(stdin)
		for _, s := range unique {
			if _, err := w.WriteString(s + "\n"); err != nil {
				writeErr <- err
				_ = stdin.Close()
				return
			}
		}
		flushErr := w.Flush()
		closeErr := stdin.Close()
		if flushErr != nil {
			writeErr <- flushErr
			return
		}
		writeErr <- closeErr
	}()

	reader := bufio.NewReaderSize(stdout, gitLogMaxScanTokenSize)
	parseErr := func() error {
		for _, s := range unique {
			// Response header: "<sha> <type> <size>\n" or "<spec> missing\n".
			header, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("reading cat-file header for %q: %w", s, err)
			}
			header = strings.TrimRight(header, "\n")
			parts := strings.Fields(header)
			if len(parts) == 2 && parts[1] == "missing" {
				result[s] = nil
				continue
			}
			if len(parts) != 3 {
				return fmt.Errorf("unexpected cat-file header %q for %q", header, s)
			}
			var size int
			if _, err := fmt.Sscanf(parts[2], "%d", &size); err != nil {
				return fmt.Errorf("parsing cat-file size %q: %w", parts[2], err)
			}
			content := make([]byte, size)
			if _, err := readFull(reader, content); err != nil {
				return fmt.Errorf("reading cat-file content for %q: %w", s, err)
			}
			// Trailing newline after each object's content.
			if _, err := reader.Discard(1); err != nil {
				return fmt.Errorf("discarding cat-file trailer for %q: %w", s, err)
			}
			result[s] = content
		}
		return nil
	}()

	wErr := <-writeErr
	waitErr := cmd.Wait()
	if parseErr != nil {
		return nil, parseErr
	}
	if wErr != nil {
		return nil, fmt.Errorf("writing cat-file specs: %w", wErr)
	}
	if waitErr != nil {
		if exitErr, ok := waitErr.(*exec.ExitError); ok {
			return nil, fmt.Errorf("git cat-file failed: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("git cat-file failed: %w", waitErr)
	}
	return result, nil
}

// readFull reads exactly len(buf) bytes from r.
func readFull(r *bufio.Reader, buf []byte) (int, error) {
	total := 0
	for total < len(buf) {
		n, err := r.Read(buf[total:])
		total += n
		if err != nil {
			return total, err
		}
	}
	return total, nil
}

// synthesizeRecordDiff produces the same `+{...}`/`-{...}` record lines that
// `git log -p --unified=0` would emit for one commit, by computing the
// line-level set difference between the parent blob (old) and the commit blob
// (new). Beads exports one whole JSON record per line, so a record's presence in
// exactly one side marks it added/removed; a modified record appears as the old
// line removed and the new line added (identical to git's unified=0 output).
//
// Only record lines (those beginning with '{') are emitted, matching what the
// downstream parser consumes. Lines unchanged between the two blobs are omitted.
func synthesizeRecordDiff(oldBlob, newBlob []byte) []byte {
	return synthesizeRecordDiffSets(recordLineSet(oldBlob), recordLineSet(newBlob))
}

// synthesizeRecordDiffSets is synthesizeRecordDiff over already-hashed record
// multisets, so the carry-forward extraction loop hashes each blob only once.
// A nil oldSet means "no parent" — every record reads as an addition.
func synthesizeRecordDiffSets(oldSet, newSet map[string]int) []byte {
	var buf bytes.Buffer
	// Removed: present in old, absent (or fewer) in new.
	for line, n := range oldSet {
		for i := 0; i < n-newSet[line]; i++ {
			buf.WriteByte('-')
			buf.WriteString(line)
			buf.WriteByte('\n')
		}
	}
	// Added: present in new, absent (or fewer) in old.
	for line, n := range newSet {
		for i := 0; i < n-oldSet[line]; i++ {
			buf.WriteByte('+')
			buf.WriteString(line)
			buf.WriteByte('\n')
		}
	}
	return buf.Bytes()
}

// recordLineSet returns the multiset of JSON record lines (those starting with
// '{') in a blob. Non-record lines are ignored, matching parseDiff which only
// acts on lines beginning with '{'.
func recordLineSet(blob []byte) map[string]int {
	set := make(map[string]int)
	if len(blob) == 0 {
		return set
	}
	scanner := bufio.NewScanner(bytes.NewReader(blob))
	scanner.Buffer(make([]byte, 64*1024), gitLogMaxScanTokenSize)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) > 0 && line[0] == '{' {
			set[line]++
		}
	}
	return set
}
