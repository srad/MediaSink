# Roadmap

Status tracking for MediaSink, so work in progress survives across sessions.

This file records *what is done and what is left*. It does not explain how the system
works or how to run it:

- Architecture, build/run/test commands, and conventions: `AGENTS.md`
- Installation and user-facing setup: `README.md`

Last verified: 2026-08-04, at commit `dbd019c`.

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

Measured after slice 2b.2: coverage 40.8%, lint 0 issues, 110 golden route cases.
(After 2b.1: 40.5%, 106 cases. At `f2929a2`, end of Phase 2a: 40.2%. At `fd95907`,
before Phase 2a: 39.8%.)

## Phase 0 - Safety net and tooling (`712c7eb`)

```
[x] Golden HTTP tests: boot the real router, snapshot status and body per route.

[~] Route coverage. 110 cases across 4 golden files (public_auth.golden and its 4
    signup/login cases were added in 2b.2), but not everything:
    [ ] /api/v2/ws has no golden coverage at all. A websocket upgrade cannot be
        driven through httptest's recorder. Needs a different harness.
    59 routes are registered in router.go against 56 auth-gate cases. The
    difference is /ws plus the two public auth routes, which public_auth.golden
    now covers directly.
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
    they actually drive. Superseded in Phase 2a: test.sh was deleted in favour of
    docker-test.sh, which can boot the server.
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

## Phase 2a - Config becomes a value, not a global (`f2929a2`, partial)

Objective: read `Cfg` once at the composition root and pass it down, so services and
middleware stop reaching for global state.

The earlier count of "18 config.Read() call sites" was wrong: two of those grep hits
are comments, so there were 16.

```
[~] Thread Cfg from the composition root. 5 of the 16 config.Read() calls are gone
    (main/app, router, info x2, db.Init). The other 11 are deferred, because
    converting them is the same work later phases already own:
    [ ] 8 sites in internal/db hang off seven path methods on ChannelName and
        RecordingID - GORM column types with about 49 external callers. Re-signing
        them is the active-record migration itself. Owner: Phase 2b.
    [ ] 3 sites are free functions in internal/services, each atop a call chain
        reaching back to handlers and startup. Owner: Phase 3.

[x] Folded in all 9 os.Getenv calls that bypassed config: SECRET x3, DB_ADAPTER x3,
    ONNXRUNTIME_LIB, and the two duplicated DSN lines in db/db.go. Also the two
    LOG_LEVEL reads in main.go, which the count of 9 had excluded as bootstrap.
    The JWT secret is now captured once at router construction rather than read
    from the environment on every authenticated request.

[x] mustEnv is deleted. config.Parse(getenv) is pure and returns an error naming
    every missing variable at once; config.Load() caches it. Note the remaining
    panic: the config.Read() shim that serves the 11 deferred call sites still
    calls log.Panicf, and goes away with the last of them in Phase 3.
```

Also landed in this phase, found by running the suite in a provisioned container
for the first time:

```
[x] Fixed an environment-dependent golden. authenticated.golden pinned the string
    `exec: "yt-dlp": executable file not found in $PATH` for
    POST /channels/:id/resume, which is only true on a host WITHOUT yt-dlp.
    Anywhere it was installed the binary really ran against example.com over the
    network and the golden failed - so the suite passed by accident of a missing
    dependency and could never have run in CI. The seeded channel now points at a
    test-local httptest.Server, and redactSubprocessError normalises the failure
    tail the same way testTmpDir is redacted. Still pinned: the 500 status and
    that the message names the URL. Not a Phase 2a regression; it predates the
    phase and is unchanged in git history.

[x] Added test.Dockerfile and docker-test.sh: the Go suite plus a real boot smoke
    test (signup, login, authenticated route, restart under a different SECRET to
    prove the JWT is bound to config, and the missing-config abort). Deleted
    test.sh, which could not do any of that.
```

Gate: met. Goldens byte-identical apart from the two deliberate lines above,
swagger.json byte-identical, lint 0 issues,
`internal/middleware` now has a test suite that supplies its own secret and database
without mutating process environment. `grep -rn 'os.Getenv' server --include='*.go'`
outside `config/` and tests returns only a comment.

## Phase 2b - Kill the global DB handle (in progress)

Objective: replace `var DB *gorm.DB` with a concrete store passed to its consumers.
This is the keystone and the largest phase.

Re-measured at `ca8a758`. The earlier counts were the seed, not the work: methods like
Job.Cancel and Recording.DestroyRecording never name DB, they persist through
something that does, so they move too.

```
141   functions/methods in internal/db (non-test)
 74   name DB directly - the previously recorded figure, correct but only the seed
 95   in the transitive closure - the actual unit of work
 32   methods on domain types - previously recorded as 23, which undercounts by 9
