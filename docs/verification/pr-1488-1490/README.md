# Verification harness — kubeflow/arena #1488 / #1489 / #1490

Reviewer-private harness for the round-1 review of three open PRs by `vicoooo26`.
**No production code is modified** — everything here is additive (one Go test package
under `test/verify/`, plus this directory).

| PR | base | head reviewed | scope |
|----|------|---------------|-------|
| [#1488](https://github.com/kubeflow/arena/pull/1488) | `master` | `93f63fc4` | v0.15.5 changelog + `workflow_dispatch` on the release workflow |
| [#1489](https://github.com/kubeflow/arena/pull/1489) | `develop-v2` @ `56912af0` | `030d6ea2` | v2 docs guides + examples restructure |
| [#1490](https://github.com/kubeflow/arena/pull/1490) | `develop-v2` @ `56912af0` | `10a35bd4` | e2e kind cluster setup/teardown |

The three PRs are **independent siblings**, not a stack (`#1489` is not an ancestor of
`#1490`; both branch from the merged `#1486` commit `56912af0`). `#1488` sits on `master`,
which is why the release-workflow checks are ref-parameterized shell scripts rather than
Go tests — `develop-v2` carries no `release.yaml`.

## Observed vs expected

| Finding | PR | Sev | Claim | Verdict |
|---|---|---|---|---|
| **F1490-1** | 1490 | high | `test/e2e` has no `//go:build v2e2e`, so `go test ./...` dies in BeforeSuite | **Still present** (carry-over of PR1483-F1) |
| **F1490-2** | 1490 | med | `INSTALL_METHOD=helm` is unreachable — no `charts/` at `v1.9.3` | **Confirmed** |
| **F1490-3** | 1490 | med | BeforeSuite gate hardcodes ns `kubeflow`, ignoring the script's `NAMESPACE` | **Confirmed by inspection** |
| **F1489-1** | 1489 | med | shm snippet marks `mount_path`/`shm` "Optional"; both omissions fail to load | **Confirmed** |
| **F1489-3** | 1489 | low | docs use `arena job submit`, which is not a registered command | **Confirmed** |
| **F1489-R** | 1489 | info | suspicion that the 16 new examples ship broken | **Not reproduced** — all 20 load; residual gap is coverage |
| **F1488-1** | 1488 | med | the "manual re-release" path is not idempotent and works exactly once | **Confirmed** |
| **F1488-2** | 1488 | low | `workflow_dispatch` has no branch guard | **Confirmed** |

### F1490-1 — the reproduction

`go test ./...` from a clean checkout, no cluster prerequisites:

```
base develop-v2 56912af0 :  [BeforeSuite] [FAILED] arena-v2 binary not found in bin/ or PATH
                            suite_test.go:35   → Ran 0 of 63 Specs, FAIL
pr-1490 10a35bd4         :  [BeforeSuite] [FAILED] training-operator not found in kubeflow namespace
                            suite_test.go:31   → Ran 0 of 63 Specs, FAIL
pr-1490 + make arena-v2  :  still FAILS at suite_test.go:31
grep -rn "go:build" test/e2e/ → NO BUILD TAGS IN test/e2e
```

So the pre-existing breakage is not introduced by #1490, but #1490 moves the failure
earlier and makes it survive the obvious workaround. CI does not catch it because
`make v2-test` enumerates `V2_PACKAGES` rather than `./...`; a contributor typing
`go test ./...` still gets a red suite. #1490 is the natural place to fix it — one line
per file in `test/e2e/`.

### F1489-1 — the reproduction

The doc snippet annotates both fields as optional; each omission is rejected:

```
(a) mount_path omitted, shm: 64Gi
    Error: failed to load task: validation failed: storages: storage "shm": mountPath must not be empty
(b) shm omitted, mount_path: /dev/shm
    Error: failed to load task: validation failed: storages: storage "shm": must specify exactly one of
           pvc, shm, tmp, hostpath, configmap, or secret
```

Worth noting for the fix discussion: `pkg/provider/interface.go` *does* default an empty
SHM mount path to `/dev/shm`, and `pkg/provider/interface_test.go` asserts that it does —
but `task.Storage.Validate` rejects the input before the provider ever sees it, so that
code path is unreachable in production and its unit test only passes by constructing the
struct directly. The doc describes what the code intends; validation forbids it.

## Round 2 (2026-07-28) — #1489 head `8ae206ae`

Author pushed `misc: update v2 docs and relax Validate()` after round 1. #1488 (`93f63fc4`)
and #1490 (`10a35bd4`) are unchanged.

**F1489-1 FIXED, and fixed the better way.** `pkg/task/types.go` now moves the mount-path
check below the storage-type check and exempts SHM:

```go
if s.MountPath == "" && s.SHM == "" {
    return fmt.Errorf("storage %q: mountPath must not be empty", s.Name)
}
```

That makes the previously unreachable defaulting in `pkg/provider/interface.go` live. Verified
end-to-end - a storage with `shm: 64Gi` and no `mount_path` now renders:

```
volumeMounts: [{"mountPath": "/dev/shm", "name": "shm"}]
volumes    : [{"emptyDir": {"medium": "Memory", "sizeLimit": "64Gi"}, "name": "shm"}]
```

The doc was corrected in the same pass: `shm` is now marked **Required** rather than claiming a
non-existent 2Gi default, and `mount_path` reads "Optional for shm". `pkg/task/types_test.go`
gained coverage; `make v2-test`, `pkg/task` and `pkg/provider` all pass.

**F1489-3 FIXED.** No `arena job submit` remains in docs/.

**F1489-4 / F1489-5 FIXED** (cheyang's comments): README now states the rename to `arena`
explicitly, and the version placeholders agree at `0.1.0`.

**F1489-2 STILL OPEN** (cheyang comment `3656988564`): frameworks.md:274 still states
deepspeed maps to the MPI provider and yields an MPIJob, while the example it links as
"verified end-to-end", `examples/v2/pretrain/deepspeed-bert.yaml`, still sets
`framework.name: pytorch`. The prose is the correct half; the example is mislabeled.

**#1490 F1490-1 unchanged and still the only merge blocker in the set.**

## #1488 follow-up (2026-07-28) — scoping the workflow_dispatch guard

cheyang asked whether the guard should allow a develop branch and release branches as well as
master. Checked against the actual upstream branch list rather than assumed:

| branch | `release.yaml` | `VERSION` | dispatchable |
|---|---|---|---|
| `master` | yes | yes (`0.15.5`) | yes |
| `release-0.12` | yes | yes (`0.12.0`) | yes |
| `release-v0.6.0` | no (predates it) | yes (`0.5.0`) | no |
| `develop-v2` | **no** | **no** | **no** |

`develop-v2` cannot be a dispatch target at all: the workflow file is absent there so GitHub
never lists it in the branch picker, `cat VERSION` would fail, and there is no
`arena-installer` target. Putting it in the guard is dead configuration. Recommended condition
is `master` + `release-*`.

Bite-tested: the guard only needs to go on the two entry jobs, since the rest cascade.

```
package-arena-installer  needs=None                                     <- guard
build-arena-image        needs=None                                     <- guard
release-image            needs=[build-arena-image]                      skips
push_tag                 needs=[package-arena-installer, release-image] skips
draft_release            needs=[push_tag]                               skips
```

YAML parses, and check-release-workflow.sh flips F1488-2 from FAIL to
"OK: a branch guard is present". F1488-1 correctly still reports FAIL (independent finding).

**Non-obvious consequence:** widening to `release-*` makes F1488-1 MORE important, not less.
`release-0.12` carries `VERSION=0.12.0`, so a dispatch from there would overwrite
`ghcr.io/kubeflow/arena:0.12.0` and then die at `git tag -a v0.12.0`. The blast radius grows
from "master's current version" to "any historical release branch's published image", so the
two fixes should land together. Posted as reply `3663248002` on #1488.

## Layout

```
docs/verification/pr-1488-1490/
  README.md                        this file
  verify-manifest.json             finding → tests → polarity → layer
  .last-reviewed-{1488,1489,1490}  per-PR marker (three independent PRs)
  scripts/
    re-verify.sh                   one-line re-run of everything
    check-release-workflow.sh      F1488-1, F1488-2 (ref-parameterized)
    check-trainer-chart-path.sh    F1490-2 (needs network)
    check-suite-gate-namespace.sh  F1490-3 (static)
  results/                         captured raw output per PR
test/verify/verify_test.go         L1 Go layer (grafted onto each PR head)
```

## Running it

From a checkout of this verification branch:

```bash
bash docs/verification/pr-1488-1490/scripts/re-verify.sh          # all three, current heads
bash docs/verification/pr-1488-1490/scripts/re-verify.sh 1490     # one PR
bash docs/verification/pr-1488-1490/scripts/re-verify.sh 1490 <sha>
```

It resolves each PR's head from the manifest's PR URLs (`git fetch <url> pull/<n>/head`,
so no locally-named remote is required), grafts `test/verify/` onto a throwaway worktree
of that head, runs the layers that apply, writes `results/pr-<n>.log`, and prints the
marker-advance command. Exit 0 iff every finding is fixed.

Prerequisites: `git`, `go` (1.26+, matching `go.mod`), network for the trainer-repo clone
in F1490-2 and for fetching PR heads. No cluster needed.

## Polarity — reading the results after a fix

| Test | Polarity | Now | After the fix |
|---|---|---|---|
| `TestVerify_F1490_1_E2EBuildTag` | contract | RED | GREEN |
| `TestVerify_F1490_1_MakefileTagContract` | contract | skip/RED | GREEN |
| `TestVerify_F1489_1_ShmOptionalFieldsHonored` | contract | RED (both subtests) | GREEN |
| `TestVerify_F1489_3_NoJobSubmitInDocs` | contract | RED | GREEN |
| `TestVerify_F1489_R_AllExamplesLoad` | **regression** | **GREEN** | **must stay GREEN** |
| `check-release-workflow.sh` | contract | exit 1 | exit 0 |
| `check-trainer-chart-path.sh` | contract | exit 1 | exit 0 |
| `check-suite-gate-namespace.sh` | contract | exit 1 | exit 0 |

`F1489-R` is the one **regression guard**: it is green today and a red result there means
a *new* example broke, not that a finding was fixed. Everything else is a contract test —
red is the reproduction.

The contract tests are written against the *doc's own promise* rather than against a
hardcoded expectation, so they self-resolve whichever way the author fixes it. Removing
the misleading "Optional" annotations turns `F1489-1` green just as relaxing `Validate`
would; rewriting the examples as `arena submit` turns `F1489-3` green.

## Harness-bites check

Every contract check was confirmed to fail *for the right reason*: the proposed fix was
applied to a throwaway worktree, the check re-run, then the fix reverted (production diff
back to empty each time).

| Check | Fix applied in the bite test | Result |
|---|---|---|
| `TestVerify_F1490_1_E2EBuildTag` | prepend `//go:build v2e2e` to all 20 `test/e2e/*.go` | RED to **GREEN** |
| `TestVerify_F1490_1_MakefileTagContract` | + `go test -tags v2e2e ./test/e2e/` in the Makefile | SKIP to **GREEN** |
| `go test ./...` (F1490-1 repro) | same | FAIL at BeforeSuite to **all packages ok** |
| `TestVerify_F1489_1_ShmOptionalFieldsHonored` | drop the two `# Optional` annotations | RED to **GREEN** |
| `TestVerify_F1489_3_NoJobSubmitInDocs` | `arena job submit` to `arena submit` | RED to **GREEN** |
| `TestVerify_F1489_R_AllExamplesLoad` | (regression guard, no fix) | GREEN to **GREEN** |
| `check-release-workflow.sh` | `if: github.ref == refs/heads/master` + `git ls-remote --tags` existence guard | exit 1 to **exit 0** |
| `check-trainer-chart-path.sh` | `TRAINER_REF` to `master` | exit 1 to **exit 0** |

Two things the bite tests settled beyond the findings themselves:

1. **F1490-1 fix is complete and cheap.** After adding the build tag, `go test ./...`
   passes every package - the e2e suite is the only thing standing between the repo and a
   working default test command.
2. **F1490-2 is a mismatched pair of defaults, not a missing directory.**
   `charts/kubeflow-trainer` exists at trainer `master` but not at `v1.9.3`: `CHART_PATH`
   was written for the v2 Kubeflow Trainer chart while `TRAINER_REF` pins the v1
   training-operator. Bumping either one alone is not enough - they have to agree.

## No live layer — and why

The available sandbox cluster is a shared Aliyun ACK v1.36 with no `kind` binary and no
training-operator installed. Provisioning a cluster or installing CRDs there would have
been a cluster-wide change to shared infrastructure, so it was deliberately not done; the
harness is L1+L2 only and needs no cluster. `F1490-1`'s reproduction only ever issues a
read-only `kubectl get`. To exercise the e2e suite for real, run
`make v2-e2e-setup-cluster` on a throwaway kind host — note that `INSTALL_METHOD=helm`
will fail there until F1490-2 is addressed.

## Continuing after the fix (another machine / another round)

```bash
git clone https://github.com/cheyang/arena.git && cd arena
git checkout verify/pr-1488-1490
bash docs/verification/pr-1488-1490/scripts/re-verify.sh
# then, per PR reviewed this round:
echo <head-sha> > docs/verification/pr-1488-1490/.last-reviewed-<pr>
git commit -am "verify: round N for #<pr>" && git push origin verify/pr-1488-1490
```

`re-verify.sh` prints the `last-reviewed..head` commit range and changed files per PR, so
the incremental review scope is resolved without anyone typing a sha. If a finding comes
back as a compile or path error rather than an assertion failure, the fix changed the code
shape — update the harness, then re-run.

Kickoff prompt for a fresh agent:

> Resume the arena v2 review from the `verify/pr-1488-1490` branch on `cheyang/arena`.
> Run `bash docs/verification/pr-1488-1490/scripts/re-verify.sh`, read
> `docs/verification/pr-1488-1490/README.md` for the polarity table, report Fixed /
> Still-broken per finding, review each `last-reviewed..head` delta for regressions,
> then advance the `.last-reviewed-*` markers and push.

## Relationship to comments already on the PRs

`cheyang` had already posted inline comments before this harness was built. Re-derived
independently:

- **#1489 shm/storage** (`3656988349`) — **agree**, and the harness adds the corollary
  that the provider's defaulting is dead code.
- **#1489 deepspeed→MPI** (`3656988564`) — **agree**: `pkg/provider/mpi.go` does accept
  `deepspeed`, so the guide's mapping is right, but the linked "verified end-to-end"
  example `examples/v2/pretrain/deepspeed-bert.yaml` sets `framework.name: pytorch` and
  produces a PyTorchJob. Not encoded as a test (it is a judgement about which artifact is
  wrong, not a behavioural claim).
- **#1489 `arena job submit`** (`3656988745`) — **agree**, encoded as F1489-3. Actual
  error is `unknown flag: --name` (cobra treats `submit` as an argument to `job`).
- **#1489 binary name / version placeholders** (`3656988964`, `3656989212`) — agree,
  cosmetic, not encoded.
- **#1488 changelog date** (`3656943526`) — **addressed by the author**: the date moved
  `2026-07-14` → `2026-07-28` in the force-push to `93f63fc4`. It is now a day in the
  future and still a guess, since `v0.15.5` has no tag yet.
