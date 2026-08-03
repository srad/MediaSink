# Roadmap

Status tracking for MediaSink, so work in progress survives across sessions.

This file records *what is done and what is left*. It does not explain how the system
works or how to run it:

- Architecture, build/run/test commands, and conventions: `AGENTS.md`
- Installation and user-facing setup: `README.md`

Last verified: 2026-08-03, at commit `fd95907`.

## Status symbols

```
[x]  done and verified
[ ]  open, not started
[~]  partial: landed, but with a named gap still open
[>]  deferred: postponed on purpose, owning phase named
[-]  rejected: decided against, reason recorded so it is not relitigated
```

Lines without a box are context, not tasks.

---

# Server: refactor to idiomatic Go

Converts the Go server from package-level mutable state and free functions to injected
dependencies. Runs in phases; every phase must build, pass tests and lint, and leave
the golden HTTP tests byte-identical unless an API change is intended.

Measured at commit `fd95907`: coverage 39.8%, lint 0 issues, 106 golden route cases.

## Phase 0 - Safety net and tooling (`712c7eb`)

```
[x] Golden HTTP tests: boot the real router, snapshot status and body per route.

[~] Route coverage. 106 cases across 3 golden files, but not everything:
    [ ] /api/v2/ws has no golden coverage at all. A websocket upgrade cannot be
        driven through httptest's recorder. Needs a different harness.
    59 routes are registered in router.go against 56 auth-gate cases. The
    difference is /ws plus the two public auth routes, which the test seed
    exercises instead.
    9 routes appear only in the auth-gate golden, not the authenticated one: six
    that would start background work inside the test process (import, recorder
    resume, analyze all, previews regenerate, jobs resume, channel upload) and
    the three DELETEs, which would make later cases order-dependent.

[x] lint.sh and server/.golangci.yml. All 122 findings resolved or explicitly
    excluded. Note the config disables golangci-lint's default issue caps, which
    otherwise truncate the report so the total never settles.

[x] Fixed defects found by that pass: ignored errors in the startup cleanup
    loops, an ignored Row.Scan that silently skipped the stale frame-vector
    rebuild, an ignored SetWriteDeadline that could let a websocket write block
    forever, dropped ONNX Destroy errors leaking native memory, and a missing
    rows.Err() check that produced silently partial similarity data.

[x] test.sh fixes: the no-test counter reported 1 package instead of 23, and
    coverage now uses -coverpkg so integration-style tests credit the packages
    they actually drive.
```

## Phase 1 - Delete the abandoned v2 stack (`638ae62`, `fd95907`)

```
[x] Deleted 918 lines across 21 files: internal/http/v2,
    internal/http/middleware, internal/service/*, internal/store/relational,
    store/contracts.go, and both Wire files. Nothing called any of it.

[x] Verified inert before deleting: no live code imported it, no symbol or
    dependency was orphaned, it carried no swagger annotations, and it built with
    no edits after removal. Goldens and swagger.json were byte-identical.

[x] Extracted parseToken into internal/middleware/token.go with sentinel errors,
    taking the JWT secret as a parameter. All 9 error strings and both log levels
    preserved. 20 test cases added; parseToken at 90.9%.
```

## Phase 2a - Config becomes a value, not a global

Objective: read `Cfg` once at the composition root and pass it down, so services and
middleware stop reaching for global state.

```
[ ] Thread Cfg from the composition root. 18 config.Read() call sites.

[ ] Fold in the 9 genuine os.Getenv calls that bypass config entirely: SECRET in
    three places, DB_ADAPTER re-derived in three, ONNXRUNTIME_LIB, and the DSN
    parts in db/db.go. The JWT secret is currently read from the environment on
    every authenticated request.

[ ] Make mustEnv return an error instead of calling log.Panicf.
```

Gate: goldens byte-identical; `internal/middleware` becomes testable without mutating
process environment.

## Phase 2b - Kill the global DB handle

Objective: replace `var DB *gorm.DB` with a concrete store passed to its consumers.
This is the keystone and the largest phase.

