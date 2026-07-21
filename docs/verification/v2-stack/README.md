# Verification — arena v2 stacked PRs (#1477–#1483, author `vicoooo26`)

Stage ② (verify) of the review pipeline for the 7-PR stacked series that rewrites `arena`
as v2. This directory turns the review findings into **reproducible evidence**: each finding
has a test that fails (contract) or a canary/script that observes the current behavior on the
code under review. **Production code is untouched** — the deliverable is additive test files,
scripts, and docs.

- **Series shape:** clean linear stack; each PR fully contains the previous. The *real* scope
  of each PR is its `prev..this` increment (see `verify-manifest.json` `prs[].range`).
- **Harness built at the stack tip** `pr-1483` (`e3af550`) so all cumulative code is reachable
  in one tree.
- **Build/vet baseline:** all 7 branches `go build` **and** `go vet` clean independently
  (no compile-break findings). Full `go test ./...` at the tip is green **except** `test/e2e`
  (finding PR1483-F1).

## How to run

```bash
# unit layer (all confirmations except the two static ones)
go test ./pkg/task/ ./pkg/cli/ ./pkg/client/ ./pkg/provider/ -run Verify -v

# static/config layer
bash docs/verification/v2-stack/scripts/verify-static.sh
bash docs/verification/v2-stack/scripts/verify-gitignore.sh

# re-run after a fix, polarity-aware verdict (in place, or graft onto a fixed ref / the PR tip)
bash docs/verification/v2-stack/scripts/re-verify.sh
bash docs/verification/v2-stack/scripts/re-verify.sh pr
```

Raw captured output is in `results/`.

## Observed vs. expected

Polarity: **contract** = asserts intended behavior → red today, green when fixed. **canary** =
asserts current behavior → green today, must be inverted when fixed.

| Finding | PR | Sev | Polarity | Layer | Status | Evidence |
|---|---|---|---|---|---|---|
| PR1482-F1 `--set` intermediate array drops value **+** nested array **panics** | 1482 | **blocker** | contract | unit | **Confirmed** | `storages[0].name=data` → `storages=[]`; `matrix[0][1]=x` → `panic: index out of range [0] with length 0` |
| PR1481-F1 single-node PyTorch + `--gpus` fails validation | 1481 | high | contract | unit | **Confirmed** (bite-checked) | `worker.replicas must be > 0, got 0` |
| PR1483-F1 e2e suite not gated by `//go:build v2e2e` → `go test ./...` hard-fails | 1483 | high | canary | static+build | **Confirmed** | 12 files lack the tag; `results/test-1483` fails only `test/e2e` BeforeSuite |
| PR1482-F2 `delete -f` ignores YAML namespace (asymmetric with `run`) | 1482 | high | canary | static | **Confirmed** | `delete.go:44 resolveNS("")` vs `run.go:90 resolveNS(t.Namespace)` |
| PR1480 MPI log selector uses `replica-type=launcher`; `LabelJobRole` defined-but-unused | 1480 | high* | canary | unit+static | **Confirmed current behavior** — correctness needs operator-label confirmation | selector `…/replica-type=launcher`; `LabelJobRole` unused in production |
| PR1477-F1 unanchored `arena-v2` gitignore ignores `cmd/arena-v2/` source | 1477 | medium | canary | static | **Confirmed** | `git check-ignore` matches `cmd/arena-v2/main.go` |
| PR1479-F1 `ResolveMPIVersion` drops `ErrCRDNotFound` sentinel | 1479 | medium | contract | unit | **Confirmed** (bite-checked) | `errors.Is(err, ErrCRDNotFound) == false` |
| PR1478-F2 `tensorboard=false` materializes non-nil config | 1478 | low* | contract | unit | **Confirmed (latent)** — not reachable via current CLI | `Logging.TensorBoard != nil` after `tensorboard=false` |
| PR1478-F5 `setInt(n>0)` can't set priority to 0/negative | 1478 | low* | contract | unit | **Confirmed (latent)** — also CLI-gated | priority stays `50` after override to `-100` |

\* severity qualified in the manifest: PR1480 correctness is **unconfirmed** pending the
mpi-operator pod-label contract; PR1478 items are **latent/API-level** (guarded by the current CLI).

### "Harness bites" check
Applied the proposed fix, confirmed the test flips, then reverted (production diff empty):
- **PR1479-F1**: add `%w` wrap → `TestVerify_PR1479…` PASS.
- **PR1481-F1**: `ensureWorker` default `Replicas:1` → `TestVerify_PR1481…` PASS.

PR1482-F1 is direct-observation (empty slice + process panic) so needs no fix to interpret;
its green-flip will be validated by `re-verify.sh` when a fix is pushed.

## Proposed fixes (summary)
- **PR1482-F1**: persist grown slices back into the parent at every navigation level in
  `setValueAtPath` (not only the last segment); add end-to-end `ApplySetOverrides` tests for
  intermediate/nested array indices.
- **PR1481-F1**: keep master-only PyTorch through overrides (route resource flags to `Master`
  when `Worker==nil`), or default `ensureWorker` `Replicas:1`; add a submit→override→validate test.
- **PR1483-F1**: add `//go:build v2e2e` to every `test/e2e/*.go`.
- **PR1482-F2**: `delete.go` → `resolveNS(t.Namespace)`.
- **PR1480**: confirm operator label; if `job-role`, switch the selector to `constants.LabelJobRole`
  (branch by apiVersion if needed) and invert the canary.
- **PR1479-F1**: `fmt.Errorf("MPIJob CRD not found in cluster: %w", ErrCRDNotFound)`.
- **PR1477-F1**: anchor gitignore to `/arena-v2`.
- **PR1478-F2/F5**: allocate TensorBoard only when enabling; track flag-changed for signed fields.

## Continuing after the fix (handoff)

All durable state lives on this pushed branch (`verify/v2-stack` on the reviewer fork
`cheyang/arena`): harness tests, `verify-manifest.json`, `.last-reviewed`, scripts, `results/`.
Machine-local things (kubeconfig, `gh` auth, toolchain) do not travel; L1/L2 verdicts are
deterministic across machines.

Copy-paste kickoff for a fresh agent / next round:

> Resume the arena v2 review pipeline. Check out `verify/v2-stack` from `cheyang/arena`,
> `git fetch`, read `docs/verification/v2-stack/README.md` + `.last-reviewed`, then run
> `bash docs/verification/v2-stack/scripts/re-verify.sh pr`. Report per finding
> Fixed / Still-broken / Partial / Harness-update (canary is fixed only when it flips —
> invert it then). Review the `.last-reviewed..head` delta for regressions, update the table,
> advance `.last-reviewed`, commit + push. Do **not** post to the GitHub PRs without explicit
> approval.
