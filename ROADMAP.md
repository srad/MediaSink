# Roadmap

Status tracking for MediaSink, so work in progress survives across sessions.

This file records *what is done and what is left*. It does not explain how the system
works or how to run it:

- How Go code must be structured, and the rules a change is held to: `ARCHITECTURE.md`
- Build/run/test commands and conventions: `AGENTS.md`
- Installation and user-facing setup: `README.md`

Last verified: 2026-08-04, at commit `7a696f6`.

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

Converts the Go server from package-level mutable state and free functions to
constructor-injected structs, as specified in `ARCHITECTURE.md`. Every phase must build,
pass tests and lint, and leave the golden HTTP tests byte-identical unless an API change
is intended.

Measured at `7a696f6`: coverage 40.8%, lint 0 issues, 110 golden route cases.

## Phase numbering was reset on 2026-08-04

The original plan split the work by layer: inject the database everywhere (2b), then turn
services into structs (3), then handlers (4). That order is unworkable. A free function
can only receive a dependency as an argument, so injecting first forces every call site
to pass a store per call, and the later phases would delete every parameter the earlier
one added.

Phases are now one per aggregate, each finishing store, service, handler and wiring
together. Nothing needs a temporary parameter, because by the time a service calls its
store the service is already a struct holding it.

Mapping from the old numbering, so nothing looks dropped:

```
old 0, 1, 2a  ->  new 0, 1, 2, unchanged and still done
old 2b        ->  new 3, marked partial and superseded
old 3         ->  absorbed into every slice from 5 on (services become structs)
old 4         ->  absorbed into every slice from 5 on (handlers become structs),
                  except the composition root itself, which is new 10
old 5         ->  new 11
old 6         ->  new 12
```

## Phase 0 - Safety net and tooling (`712c7eb`)

```
[x] Golden HTTP tests: boot the real router, snapshot status and body per route.

[~] Route coverage. 110 cases across 4 golden files, but not everything:
    [ ] /api/v2/ws has no golden coverage at all. A websocket upgrade cannot be
        driven through httptest's recorder. Needs a different harness.
    59 routes are registered in router.go against 56 auth-gate cases. The
    difference is /ws plus the two public auth routes, which public_auth.golden
    covers directly.
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

[x] test.sh fixes, later superseded: test.sh was deleted in favour of
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

## Phase 2 - Config becomes a value, not a global (`f2929a2`)

```
[x] Cfg is read once at the composition root and passed to constructors. This is
    exactly the shape ARCHITECTURE.md requires of everything else, so it needs no
    rework and keeps its tick.

[x] Folded in all 9 os.Getenv calls that bypassed config, plus the two LOG_LEVEL
    reads in main.go. The JWT secret is captured once at router construction
    rather than read from the environment on every authenticated request.

[x] mustEnv is deleted. config.Parse(getenv) is pure and returns an error naming
    every missing variable at once; config.Load() caches it.

[~] 11 config.Read() call sites remain, each inside code that has not been
    converted yet. They go with their aggregate:
    [ ] 8 in internal/db, on ChannelName and RecordingID path methods -> phase 8
    [ ] 3 in internal/services free functions                        -> phases 6-9
    The config.Read() shim still calls log.Panicf; it goes with the last of them.

[x] Fixed an environment-dependent golden. authenticated.golden pinned the string
    `exec: "yt-dlp": executable file not found in $PATH`, which is only true on a
    host WITHOUT yt-dlp, so the suite passed by accident of a missing dependency
    and could never have run in CI. The seeded channel now points at a test-local
    httptest.Server.

[x] Added test.Dockerfile and docker-test.sh: the Go suite plus a real boot smoke
    test (signup, login, authenticated route, restart under a different SECRET to
    prove the JWT is bound to config, and the missing-config abort).
```

## Phase 3 - Kill the global DB handle (`152ac5a`, `dbd019c`, superseded)

Objective was to replace `var DB *gorm.DB` with an injected store. It landed, and the
result is the wrong design.

```
[~] Landed, but procedurally. db.Store was introduced and threaded to consumers
    as a per-call function parameter:

        func CreateUser(store *db.Store, auth requests.AuthenticationRequest) error
        func RequireAuth(store *db.Store, secret string) gin.HandlerFunc

    Five things wrong with it, all fixed by ARCHITECTURE.md:
    [ ] the dependency is a parameter rather than a field on a struct
    [ ] db.Store is a god object - Users(), Settings(), Jobs() on one type hands
        the auth code the job and video tables as well
    [ ] no interfaces, so nothing above the store is testable without real SQLite
    [ ] no context.Context anywhere in the server
    [ ] side effects hard-wired: three functions call util.Interrupt directly

    Superseded by phases 4-10. Both commits stay in history; the code is
    refactored forward, not reverted. Do not copy this pattern.

[x] Worth keeping from those commits, and carried forward unchanged:
    db.Open returns errors instead of panicking; Migrate names the failing table;
    public_auth.golden and its 4 signup/login cases; 17 tests across db,
    services and middleware; ExistsUsername answering (bool, error) with the
    sentinel moved to services.ErrUsernameTaken.