```
[ ] Replace db.Init() with db.Open(cfg) (*db.Store, error), returning errors
    instead of panicking.

[ ] Migrate active record to repository. 74 functions inside internal/db read the
    global; 23 of those are methods on domain types (Recording.Save(),
    Job.CreateJob(), ChannelID.FavChannel()). This is a design migration, not a
    mechanical rename.

[ ] Repoint roughly 81 call sites outside internal/db. The 272 db.Type references
    stay put; only behaviour moves.

[ ] Remove the 6 direct db.DB reach-throughs in non-test code, plus 11 in
    services/chapter_regeneration_test.go.
```

Gate: `grep -rn 'db\.DB\b' server/internal --include='*.go' | grep -v '/internal/db/'`
returns nothing; the 23 active-record methods are gone rather than wrapped.
Recommended: split per aggregate (recordings, channels, jobs, users) with green goldens
between each.

## Phase 3 - Services become injected structs

Objective: convert free functions to constructor-injected structs and move mutable
package state into fields.

```
[ ] Convert the 49 exported functions in internal/services to methods on injected
    structs.

[ ] Move 9 clusters of mutable package state into struct fields. Riskiest single
    item: streaming_service.go holds live recording state in three package maps
    plus two mutexes. Move it alone, with -race.

[ ] Fix a known bug: detectors.CreateSceneDetector(t) caches on first call and
    thereafter ignores its detectorType argument. Harmless today only because one
    detector type exists. Add the test that would have caught it.

[ ] Remove four pass-through wrappers in services/job_service.go that exist only
    for "backward compatibility" with an earlier half-finished extraction.
```

Gate: `-race` clean on services and jobs; no `vector.Default()` or `vector.SetDefault`
remaining.

## Phase 4 - Handlers become structs, and a composition root

Objective: group the 58 handlers in `internal/api/v1` into structs holding their
services, and wire everything in one place.

```
[ ] Group the 58 handlers into per-resource structs.

[ ] Add internal/httperr with a typed error and a single renderer. Keep today's
    bare-string wire format in this phase; changing it belongs to Phase 6.

[ ] Take the goroutine out of the router constructor. Setup() currently runs
    `go ws.WsListen()`, so merely building a router starts background work.

[ ] Return *gin.Engine from Setup instead of http.Handler, which app.go
    immediately type-asserts back.

[ ] Write the composition root: roughly 45 lines of plain constructor calls. No
    DI library.
```

Gate: goldens byte-identical; swagger unchanged.

## Phase 5 - Break up `internal/util` and `internal/models`

Objective: replace grab-bag packages with named ones.

```
[ ] Split util: video.go (771 lines) to internal/media, sys.go (408) to
    internal/procs, nettop.go to internal/netstat.

[ ] Delete util.ParseNumbers and util.FileNameWithoutExtension, which have no
    consumers.

[ ] Fold internal/models/sort.go into its consumer and drop the 20-line models
    package. models/requests and models/responses stay.

[ ] Promote internal/store/vector to internal/vector, now that store/ holds
    nothing else.
```

## Phase 6 - API redesign

Objective: fix the API defects listed below. Deliberately last, because it is the only
phase that reaches outside Go.

```
[ ] Fix the three known API defects (see "Known defects" below).
```

Blast radius: 31 frontend call sites, the hand-written Rust `cli/src/api.rs` (about
1.1k lines), and the Dart mobile client. The generated TypeScript client absorbs shape
changes; its callers do not. Ship as its own change with matching frontend, CLI and
mobile commits.

---

# Deferred

Postponed on purpose. Each names the phase that owns it.

```
[>] Rename main.ApiVersion to APIVersion. Not a safe standalone rename: the
    symbol is injected via -X 'main.ApiVersion=...' in build.sh and run.sh, and
    the Go linker silently ignores -X flags matching no symbol. Renaming without
    updating both scripts leaves the API version empty, and every client request
    then fails with 412. Owner: whenever done, in lockstep with both scripts.
    Currently excluded in server/.golangci.yml.

[>] Rename ChannelRequest.Url to URL. The JSON tag keeps the wire name, but every
    Go usage moves. Owner: Phase 6. Excluded in server/.golangci.yml.

[>] revive's exported (22 findings) and package-comments (34) rules. Both are
    missing-documentation churn, not correctness. Owner: after the substantive
    findings are gone; drop both exclusions together.

[>] Promote internal/store/vector to internal/vector. Owner: Phase 5, batched
    with the other package moves to avoid isolated import churn.
```

# Rejected

