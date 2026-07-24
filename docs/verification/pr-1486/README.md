# Verification — kubeflow/arena PR #1486

**PR:** https://github.com/kubeflow/arena/pull/1486 — *"feat: add CLI core, extended commands, examples, and e2e tests for Arena v2"* (author `vicoooo26`)
**Head reviewed:** `2c0084fe` · **Base:** `develop-v2` (empty init commit)
**Toolchain:** go 1.26.5 (auto-downloaded; go.mod pins 1.26.5)

## Context

PR #1486 is the **resubmission** of the earlier 7-PR v2 stack (`#1477–#1483`, all closed
unmerged). The author reorganized it into three combined PRs — `#1484` (scaffolding, closed),
`#1485` (model/client/providers, closed), and **#1486** (CLI core + extended + examples/e2e,
open). Because #1486's base `develop-v2` is the empty init commit, #1486 actually carries the
**entire cumulative v2 tree** (140 files, ~28k insertions), so every round-1 finding is in
scope here. This round re-proves each finding against the resubmitted head.

The deliverable is **additive only**: `pkg/*/verify_findings_test.go` + this docs dir.
Production code is untouched; the author's own unit suite (`go test ./pkg/... -skip TestVerify_`)
is green.

## Findings — observed vs. expected

| Finding | Sev | Round-1 | This round (#1486) | Evidence |
|---|---|---|---|---|
| **PR1482-F1** `--set` array parser | blocker | broken | ✅ **FIXED** | `storages[0].name=data` round-trips; `matrix[0][1]=x` → `[[nil,"x"]]`, no panic. setter.go rewritten w/ `updateSliceInParent` write-back + `maxIndex` OOM guard. |
| **PR1481-F1** single-node pytorch + `--gpus` | high | broken | ❌ **STILL BROKEN** | `worker.replicas must be > 0, got 0` — `ensureWorker()` still resurrects `Worker{Replicas:0}`. |
| **PR1483-F1** e2e not build-gated | high | broken | ❌ **STILL BROKEN** | `go test ./test/e2e/...` → `[BeforeSuite] CRD directory not found`; Ran 0/52, package FAIL. No `//go:build` tag on the suite. |
| **PR1479-F1** `ErrCRDNotFound` sentinel dropped | medium | broken | ❌ **STILL BROKEN** | `errors.Is == false`; crd.go:72 `fmt.Errorf(... )` still lacks `%w`. |
| **PR1482-F2** `delete -f` ignores YAML namespace | medium | broken | ✅ **FIXED** | `delete.go` threads `yamlNS` → `resolveNS`; precedence flag > YAML > kubeconfig > default. |
| **PR1480** MPI log selector uses replica-type | high?† | canary | 🔵 **UNCHANGED** | selector still `replica-type=launcher`; `LabelJobRole` still unused in prod. Needs operator-label confirmation. |
| **PR1478-F2** `tensorboard=false` → non-nil cfg | latent | present | ⚠️ **STILL PRESENT** | produces `&TensorBoardConfig{Enabled:false}`. CLI-gated, not reachable via submit. |
| **PR1478-F5** negative priority dropped | latent | present | ⚠️ **STILL PRESENT** | `priority=-100` stays `50` (setInt `n>0` guard). |

† severity unconfirmed pending mpi-operator label-contract check.

**Net:** 2 of the round-1 defects fixed (incl. the `--set` **blocker**), **3 real defects
still open** (1 high + 1 high-build + 1 medium), 2 latent, 1 canary awaiting confirmation.

## Reproduce

```bash
bash docs/verification/pr-1486/scripts/re-verify.sh
```

Layers: unit (`pkg/*` contract/canary tests) always run; the `go test ./test/e2e/...` probe
demonstrates PR1483-F1 and needs no cluster (it fails at BeforeSuite by design). The live
layer (actual job submit/delete) requires `KUBECONFIG` + `make v2-e2e-setup` and is not part
of the defect proofs.

## Polarity legend

- **CONTRACT** — asserts intended-correct behavior. Red = bug reproduced; flips green once fixed.
- **CANARY** — asserts current observed behavior. Green = unchanged; invert once the fix lands.

---

## Round 2b — re-check at head `fcd732d4` (2026-07-22)

Author force-pushed a commit *"fix: upgrade Go toolchain and harden RBAC, error handling, code style"*.
Re-ran the harness against the new head. **None of the three open findings were fixed** — the
changes to the relevant files are cosmetic:

| Finding | Change at fcd732d4 | Status |
|---|---|---|
| **PR1479-F1** sentinel | crd.go only lowercased strings: `fmt.Errorf("MPIJob CRD not found…")` → `errors.New("mpijob crd not found…")` — still **no `%w`** | ❌ STILL BROKEN (`errors.Is == false`) |
| **PR1481-F1** pytorch+`--gpus` | override.go only `%q` formatting; `ensureWorker()` unchanged; submit.go changes are gpu-topology/root-cmd, unrelated | ❌ STILL BROKEN (`worker.replicas must be > 0, got 0`) |
| **PR1483-F1** e2e gate | `test/e2e` untouched; still no `//go:build` tag | ❌ STILL BROKEN (BeforeSuite Fail; `go test ./...` red) |

PR1482-F1/F2 remain fixed; PR1480 canary still green; PR1478-F2/F5 still latent-present.

---

## Round 2c — re-check at head `9a113794` (2026-07-22)

Author pushed *"misc: upgrade Go toolchain and harden RBAC, error handling, code style, and
concurrency, add Github workflow YAML"*. Re-ran the harness. **All 3 open findings still red.**

- **PR1481-F1 / PR1479-F1** — `override.go`, `submit.go`, `crd.go` are NOT in this diff; code
  unchanged → still `worker.replicas must be > 0, got 0` and `errors.Is == false`.
- **PR1483-F1** — partially *attempted* but **still broken, and now demonstrably incomplete**:
  - New `Makefile` target `v2-e2e-test` runs `go test **-tags v2e2e** ./test/e2e/` — i.e. the
    author *intends* the suite to be gated by a `v2e2e` build tag.
  - But **no `test/e2e/*.go` file declares `//go:build v2e2e`** (`git grep -c "go:build v2e2e"
    test/e2e` → 0). So the tag passed by the Makefile is a **no-op**: `go test -tags v2e2e
    ./test/e2e/` and `go test ./test/e2e/` behave identically — both fail at `BeforeSuite`.
  - `go test ./...` still fails. The new CI (`.github/workflows/integration.yaml`) only dodges
    this because it runs `make v2-test` = `go test ./test/` (no `...`), which doesn't recurse
    into `test/e2e`.
  - **Fix is now one step:** add `//go:build v2e2e` as the first line of every `test/e2e/*.go`
    (the Makefile already passes the matching `-tags v2e2e`).

PR1482-F1/F2 still fixed; PR1480 canary green; PR1478-F2/F5 still latent-present.

---

## Round 2d — re-check at head `4b4ce3cf` (2026-07-23)

Two more fixes landed. Re-ran the harness:

- ✅ **PR1479-F1 FIXED** — `crd.go:72` now `fmt.Errorf("mpijob crd not found in cluster: %w", ErrCRDNotFound)`; `TestVerify_PR1479_*` passes (`errors.Is == true`).
- ✅ **PR1481-F1 FIXED** — `override.go` reworked; single-node PyTorch + `--gpus` now validates end-to-end; `TestVerify_PR1481_*` passes.
- ❌ **PR1483-F1 STILL OPEN** — `suite_test.go` dropped the crds-dir `Fail`, but `BeforeSuite` still `Fail`s on missing binary, and **no `test/e2e/*.go` carries `//go:build v2e2e`** (Makefile `v2-e2e-test` still passes `-tags v2e2e` with nothing to match). `go test ./...` / `go test ./test/e2e/` still fail at BeforeSuite. One-step fix unchanged: add `//go:build v2e2e` first-line to every `test/e2e/*.go`.

Remaining: **1 real blocker (PR1483-F1)** + 2 latent (PR1478-F2/F5) + 1 canary (PR1480). PR1482-F1/F2 fixed.

---

## Watcher tick — head `a5f0a19e` (2026-07-23)

New push (lint config `.golangci.yaml`, `AfterSuite` cleanup in `suite_test.go`, more e2e
tests, test refactors, workflow tweak). **No tracked-finding delta:**
- PR1479-F1 ✅ fixed · PR1481-F1 ✅ fixed (both still green)
- **PR1483-F1 ❌ still open** — still no `//go:build v2e2e` in `test/e2e`; `BeforeSuite` still
  `Fail`s (now on the binary check), `go test ./test/e2e/` still red.
- PR1478-F2/F5 latent-present; PR1480 canary green; PR1482-F1/F2 fixed.

---

## Watcher tick — head `d58a7a77` (2026-07-23)

New push (small changes in `pkg/cli/helpers.go`, `run.go`, `submit.go`). **No tracked-finding delta:**
PR1479-F1 ✅ / PR1481-F1 ✅ still fixed; **PR1483-F1 ❌ still open** (no `//go:build v2e2e`, BeforeSuite still Fails);
PR1478-F2/F5 latent; PR1480 canary green; PR1482-F1/F2 fixed.

---

## Watcher tick — head `6ddad9fd` (2026-07-23)

New push: RBAC/lifecycle work (`pkg/cli/lifecycle.go` +75, `lifecycle_rbac_test.go` +477,
`pkg/client/kubernetes.go` +69, `kubernetes_test.go` +215; 3-line `test/e2e/logs` tweak).
**No tracked-finding delta:** PR1479-F1 ✅ / PR1481-F1 ✅ still fixed; **PR1483-F1 ❌ still open**
(no `//go:build v2e2e`, BeforeSuite still Fails); PR1478-F2/F5 latent; PR1480 canary green.
NOTE: the ~760-line RBAC/lifecycle delta is **untriaged** by the harness — flag for a full
incremental review (blast radius: RBAC check, kubernetes client) if the user wants it.

---

## Watcher tick — head `90cd237f` (2026-07-24)

New push: `submit.go`, `pkg/constants`, `pkg/log`, `pkg/provider/tensorflow.go`, `pkg/task/types.go`
(+ tests). `test/e2e` untouched. **No tracked-finding delta:** PR1479-F1 ✅ / PR1481-F1 ✅ still fixed;
**PR1483-F1 ❌ still open** (0/20 e2e files carry `//go:build v2e2e`, BeforeSuite still Fails);
PR1478-F2/F5 latent; PR1480 canary green.

---

## Watcher tick — head `95e52036` (2026-07-24)

New push: shell-completion functionality (`pkg/cli/completion_funcs.go` +106, per-command
completion registration). `test/e2e` untouched. **No tracked-finding delta:** PR1479-F1 ✅ /
PR1481-F1 ✅ still fixed; **PR1483-F1 ❌ still open** (0/20 e2e files gated, BeforeSuite still Fails);
PR1478-F2/F5 latent; PR1480 canary green.

---

## Watcher tick — head `f3a89928` (2026-07-24)

New push: workflow tweak, completion test, and a **doc caveat added to `preCreateRBAC`** (author
responded to the owner-less-RBAC inline comment — notes the crash window + that re-submit
re-patches ownerRefs via `CreateOrUpdate`; accurate). **No tracked-finding delta:** PR1479-F1 ✅ /
PR1481-F1 ✅ still fixed; **PR1483-F1 ❌ still open** (0/20 e2e files gated, BeforeSuite still Fails);
PR1478-F2/F5 latent; PR1480 canary green.

---

## Watcher tick — head `08e079bf` (2026-07-24)

New push: 1-line change in `.github/workflows/integration.yaml` only (no Go/test/e2e code).
**No tracked-finding delta:** PR1479-F1 ✅ / PR1481-F1 ✅ still fixed; **PR1483-F1 ❌ still open**
(0/20 e2e files gated, BeforeSuite still Fails); PR1478-F2/F5 latent; PR1480 canary green.

---

## Watcher tick — head `d7814be4` (2026-07-24)

New push: `main.go` cleanup only (-7 lines). **No tracked-finding delta:** PR1479-F1 ✅ /
PR1481-F1 ✅ still fixed; **PR1483-F1 ❌ still open** (0/20 e2e files gated, BeforeSuite still Fails);
PR1478-F2/F5 latent; PR1480 canary green.