```

## Phase 4 - Foundation

Objective: the rules, and a concrete database handle to build stores on.

```
[x] ARCHITECTURE.md, with a pointer from AGENTS.md.

[ ] db.Open returns a concrete *db.Handle wrapping the gorm connection:

        type Handle struct{ gorm *gorm.DB }
        func Open(cfg config.Cfg) (*Handle, error)
        func (h *Handle) Gorm() *gorm.DB
        func (h *Handle) Migrate() error
        func (h *Handle) Ping() error
        func (h *Handle) Close() error

    Named Handle, not DB: the global being deleted is `var DB *gorm.DB` at
    internal/db/db.go:13, so `type DB` would collide with it for the whole of
    phases 4-9 while both exist.

    No connection interface. "Return concrete types" is the same rule that
    settles the interface question in ARCHITECTURE.md; if something later needs
    to substitute the connection, that consumer declares the interface.
    internal/store/vector needs the raw *sql.DB and reaches it via Gorm().DB().
```

Gate: builds, goldens byte-identical, lint 0.

## Phase 5 - Users and settings

Objective: redo what `152ac5a`/`dbd019c` did, in the shape ARCHITECTURE.md defines. This
is the pattern-setting slice; every later phase copies its structure.

```
[ ] internal/db: UserStore and SettingStore, concrete, ctx on every method.
[ ] internal/services: UserService holding its own narrow userStore interface.
[ ] internal/api/v1: AuthHandler{users, jwtSecret} with CreateUser/Login/Logout
    as methods.
[ ] internal/middleware: AuthMiddleware{users, jwtSecret}.
[ ] internal/api/router.go takes constructed handlers; app/app.go constructs them.
[ ] Delete db.Store.Users() and db.Store.Settings().
```

Two risks to retire before converting the other 57 handlers:

```
[ ] The golden harness must build the object graph. golden_test.go:111 calls
    Setup(store, cfg, ...). Once handlers are structs, TestMain has to assemble
    stores, services and handlers - a second composition root that must not drift
    from app/app.go. Extract one exported builder used by both.

[ ] Prove swagger survives handlers becoming methods BEFORE converting all 58.
    Convert exactly one, run swag init, confirm docs/swagger.json is
    byte-identical. swag parses comments above methods as well as functions, but
    this is unverified here - the reference branch commits no generated docs.
```

Gate: goldens byte-identical, swagger byte-identical, docker-test.sh green.

## Phase 6 - Jobs

Objective: the job engine off global state - the only code that runs continuously against
the database.

Findings already established by reading and running the tree, before any code moves:

```
[ ] handlers.HandlerDependencies.DB is dead - assigned at construction, never
    read by any handler. conversion_test.go already passes nil for it. The
    roadmap previously called this the seam to reuse; it is not one.
[ ] Job.Cancel and GetNextJobTask[T] have zero callers. Delete, do not convert.
[ ] db.DeleteJob does not delete. It sets status=canceled and keeps the row.
[ ] JobList with an empty status slice matches nothing, not everything - and
    authenticated.golden posts exactly that.
[ ] EmitJobProgress stores "NaN" when total is 0 (no division guard).
[ ] Three functions call util.Interrupt on a job pid: updateStatus, DeleteJob,
    PurgeJobsByTask. Inject the seam per ARCHITECTURE.md section 6.
[ ] internal/db/job.go broadcasts websocket events at :446, :484, :498. The
    broadcast moves up to the service; internal/db must import no ws.
[ ] internal/jobs holds package state in types.go: ctxJobs, cancelJobs,
    processing, processingMutex. All become fields.
[ ] services/job_service.go has four pass-through wrappers whose own comments
    call them backward-compatibility shims. Delete them.
```

Gate: `-race` clean on internal/jobs and internal/db.

## Phase 7 - Recordings

```
[ ] recording.go, video_analysis.go, video_previews.go into stores.
[ ] Collapse the duplicate FindRecordingByID (function form at videos.go:440,
    method form at :94, :124, :195, :332, :468).
[ ] services/video_analysis.go registers a job handler from init() at :22.
    Register from the composition root instead.
[ ] The Recording.Enqueue*Job family moves here with recording.go.
```

## Phase 8 - Channels

```
[ ] channel.go and channel_id.go into stores; 12 routes in the /channels group.
[ ] services/startup_service.go and recorder_service.go.
[ ] The 8 deferred config.Read() calls on ChannelName and RecordingID path
    methods land here. Extract a Paths value from cfg rather than hanging it off
    a store - these are filesystem concerns, not persistence.
```

## Phase 9 - Frame vectors and video analysis

```
[ ] frame_vectors.go into a store. store/vector/store.go is a pure adapter:
    8 one-to-one delegations plus the reach-through at :72 and :76.
[ ] Delete the second global: vector.defaultStore behind SetDefault/Default,
    with lazy init at store.go:63. 7 call sites - app/app.go:52,
    api/v1/similarity.go:203, services/visual_similarity.go x3,
    services/startup_service.go:166, services/video_analysis.go:54. All seven
    currently pass context.Background(); real ctx replaces it.
