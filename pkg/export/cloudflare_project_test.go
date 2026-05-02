package export

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"testing"
)

// These tests exercise CloudflareProjectExists, CreateCloudflareProject, and
// EnsureCloudflareProject via a stub wrangler on PATH. They were previously
// at 0% coverage because the only other reference to them is the opt-in
// TestEnsureCloudflareProject_Integration, which skips whenever wrangler is
// not installed (i.e. always in CI).

func TestCloudflareProjectExists_WranglerStub(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script stubs not supported on windows in this test")
	}

	cases := []struct {
		name         string
		stdout       string
		exit         int
		projectName  string
		wantExists   bool
		wantErrMatch string
	}{
		{
			name: "project present",
			stdout: `Name           Created
my-project     2026-01-01
other-project  2026-02-01
`,
			projectName: "my-project",
			wantExists:  true,
		},
		{
			name: "project absent from populated list",
			stdout: `Name           Created
other-project  2026-02-01
`,
			projectName: "my-project",
			wantExists:  false,
		},
		{
			name:        "empty list",
			stdout:      "",
			projectName: "my-project",
			wantExists:  false,
		},
		{
			name:         "wrangler command fails",
			stdout:       "",
			exit:         1,
			projectName:  "my-project",
			wantErrMatch: "failed to list projects",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			binDir := t.TempDir()
			script := fmt.Sprintf(`#!/bin/sh
cat <<'STDOUT_EOF'
%sSTDOUT_EOF
exit %d
`, tc.stdout, tc.exit)
			writeExecutable(t, binDir, "wrangler", script)

			origPath := os.Getenv("PATH")
			t.Setenv("PATH", fmt.Sprintf("%s%c%s", binDir, os.PathListSeparator, origPath))

			exists, err := CloudflareProjectExists(tc.projectName)
			if tc.wantErrMatch != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil (exists=%v)", tc.wantErrMatch, exists)
				}
				if !strings.Contains(strings.ToLower(err.Error()), tc.wantErrMatch) {
					t.Fatalf("error %q does not contain %q", err.Error(), tc.wantErrMatch)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if exists != tc.wantExists {
				t.Fatalf("exists = %v, want %v", exists, tc.wantExists)
			}
		})
	}
}

func TestCreateCloudflareProject_WranglerStub(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script stubs not supported on windows in this test")
	}

	cases := []struct {
		name         string
		stdout       string
		exit         int
		projectName  string
		branch       string
		wantErrMatch string // "" means success expected
	}{
		{
			name:        "project created",
			stdout:      "Successfully created project",
			projectName: "fresh-project",
			branch:      "main",
		},
		{
			name:        "empty branch defaults to main",
			stdout:      "Successfully created project",
			projectName: "fresh-project",
			branch:      "",
		},
		{
			name:        "already exists (treated as success)",
			stdout:      "A project with this name already exists",
			exit:        1,
			projectName: "existing-project",
			branch:      "main",
		},
		{
			name:         "generic wrangler failure",
			stdout:       "something went wrong",
			exit:         1,
			projectName:  "doomed-project",
			branch:       "main",
			wantErrMatch: "failed to create project",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			binDir := t.TempDir()
			script := fmt.Sprintf(`#!/bin/sh
cat <<'STDOUT_EOF'
%s
STDOUT_EOF
exit %d
`, tc.stdout, tc.exit)
			writeExecutable(t, binDir, "wrangler", script)

			origPath := os.Getenv("PATH")
			t.Setenv("PATH", fmt.Sprintf("%s%c%s", binDir, os.PathListSeparator, origPath))

			err := CreateCloudflareProject(tc.projectName, tc.branch)
			if tc.wantErrMatch != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tc.wantErrMatch)
				}
				if !strings.Contains(strings.ToLower(err.Error()), tc.wantErrMatch) {
					t.Fatalf("error %q does not contain %q", err.Error(), tc.wantErrMatch)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestEnsureCloudflareProject_WranglerStub(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script stubs not supported on windows in this test")
	}

	// One wrangler stub that dispatches on argv[1..2]: "pages project list"
	// vs "pages project create". State file records each create attempt so
	// the test can assert Ensure's create-when-missing semantics without
	// peering into unrelated globals.
	setup := func(t *testing.T, existingProjects []string) string {
		t.Helper()
		binDir := t.TempDir()
		stateDir := t.TempDir()

		// Build the static project-list output once.
		var listOut strings.Builder
		listOut.WriteString("Name        Created\n")
		for _, p := range existingProjects {
			listOut.WriteString(p)
			listOut.WriteString("  2026-01-01\n")
		}

		script := fmt.Sprintf(`#!/bin/sh
set -eu
state_dir=%q
if [ "$1" = "pages" ] && [ "$2" = "project" ] && [ "$3" = "list" ]; then
cat <<'LIST_EOF'
%sLIST_EOF
  exit 0
fi
if [ "$1" = "pages" ] && [ "$2" = "project" ] && [ "$3" = "create" ]; then
  echo "$4" >> "$state_dir/creates.log"
  echo "Successfully created $4"
  exit 0
fi
echo "unhandled: $*" >&2
exit 2
`, stateDir, listOut.String())
		writeExecutable(t, binDir, "wrangler", script)

		origPath := os.Getenv("PATH")
		t.Setenv("PATH", fmt.Sprintf("%s%c%s", binDir, os.PathListSeparator, origPath))
		return stateDir
	}

	t.Run("no-op when project exists", func(t *testing.T) {
		stateDir := setup(t, []string{"my-project", "another"})

		if err := EnsureCloudflareProject("my-project", "main"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, err := os.Stat(stateDir + "/creates.log"); err == nil {
			t.Fatal("EnsureCloudflareProject attempted a create when project already existed")
		}
	})

	t.Run("creates when missing", func(t *testing.T) {
		stateDir := setup(t, []string{"another"})

		if err := EnsureCloudflareProject("my-project", "trunk"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		data, err := os.ReadFile(stateDir + "/creates.log")
		if err != nil {
			t.Fatalf("expected creates.log, got: %v", err)
		}
		if got := strings.TrimSpace(string(data)); got != "my-project" {
			t.Fatalf("created project = %q, want %q", got, "my-project")
		}
	})
}
