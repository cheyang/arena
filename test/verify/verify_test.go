// Package verify holds a reviewer-private verification harness for the review of
// kubeflow/arena PRs #1488, #1489 and #1490.
//
// It is additive: no production code is modified. Each test is grafted onto a PR
// head by docs/verification/pr-1488-1490/scripts/re-verify.sh and skips itself
// when the file it inspects is absent from that tree (the three PRs sit on two
// different bases, so not every target exists on every ref).
//
// Polarity is recorded in each test's doc comment and in verify-manifest.json:
//
//	CONTRACT   — asserts intended behavior. RED on the code under review, GREEN once fixed.
//	REGRESSION — asserts behavior that is already correct. GREEN now, must stay GREEN.
package verify

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/kubeflow/arena/pkg/task"
)

// repoRoot resolves the repository root relative to test/verify/.
func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	return filepath.Join(wd, "..", "..")
}

// readOrSkip reads a repo-relative file, skipping the test when the ref under
// verification does not contain it.
func readOrSkip(t *testing.T, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(repoRoot(t), rel))
	if err != nil {
		t.Skipf("%s not present on this ref (%v) — not applicable here", rel, err)
	}
	return string(b)
}

// ---------------------------------------------------------------------------
// F1490-1  CONTRACT  (carry-over of PR1483-F1)
// ---------------------------------------------------------------------------

// TestVerify_F1490_1_E2EBuildTag asserts every Go file under test/e2e/ carries a
// `//go:build v2e2e` constraint, so that a plain `go test ./...` does not drag the
// cluster-dependent Ginkgo suite in and fail at BeforeSuite.
//
// CONTRACT: RED while the tag is missing. Goes GREEN when each test/e2e file gets
// `//go:build v2e2e` as its first line (the Makefile already passes -tags v2e2e).
func TestVerify_F1490_1_E2EBuildTag(t *testing.T) {
	dir := filepath.Join(repoRoot(t), "test", "e2e")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Skipf("test/e2e not present on this ref: %v", err)
	}

	var untagged []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatalf("read %s: %v", e.Name(), err)
		}
		if !regexp.MustCompile(`(?m)^//go:build\s+.*\bv2e2e\b`).Match(b) {
			untagged = append(untagged, e.Name())
		}
	}
	if len(untagged) > 0 {
		t.Errorf("F1490-1: %d file(s) under test/e2e lack a //go:build v2e2e constraint: %v\n"+
			"  effect: `go test ./...` compiles and runs the e2e suite and fails in BeforeSuite\n"+
			"  (suite_test.go Fail: \"training-operator not found\" / \"arena-v2 binary not found\").\n"+
			"  CI only escapes this because `make v2-test` enumerates V2_PACKAGES instead of ./... .",
			len(untagged), untagged)
	}
}

// TestVerify_F1490_1_MakefileTagContract records the other half of the contract:
// if the Makefile passes -tags v2e2e, at least one file must match that tag,
// otherwise the flag is a silent no-op.
//
// CONTRACT: RED whenever the Makefile advertises the tag but nothing declares it.
func TestVerify_F1490_1_MakefileTagContract(t *testing.T) {
	mk := readOrSkip(t, "Makefile")
	if !strings.Contains(mk, "v2e2e") {
		t.Skip("Makefile does not reference the v2e2e tag on this ref")
	}
	matches, _ := filepath.Glob(filepath.Join(repoRoot(t), "test", "e2e", "*.go"))
	for _, m := range matches {
		b, err := os.ReadFile(m)
		if err != nil {
			continue
		}
		if regexp.MustCompile(`(?m)^//go:build\s+.*\bv2e2e\b`).Match(b) {
			return // tag is honored by at least one file
		}
	}
	t.Error("F1490-1: Makefile references the v2e2e build tag but no file under test/e2e declares it — the flag is a no-op")
}

// ---------------------------------------------------------------------------
// F1489-1  CONTRACT
// ---------------------------------------------------------------------------

// shmDocSnippet returns the fenced YAML block from the shared-memory section of
// docs/best-practices.md.
func shmDocSnippet(t *testing.T, doc string) string {
	t.Helper()
	idx := strings.Index(doc, "### Use shared memory")
	if idx < 0 {
		t.Skip("shared-memory section not found in docs/best-practices.md on this ref")
	}
	rest := doc[idx:]
	re := regexp.MustCompile("(?s)```yaml\n(.*?)```")
	m := re.FindStringSubmatch(rest)
	if m == nil {
		t.Skip("no yaml fence under the shared-memory section")
	}
	return m[1]
}

// wrapTask embeds a `storages:`-only doc fragment into an otherwise minimal,
// valid task so it can be round-tripped through task.LoadFromFile.
func wrapTask(t *testing.T, fragment string) string {
	t.Helper()
	doc := "version: 0.1.0\nname: shm-doc-check\nimage: busybox\nframework:\n  name: pytorch\nrun: sleep 1\nworker:\n  replicas: 1\n" +
		fragment + "mounts:\n  - name: shm\n"
	f, err := os.CreateTemp(t.TempDir(), "shm-*.yaml")
	if err != nil {
		t.Fatalf("temp: %v", err)
	}
	if _, err := f.WriteString(doc); err != nil {
		t.Fatalf("write: %v", err)
	}
	f.Close()
	return f.Name()
}