[ ] Delete var DB. Convert the 11 db.DB lines in
    services/chapter_regeneration_test.go.
```

Gate: `grep -rn 'db\.DB\b' server/internal | grep -v '/internal/db/'` returns 0.

## Phase 10 - Composition root

```
[ ] Roughly 45 lines of plain constructor calls in app/. No DI framework.
[ ] Delete db.Store entirely.
[ ] Take the goroutine out of the router constructor. Setup() currently runs
    `go ws.WsListen()`, so merely building a router starts background work.
[ ] Return *gin.Engine from Setup instead of http.Handler, which app.go
    immediately type-asserts back.
[ ] Add internal/httperr with a typed error and a single renderer. Keep today's
    bare-string wire format; changing it is phase 12.
[ ] Fix a known bug: CreateEmbeddingExtractor (factory.go:67) and
    CreateHighlightDetector (factory.go:98) cache on first call and thereafter
    ignore their detectorType argument. Harmless today only because one detector
    type exists. The package has no test file at all. CreateSceneDetector
    (factory.go:42) has zero callers and should just be deleted.
```

## Phase 11 - Break up internal/util and internal/models

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

## Phase 12 - API redesign

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
    Go usage moves. Owner: Phase 12. Excluded in server/.golangci.yml.

[>] revive's exported (22 findings) and package-comments (34) rules. Both are
    missing-documentation churn, not correctness. Owner: after the substantive
    findings are gone; drop both exclusions together.
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
    Re-examined 2026-08-04 when the phase-3 design was rejected: the wiring tool
    was never the problem. The code had no constructors to wire.

[-] Finishing the v2 migration instead of deleting it. A parallel tree that only
    goes live at the very end never goes live; this repo has two abandoned
    attempts as evidence. The live code is refactored in place instead.

[-] Declaring store interfaces in internal/db next to their implementations, as
    ~/src/MediaSink.Go branch refactor/idiomatic-go does. Google's style guide
    puts the interface in the consumer, listing only the methods it uses. A
    store-side interface would make the job equivalent roughly 20 methods, and
    every test fake would have to implement all 20 to exercise one. See
    ARCHITECTURE.md section 2.

[-] Adding testify. Not vendored, and this repo vendors all dependencies. Fakes
    of two-to-three-method consumer interfaces are short enough by hand.

[-] Keeping safeJoinPath. It was a path-traversal guard that was written but
    never wired to any call site. Deleted with the reasoning recorded at its old
    location in db/channel_name.go: validChannelName is ^[a-z_0-9]+$, which
    admits neither "." nor "/", so traversal cannot reach filepath.Join. If a
    future code path writes channel names by another route, restore a guard
    rather than trusting the regex alone.
```

# Known defects, not yet fixed

Found while auditing. All are client-visible, so they are grouped into Phase 12 rather
than fixed piecemeal.

```
[ ] Not-found is inconsistent across three endpoints. For an id that does not
    exist: GET /channels/1 returns 500, GET /videos/1 returns 200 with a
    zero-valued object, and GET /analysis/1 returns 200 with null fields. All
    three should be 404. The golden files record the current behaviour, so the
    fix will show as a reviewable diff.

[~] A duplicate signup returns 500 where 409 is correct. The signature half is
    done: UserStore.Exists answers (bool, error), so "name taken" and "database
    unavailable" are distinguishable, and the sentinel lives in
    services.ErrUsernameTaken. The status code is unchanged and still wrong.
    Pinned by internal/api/testdata/public_auth.golden, so the change to 409 will
    show as a reviewable diff. Owner: Phase 12.

[ ] The websocket auth workaround passes the bearer token as a URL query
    parameter, which leaks it into access logs and browser history. It is a
    deliberate, documented workaround; changing it is an API change.

[ ] EmitJobProgress stores the string "NaN" when total is 0 - no division guard
    at jobs/handlers/progress.go:17. Reachable from preview.go:107. Owner:
    Phase 6, pinned by test first.

[ ] db.DeleteJob does not delete; it cancels. The name has already misled once.
    Renaming it is wire-visible (the route is DELETE /jobs/:id). Owner: Phase 12.
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

# Architectural gates

Greps, not opinions. `ARCHITECTURE.md` section 12 lists the commands; the numbers below
are the current readings, measured at `7a696f6`.

```
a store or service passed as a function parameter      9  ->  0   (phases 5-10)
    router.go:37, api/v1/auth.go x2, services/user_service.go x3,
    services/startup_service.go x2, middleware/authentication_middleware.go:27

exported store methods missing ctx                     n/a ->  0   (phases 5-9)
    no *XStore types exist yet; the gate starts reading in phase 5

internal/db importing internal/ws                      1  ->  0   (phase 6)

db.DB referenced outside internal/db                  17  ->  0   (phase 9)
    6 non-test plus 11 in services/chapter_regeneration_test.go

vector.SetDefault / vector.Default() call sites         7  ->  0   (phase 9)
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
- Architecture rules belong in `ARCHITECTURE.md`, not here. This file records status and
  the current reading of each gate.
