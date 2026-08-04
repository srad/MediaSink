# Architecture

How the Go server is structured, and the rules every change is held to.

This file is the contract. `ROADMAP.md` tracks which parts of the tree already satisfy
it; `AGENTS.md` covers build and test commands. Where this file and existing code
disagree, the code is what is being fixed - check `ROADMAP.md` for which phase owns it.

Two sources shaped these rules:

- Google's Go Style Guide (`google.github.io/styleguide/go`), which governs the details.
- `~/src/MediaSink.Go`, branch `refactor/idiomatic-go`, an abandoned parallel rewrite
  whose layering is worth copying. Where the two disagree, **Google wins** - see
  "Interfaces" below for the one place this matters.

---

## 1. Layers

```
internal/api/v1   handlers, gin lives here and nowhere else
      |
internal/services business logic, no gin, no gorm
      |
internal/db       stores, gorm lives here and nowhere else
      |
  *gorm.DB
```

Dependencies point one way only.

- `internal/db` imports no service, no handler, no `gin`, and no `internal/ws`.
- `internal/services` imports no `gin` and no `gorm`.
- Only `app/` - the composition root - imports every layer.

A store persists rows. It does not broadcast, enqueue, or call the filesystem.

---

## 2. Interfaces: the consumer declares them

**The package that calls the methods writes the interface. The package that implements
them exports a concrete type.**

Google's `decisions.md`:

> The consumer of the interface should define it (not the package implementing the
> interface), ensuring it includes only the methods they actually use.

> Functions should take interfaces as arguments but return concrete types.

The store is a plain struct with no interface next to it:

```go
// internal/db/user.go
type UserStore struct{ gorm *gorm.DB }

func NewUserStore(g *gorm.DB) *UserStore { return &UserStore{gorm: g} }

// Exists reports whether the username is taken.
func (s *UserStore) Exists(ctx context.Context, name string) (bool, error) {
	var count int64
	if err := s.gorm.WithContext(ctx).
		Model(&User{}).
		Where("username = ?", name).
		Count(&count).Error; err != nil {
		return false, fmt.Errorf("count users named %q: %w", name, err)
	}
	return count > 0, nil
}
```

The service declares the short list it actually calls:

```go
// internal/services/user_service.go
type userStore interface {
	Exists(ctx context.Context, name string) (bool, error)
	Create(ctx context.Context, u *db.User) error
	ByUsername(ctx context.Context, name string) (*db.User, error)
}

// Compile-time proof that the real store satisfies what this service needs.
var _ userStore = (*db.UserStore)(nil)

type UserService struct{ users userStore }

func NewUserService(users userStore) *UserService {
	return &UserService{users: users}
}
```

**Why the `var _` line.** Without it the mismatch still fails the build - the compiler
checks it where `app/app.go` passes the store into the constructor. With it the error
appears in the file that *states* the requirement, next to the list it broke, instead of
in a wiring file far away.

**Why not one big `UserStorer` in `internal/db`.** The reference branch does that, and it
is the one place this codebase deliberately diverges. A store-side interface has to list
every method the type has, so the job equivalent would be roughly twenty. Every test fake
would then have to implement all twenty to test something that calls one. Consumer-side
lists are two or three methods, so a fake is a few lines. Do not "fix" this back.

Keep the interface unexported (`userStore`, not `UserStore`) - it is an implementation
detail of the consumer.

---

## 3. Dependencies are constructor arguments held as fields

A dependency is passed **once**, at construction, and stored on the struct. It is never a
parameter on the method that uses it.

```go
// Right
type UserService struct{ users userStore }

func (s *UserService) CreateUser(ctx context.Context, auth requests.AuthenticationRequest) error {
	taken, err := s.users.Exists(ctx, auth.Username)
	...
}

// Wrong - this is the shape the refactor exists to remove
func CreateUser(store *db.Store, auth requests.AuthenticationRequest) error
```

This applies to stores, services, config values, loggers and clocks alike.

`ctx` is the single exception: it is per-call state, so it goes in every signature that
does I/O, and it is never stored in a struct.

---

## 4. context.Context

Every method that touches the database, the network or a subprocess takes
`ctx context.Context` as its **first** parameter, and passes it down.

```go
func (s *UserStore) ByID(ctx context.Context, id uint) (*db.User, error)
func (s *UserService) CreateUser(ctx context.Context, auth requests.AuthenticationRequest) error
```

gorm receives it with `WithContext(ctx)`, which is what makes a query cancellable:

```go
s.gorm.WithContext(ctx).Model(&User{}).First(&user)
```

`ctx` enters the program in exactly two places:

- a handler, from `c.Request.Context()`
- a background worker, from the context its lifecycle owns

Never store a `Context` in a struct field. Never pass `context.Background()` from
somewhere that has a real one available.

---

## 5. Handlers are structs; gin stops at the controller

```go
// internal/api/v1/auth.go
type AuthHandler struct {
	users     *services.UserService
	jwtSecret string
}

func NewAuthHandler(users *services.UserService, jwtSecret string) *AuthHandler {
	return &AuthHandler{users: users, jwtSecret: jwtSecret}
}

// CreateUser godoc
// @Summary Create new user account
// @Router  /auth/signup [post]
func (h *AuthHandler) CreateUser(c *gin.Context) {
	var auth requests.AuthenticationRequest
	if err := c.BindJSON(&auth); err != nil {
		appG.Error(http.StatusBadRequest, err)
		return
	}
	if err := h.users.CreateUser(c.Request.Context(), auth); err != nil {
		...
	}
}
```

A handler decodes the request, calls one service method, and renders the result. No
database access, no business rules. `*gin.Context` never travels below this layer.