Recorded so they are not reopened without new information.

```
[-] google/wire for dependency injection. Archived by Google on 2025-08-25, final
    release v0.7.0. Verified it still works, including under the vendored build,
    but adopting an archived tool for a graph of roughly 30 nodes is not worth it.

[-] uber-go/fx, uber-go/dig, samber/do. All maintained, but all resolve at
    runtime. That moves wiring errors from compile time to startup, which is a
    regression against what the code already gives you. A hand-written
    composition root keeps compile-time checking and adds no dependency.

[-] Finishing the v2 migration instead of deleting it. A parallel tree that only
    goes live at the very end never goes live; this repo has two abandoned
    attempts as evidence. The live code is refactored in place instead.

[-] Keeping safeJoinPath. It was a path-traversal guard that was written but
    never wired to any call site. Deleted with the reasoning recorded at its old
    location in db/channel_name.go: validChannelName is ^[a-z_0-9]+$, which
    admits neither "." nor "/", so traversal cannot reach filepath.Join. If a
    future code path writes channel names by another route, restore a guard
    rather than trusting the regex alone.
```

# Known defects, not yet fixed

Found while auditing. All are client-visible, so they are grouped into Phase 6 rather
than fixed piecemeal.

```
[ ] Not-found is inconsistent across three endpoints. For an id that does not
    exist: GET /channels/1 returns 500, GET /videos/1 returns 200 with a
    zero-valued object, and GET /analysis/1 returns 200 with null fields. All
    three should be 404. The golden files record the current behaviour, so the
    fix will show as a reviewable diff.

[ ] A duplicate signup returns 500 where 409 is correct. db.ExistsUsername
    returns an error to mean "true", so a caller cannot distinguish "name taken"
    from "database unavailable". Fixing the signature to (bool, error) belongs to
    Phase 2b; the status code change belongs to Phase 6. No golden currently pins
    this case, so add one before changing it.

[ ] The websocket auth workaround passes the bearer token as a URL query
    parameter, which leaks it into access logs and browser history. It is a
    deliberate, documented workaround; changing it is an API change.
```

# In-code TODOs

```
[ ] server/internal/api/v1/videos.go:51 - make into a cancelable job.
[ ] server/internal/api/v1/videos.go:70 - "do it" (unspecified).
[ ] server/internal/jobs/handlers/cutting.go:136 - ffmpeg gives no obvious
    progress information for cutting and merging; recheck.
[ ] frontend/src/components/VideoEditor.vue:69 - check if needed.
```

---

# Other areas

Inventory below was measured on 2026-08-03. These areas have NOT been audited: file and
test counts are inventory, not a health assessment. No backlog is listed for them
because none has been established by reading the code.

## Frontend (Vue 3 + TypeScript)

Inventory: 66 `.vue` and 54 `.ts` files in `frontend/src`. 20 test files in
`frontend/tests` (components, views, stores, utils) plus 4 mock helpers. Note that
`frontend/src` itself contains no test files. Tooling exists: `test:unit` (vitest),
`lint` (eslint), `type-check` (vue-tsc).

```
[ ] Audit the frontend. Not yet assessed.
```

## CLI (Rust, `mediasink-tui`)

Inventory: 37 Rust files, about 14.3k lines, 55 `#[test]` functions across 23 of those
files as inline `mod tests`.

```
[ ] Audit the CLI. Not yet assessed.
```

## Mobile (Flutter/Dart)

Inventory: 197 Dart files, about 16.8k lines, 13 test files in `mobile/test`. Lint
config present at `mobile/analysis_options.yaml`.

```
[ ] Audit the mobile client. Not yet assessed.
```

## Cross-cutting

```
[ ] There is no CI. No workflow configuration exists anywhere in the repository,
    so every gate is run manually.

[ ] test.sh and lint.sh cover the Go module only. The frontend, CLI and mobile
    test and lint targets exist but are not wired into a single entry point.
```

---

# Maintaining this file

- Update it at the end of each phase, not in between.
- Re-stamp "Last verified" with the date and commit whenever numbers are re-measured.
  Numbers here are snapshots and go stale silently otherwise.
- When a phase completes, record its commit SHA in the heading and check its boxes.
- Use `[~]` honestly. If a phase lands with a gap, name the gap rather than marking it
  `[x]`.