```

The 9 that were missed: Job.Cancel, Job.Error, Recording.DestroyRecording,
Recording.DestroyPreviews, Recording.DestroyPreview, and the four
Recording.Enqueue{Analysis,Conversion,Cutting,PreviewFrames}Job.

Closure per file, which sets the slice sizes:

```
job.go 26   recording.go 19   channel_id.go 11   frame_vectors.go 9   channel.go 7
setting.go 5   video_analysis.go 5   user.go 4   db.go 4   video_previews.go 4
```

Outside internal/db there are 70 free-function call sites in non-test code. Method
call sites are not grep-countable - `.Save(`, `.Update(`, `.Error(` collide with gorm
and with `error`; a first attempt returned 283 with false positives in files that
touch no database. Deleting the method makes the compiler enumerate them exactly.

Handlers must become closures. Nothing in the chain router -> handler -> service -> db
carries a value, and the gate forbids wrapping, so `func GetChannels(c *gin.Context)`
becomes `func GetChannels(s *db.Store) gin.HandlerFunc`. Precedent already in the tree:
`v1.GetVersion(version, commit)` at api/v1/admin.go:92. Verified this does not move
swagger - GetVersion carries a full annotation block and /admin/version is present in
docs/swagger.json.

Lint constrains the sequencing: revive's default rule set is active with only
`exported` and `package-comments` excluded, so `unused-parameter` fires. Never add a
store parameter in a commit whose body does not yet use it.

Slices, each landing separately with green goldens:

```
[x] 2b.1 Foundation (`152ac5a`) (db.go, 4). db.Open(cfg) (*db.Store, error) replaces db.Init;
    the seven AutoMigrate panics and InitSettings' log.Panicf now return. Migrate
    stops at the first failure naming the table rather than joining, because the
    targets are ordered parent-first and a cascade would bury the real error.
    app.App holds the store. Also converted three db.Init callers, not the two
    planned: app/app.go:45, api/golden_test.go:105, and
    middleware/authentication_middleware_test.go:143.
    Fixed while here: Migrate reached InitSettings and GetEmbeddingModel through
    the package global, so NewStoreFrom(h).Migrate() seeded settings into whatever
    DB pointed at rather than h. Invisible while Open was the only caller. Both are
    now store-scoped; regression test proven against the reverted fix via
    go test -overlay.
    13 tests in internal/db/store_test.go.

[x] 2b.2 Users and settings (`dbd019c`) (user.go 4, setting.go 6). The pilot.
    Sets the two patterns the rest of 2b reuses: per-aggregate repositories off
    Store (store.Users(), store.Settings()) rather than ~95 flat methods, and
    handler closures - func CreateUser(*db.Store) gin.HandlerFunc.
    setting.go held 6 functions, not the 5 counted at ca8a758: 2b.1 had since
    split InitSettings into a Store method and added Store.EmbeddingModel.
    ExistsUsername -> UserRepo.Exists (bool, error). The "username already
    exists" sentinel moved up to services.ErrUsernameTaken, so the 500 and the
    body are unchanged; Phase 6 still owns the 409.
    Deleted db.GetValue - zero callers. It was the only reader of
    Setting.SettingType, which is now written but never read; the column stays
    because dropping a persisted one is a migration, not a refactor.
    Verified: AutoMigrate really does emit the UNIQUE constraint on
    User.Username, so the schema backstops Exists. Nothing had asserted that.
    Reached one hop further than planned: enqueueUnanalyzedRecordings also takes
    the store, since it is where the SetEmbeddingModel call actually lives.

[ ] 2b.3 Jobs (job.go 26). Plus jobs/lifecycle.go:47 and jobs/executor.go:25.
    Reuse the existing seam: handlers.NewHandlerDependencies takes a raw *gorm.DB
    at jobs/handlers/dependencies.go:15; change it to *db.Store and the six job
    handlers follow. Four generic functions in job.go stay free functions - Go has
    no generic methods. CreateJob[T] at :372 collides with (job *Job) CreateJob()
    at :75 once the receiver is gone; rename the method to JobRepo.Create.

[ ] 2b.4 Recordings (recording.go 19, video_analysis.go 5, video_previews.go 4).
    Plus services/video_analysis.go:194. Delete the init() at
    services/video_analysis.go:22 - it registers AnalyzeVideoFramesWithJob into a
    func(*db.Job) error field at package-init time, when no store exists; register
    from app.InitializeApp instead. Collapse the duplicate FindRecordingByID
    (function form at videos.go:440, method form at :94, :124, :195, :332, :468).

[ ] 2b.5 Channels (channel.go 7, channel_id.go 11). Plus
    services/startup_service.go:141 and recorder_service.go. 12 routes in the
    /channels group. api/v1/channels.go reaches db only for types; those
    db.ChannelID(id) conversions stay. channel.go:147 is the 8th deferred
    config.Read() and lands here.

[ ] 2b.6 Frame vectors (frame_vectors.go 9). store/vector/store.go is a pure
    adapter: 8 one-to-one delegations plus the reach-through at :72 and :76.
    NewSQLiteVecStore gains the store. Then delete var DB and convert the 11 lines
    in services/chapter_regeneration_test.go. vector.SetDefault stays - Phase 3.

[ ] 2b.7 Path methods and the 7 remaining deferred config.Read() calls. 49
    external callers; extract a Paths value from cfg rather than hanging it off
    Store, since these are filesystem concerns. No overlap with 2b.1-2b.6, so it
    can ship as its own commit or as Phase 2c without blocking the gate.
```

Gate, arithmetic rather than judgement. All three commands run today:

```
grep -rn 'db\.DB\b' server/internal --include='*.go' | grep -v '/internal/db/' | wc -l
    17 at ca8a758 (6 non-test + 11 in chapter_regeneration_test.go)  ->  0
    Still 17 after 2b.2, which touches none of those sites.

grep -rn 'func (\w* \*\?\(Recording\|Channel\|Job\|Setting\|VideoPreview\|VideoAnalysisResult\|ChannelID\|RecordingID\)) ' \
     server/internal/db --include='*.go' | wc -l
    47 at ca8a758  ->  46 after 2b.2  ->  15 after 2b.6  ->  11 after 2b.7

Both greps are textual, so keep the deleted names out of prose too: a comment
naming db.DB or a converted function inflates the count and reads as a leftover.
```

47 - 32 = 15 cross-checks the 32: the survivors are exactly the methods that do no
persistence. Scope the first grep to server/internal as written; widening it to
server picks up two hits inside vendored gorm.

## Phase 3 - Services become injected structs

Objective: convert free functions to constructor-injected structs and move mutable
package state into fields.

```
[ ] Convert the 49 exported functions in internal/services to methods on injected
    structs.

[ ] Move 9 clusters of mutable package state into struct fields. Riskiest single
    item: streaming_service.go holds live recording state in three package maps
    plus two mutexes. Move it alone, with -race.

[ ] Fix a known bug: a detector factory caches on first call and thereafter
    ignores its detectorType argument. Corrected 2026-08-03 - this was filed
    against the wrong function. CreateSceneDetector (factory.go:42) has zero
    callers and should just be deleted. The live instances are
    CreateEmbeddingExtractor (factory.go:67), called from
    services/visual_similarity.go:51 and services/video_analysis.go:64, and
    CreateHighlightDetector (factory.go:98). Harmless today only because one
    detector type exists. The package has no test file at all; add the test that
    would have caught it.

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

[-] Delete util.ParseNumbers and util.FileNameWithoutExtension. Rejected
    2026-08-03: the "no consumers" claim was wrong. ParseNumbers has 4 callers at
    util/sys.go:314-317 and FileNameWithoutExtension has 2 at util/video.go:258
    and :300. Both deletions would break the build. They move with their packages
    in the split above instead.

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

[~] A duplicate signup returns 500 where 409 is correct. The signature half is
    done in 2b.2: UserRepo.Exists answers (bool, error), so "name taken" and
    "database unavailable" are now distinguishable, and the sentinel lives in
    services.ErrUsernameTaken. The status code is unchanged and still wrong.
    Now pinned by internal/api/testdata/public_auth.golden, so the Phase 6
    change to 409 will show as a reviewable diff. Owner: Phase 6.

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
    so every gate is run manually. docker-test.sh is now the piece that was
    missing: it is self-contained and environment-independent, so it can be
    invoked from a workflow as-is.

[ ] docker-test.sh and lint.sh cover the Go module only. The frontend, CLI and
    mobile test and lint targets exist but are not wired into a single entry
    point.

[ ] docker-test.sh needs the Docker daemon and a first build of roughly five
    minutes. There is no host-only fast path any more; `cd server && GOWORK=off
    go test ./...` is the fallback, but it cannot boot the server.
```

---

# Maintaining this file

- Update it at the end of each phase, not in between.
- Re-stamp "Last verified" with the date and commit whenever numbers are re-measured.
  Numbers here are snapshots and go stale silently otherwise.
- When a phase completes, record its commit SHA in the heading and check its boxes.
- Use `[~]` honestly. If a phase lands with a gap, name the gap rather than marking it
  `[x]`.