Swagger annotations sit above the method exactly as they sat above the function. `swag`
parses comments above any function declaration, method or not, so this should be
transparent - but it is **not yet proven in this repository**. Phase 5 converts one
handler and checks `docs/swagger.json` is byte-identical before the rest follow.

---

## 6. Side effects that touch the OS are injected

Anything that signals a process, shells out, reads the clock or touches the filesystem is
a struct field holding a function, so a test can substitute it.

```go
type JobStore struct {
	gorm      *gorm.DB
	interrupt func(pid int) error // util.Interrupt in production
}

func NewJobStore(g *gorm.DB) *JobStore {
	return &JobStore{gorm: g, interrupt: util.Interrupt}
}
```

The rule exists because of a real hazard: three functions in `internal/db/job.go` call
`util.Interrupt(*job.Pid)` directly, so any test seeding a job that carries a pid sends a
signal to whatever process on the machine currently holds that number.

---

## 7. No package-level mutable state, no init() with side effects

Package-level `var`s are for constants and sentinel errors. Anything mutable belongs on a
struct owned by the composition root.

Two offenders exist today, both scheduled in `ROADMAP.md`:

- `internal/jobs/types.go` - `ctxJobs`, `cancelJobs`, `processing`, `processingMutex`
- `internal/store/vector/store.go` - `defaultStore` behind `SetDefault`/`Default`

`init()` must not register handlers, open connections or start goroutines.
`internal/services/video_analysis.go` registers a job handler from `init()`; that becomes
an explicit call from the composition root.

---

## 8. Errors

Wrap with `%w` and a verb naming the operation that failed:

```go
return fmt.Errorf("count users named %q: %w", name, err)
```

- Error strings start lowercase and have no trailing punctuation.
- Sentinel errors for anything a caller branches on: `services.ErrUsernameTaken`.
  Compare with `errors.Is`, never by string.
- Never return an error to mean a boolean. `ExistsUsername` returned an error to mean
  "taken", which left callers unable to tell a duplicate name from a dead database - the
  reason a duplicate signup still answers 500 instead of 409.
- Handle or return; do not log and return the same error.

---

## 9. Tests

Standard library `testing`. testify is **not** vendored, and this repo vendors all
dependencies, so introducing it is a separate decision - do not add it in passing.

| Layer | What the test uses |
|---|---|
| `internal/db` | a real temp-file SQLite through `db.Open`, exercising real migrations |
| `internal/services` | a hand-written fake of the consumer interface - no database |
| `internal/api/v1` | the golden HTTP tests, plus fakes for anything else |

Fakes are hand-written structs, not generated mocks; there is no mock library here.
Because consumer interfaces are two or three methods, a fake is short:

```go
type fakeUserStore struct {
	exists  bool
	created *db.User
}

func (f *fakeUserStore) Exists(context.Context, string) (bool, error) { return f.exists, nil }
func (f *fakeUserStore) Create(_ context.Context, u *db.User) error   { f.created = u; return nil }
```

Every change needs a test. The golden files under `internal/api/testdata` are the
behavioural contract for the whole route table: during a refactor they must come out
byte-identical, and a diff is a bug until proven otherwise.

---

## 10. Naming

| Thing | Form | Example |
|---|---|---|
| Store type | `XStore` | `UserStore`, `JobStore` |
| Store constructor | `NewXStore` | `NewUserStore` |
| Service type | `XService` | `UserService` |
| Consumer interface | unexported, in the consumer | `userStore` |
| Receiver | one or two letters, consistent per type | `func (s *UserStore)` |

- No `Get` prefix on accessors: `ByID`, not `GetByID`.
- No `Repo`, no `Manager`, no `Helper`, no `Util`.
- No stutter: `db.UserStore`, never `db.DBUserStore`.
- Package names are short, lowercase, no underscores, no plurals.
- Doc comments start with the name of the thing: `// Exists reports whether ...`.

---

## 11. Composition root

`app/` constructs everything, once, in plain Go:

```go
handle, err := db.Open(cfg)
if err != nil {
	return nil, err
}

userStore := db.NewUserStore(handle.Gorm())
userService := services.NewUserService(userStore)
authHandler := v1.NewAuthHandler(userService, cfg.JWTSecret)
```

No DI framework. `google/wire` was evaluated and rejected - it was archived upstream in
August 2025 - and the runtime containers (`fx`, `dig`, `do`) move wiring errors from
compile time to startup, which is a regression against what the compiler already gives.
See `ROADMAP.md` for the full reasoning. At roughly thirty nodes a hand-written root is
smaller than the configuration a framework would need; revisit past about fifty.

The test harness must not build a second, drifting copy of this graph. It calls the same
exported builder the server does.

---

## 12. Checks

These are greps, not opinions. `ROADMAP.md` records the current number for each and the
phase that drives it to zero.

```sh
# A store or service must never be a parameter - only New* constructors may match.
grep -rnE 'func [A-Za-z]+\([^)]*\*(db\.)?[A-Za-z]*(Store|Service)\b' server/internal \
  --include='*.go' | grep -v 'func New' | grep -v '_test.go'

# Every exported store method takes a ctx.
grep -rnE 'func \([a-z]+ \*[A-Za-z]+Store\) [A-Z][A-Za-z]*\(' server/internal/db \
  --include='*.go' | grep -v 'ctx context\.Context'

# Stores do not notify.
grep -rn 'internal/ws' server/internal/db --include='*.go'

# No global database handle.
grep -rn 'db\.DB\b' server/internal --include='*.go' | grep -v '/internal/db/'

# No global vector store.
grep -rn 'vector\.SetDefault\|vector\.Default()' server --include='*.go' | grep -v vendor/
```