// TestVerify_F1489_1_ShmOptionalFieldsHonored checks the doc's own promise: every
// field the shared-memory snippet annotates as "Optional" must actually be
// omittable. For each such field the test removes that line and loads the task.
//
// CONTRACT: RED while docs/best-practices.md marks mount_path and shm "Optional"
// but task.Storage.Validate (pkg/task/types.go) rejects both omissions. Goes GREEN
// either by dropping the misleading annotations from the doc, or by relaxing
// Validate so pkg/provider/interface.go's SHM mount-path defaulting is reachable.
func TestVerify_F1489_1_ShmOptionalFieldsHonored(t *testing.T) {
	doc := readOrSkip(t, "docs/best-practices.md")
	snippet := shmDocSnippet(t, doc)

	lines := strings.Split(snippet, "\n")
	type optField struct {
		name string
		idx  int
	}
	var optionals []optField
	fieldRe := regexp.MustCompile(`^\s*([a-z_]+):`)
	for i, ln := range lines {
		if !strings.Contains(ln, "Optional") {
			continue
		}
		if m := fieldRe.FindStringSubmatch(ln); m != nil {
			optionals = append(optionals, optField{name: m[1], idx: i})
		}
	}
	if len(optionals) == 0 {
		t.Log("doc no longer annotates any shm field as Optional — nothing to check")
		return
	}

	for _, of := range optionals {
		t.Run("omit_"+of.name, func(t *testing.T) {
			kept := make([]string, 0, len(lines))
			for i, ln := range lines {
				if i != of.idx {
					kept = append(kept, ln)
				}
			}
			path := wrapTask(t, strings.Join(kept, "\n"))
			if _, err := task.LoadFromFile(path); err != nil {
				t.Errorf("F1489-1: docs/best-practices.md marks %q Optional, but omitting it fails to load:\n  %v", of.name, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// F1489-3  CONTRACT
// ---------------------------------------------------------------------------

// TestVerify_F1489_3_NoJobSubmitInDocs asserts the docs never spell the submit
// command as `arena job submit`. submitCmd is registered on the root command
// (pkg/cli/submit.go: rootCmd.AddCommand(submitCmd)), never on jobCmd, so
// `arena-v2 job submit pytorch --name x` fails with "unknown flag: --name".
//
// CONTRACT: RED while docs/cli-reference.md documents the nested form.
func TestVerify_F1489_3_NoJobSubmitInDocs(t *testing.T) {
	root := repoRoot(t)
	docsDir := filepath.Join(root, "docs")
	if _, err := os.Stat(docsDir); err != nil {
		t.Skipf("docs/ not present on this ref: %v", err)
	}

	// Confirm the premise from the code before judging the docs.
	if src, err := os.ReadFile(filepath.Join(root, "pkg", "cli", "submit.go")); err == nil {
		if !strings.Contains(string(src), "rootCmd.AddCommand(submitCmd)") {
			t.Skip("submitCmd is no longer registered on rootCmd — re-derive this finding")
		}
	}

	re := regexp.MustCompile(`arena(-v2)? job submit\b`)
	var hits []string
	err := filepath.Walk(docsDir, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(p, ".md") {
			return nil
		}
		b, rerr := os.ReadFile(p)
		if rerr != nil {
			return nil
		}
		for i, ln := range strings.Split(string(b), "\n") {
			if re.MatchString(ln) {
				rel, _ := filepath.Rel(root, p)
				hits = append(hits, rel+":"+itoa(i+1))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk docs: %v", err)
	}
	if len(hits) > 0 {
		t.Errorf("F1489-3: docs document `arena job submit`, which is not a registered command (submit lives on the root): %v", hits)
	}
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b []byte
	for i > 0 {
		b = append([]byte{byte('0' + i%10)}, b...)
		i /= 10
	}
	return string(b)
}

// ---------------------------------------------------------------------------
// F1489-R  REGRESSION (green now — must stay green)
// ---------------------------------------------------------------------------

// TestVerify_F1489_R_AllExamplesLoad walks examples/v2 recursively and loads every
// YAML. #1489 grows the example set from 8 to 20 files but narrows the unit tests'
// examplesDir() to examples/v2/quickstart, so 16 of them are never exercised.
//
// REGRESSION: GREEN at pr-1489 (all 20 load). It exists to keep the untested
// reference/ and pretrain/ examples from silently rotting, and is the piece of the
// harness worth upstreaming.
func TestVerify_F1489_R_AllExamplesLoad(t *testing.T) {
	dir := filepath.Join(repoRoot(t), "examples", "v2")
	if _, err := os.Stat(dir); err != nil {
		t.Skipf("examples/v2 not present on this ref: %v", err)
	}

	var found int
	err := filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(p, ".yaml") && !strings.HasSuffix(p, ".yml") {
			return nil
		}
		found++
		rel, _ := filepath.Rel(dir, p)
		t.Run(rel, func(t *testing.T) {
			if _, err := task.LoadFromFile(p); err != nil {
				t.Errorf("F1489-R: examples/v2/%s does not load: %v", rel, err)
			}
		})
		return nil
	})
	if err != nil {
		t.Fatalf("walk examples: %v", err)
	}
	if found == 0 {
		t.Error("F1489-R: no example YAML found under examples/v2 — harness needs updating")
	}
	t.Logf("validated %d example YAML file(s) under examples/v2", found)
}
