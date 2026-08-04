package services

import (
	"context"
	"errors"
	"testing"

	"github.com/golang-jwt/jwt/v4"
	"github.com/srad/mediasink/server/internal/db"
	"github.com/srad/mediasink/server/internal/models/requests"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const testSecret = "user-service-test-secret"

// fakeUserStore is a hand-written double for the three-method interface UserService
// declares. No SQLite: the store's own behaviour is covered by internal/db's tests, and
// repeating it here would only make these slower.
type fakeUserStore struct {
	byName map[string]*db.User
	nextID uint

	existsErr error
	createErr error
	findErr   error
}

func newFakeUserStore() *fakeUserStore {
	return &fakeUserStore{byName: map[string]*db.User{}, nextID: 1}
}

func (f *fakeUserStore) Exists(_ context.Context, name string) (bool, error) {
	if f.existsErr != nil {
		return false, f.existsErr
	}
	_, ok := f.byName[name]
	return ok, nil
}

func (f *fakeUserStore) Create(_ context.Context, u *db.User) error {
	if f.createErr != nil {
		return f.createErr
	}
	u.UserID = f.nextID
	f.nextID++
	f.byName[u.Username] = u
	return nil
}

func (f *fakeUserStore) ByUsername(_ context.Context, name string) (*db.User, error) {
	if f.findErr != nil {
		return nil, f.findErr
	}
	u, ok := f.byName[name]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return u, nil
}

func (f *fakeUserStore) ByID(_ context.Context, id uint) (*db.User, error) {
	if f.findErr != nil {
		return nil, f.findErr
	}
	for _, u := range f.byName {
		if u.UserID == id {
			return u, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func newService(t *testing.T) (*UserService, *fakeUserStore) {
	t.Helper()
	store := newFakeUserStore()
	return NewUserService(store, testSecret), store
}

func creds(username, password string) requests.AuthenticationRequest {
	return requests.AuthenticationRequest{Username: username, Password: password}
}

// The message is the response body verbatim: app.Gin.Error JSON-encodes err.Error(),
// and internal/api/testdata/public_auth.golden pins the same bytes. If the wording
// drifts here, that golden fails.
func TestCreateUserRejectsATakenUsername(t *testing.T) {
	svc, _ := newService(t)

	if err := svc.CreateUser(t.Context(), creds("taken@example.com", "first-password")); err != nil {
		t.Fatalf("first CreateUser: %v", err)
	}

	err := svc.CreateUser(t.Context(), creds("taken@example.com", "second-password"))
	if !errors.Is(err, ErrUsernameTaken) {
		t.Fatalf("CreateUser on a taken username = %v, want ErrUsernameTaken", err)
	}
	if err.Error() != "username already exists" {
		t.Errorf("error text = %q, want %q — public_auth.golden pins this string",
			err.Error(), "username already exists")
	}
}

// A database failure must not be reported as "name taken". That distinction is the
// whole reason Exists answers (bool, error).
func TestCreateUserPropagatesAStoreFailure(t *testing.T) {
	svc, store := newService(t)
	store.existsErr = errors.New("database unavailable")

	err := svc.CreateUser(t.Context(), creds("someone@example.com", "password"))
	if err == nil {
		t.Fatal("CreateUser succeeded despite the store failing")
	}
	if errors.Is(err, ErrUsernameTaken) {
		t.Error("a store failure was reported as a taken username")
	}
}

func TestCreateUserStoresABcryptHashNotThePlaintext(t *testing.T) {
	svc, store := newService(t)

	if err := svc.CreateUser(t.Context(), creds("hashed@example.com", "plaintext-password")); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	user := store.byName["hashed@example.com"]
	if user == nil {
		t.Fatal("CreateUser did not reach the store")
	}
	if user.Password == "plaintext-password" {
		t.Fatal("the password was stored in plaintext")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte("plaintext-password")); err != nil {
		t.Errorf("stored value is not a bcrypt hash of the password: %v", err)
	}
}

func TestAuthenticateReturnsATokenBoundToTheUserAndSecret(t *testing.T) {
	svc, store := newService(t)

	if err := svc.CreateUser(t.Context(), creds("login@example.com", "good-password")); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	seeded := store.byName["login@example.com"]

	tokenString, err := svc.Authenticate(t.Context(), creds("login@example.com", "good-password"))
	if err != nil {
		t.Fatalf("Authenticate: %v", err)
	}

	claims := jwt.MapClaims{}
	if _, err = jwt.ParseWithClaims(tokenString, claims, func(*jwt.Token) (any, error) {
		return []byte(testSecret), nil
	}); err != nil {
		t.Fatalf("the token does not verify against the secret it was signed with: %v", err)
	}
	if id, ok := claims["id"].(float64); !ok || uint(id) != seeded.UserID {
		t.Errorf("token id claim = %v, want %d", claims["id"], seeded.UserID)
	}

	// The secret is a field on the service, so a token signed by one service must not
	// verify under another. This is the unit-level half of what docker-test.sh proves
	// by restarting the server under a different SECRET.
	if _, err = jwt.Parse(tokenString, func(*jwt.Token) (any, error) {
		return []byte("a-different-secret"), nil
	}); err == nil {
		t.Error("the token verified under a secret it was not signed with")
	}
}

func TestAuthenticateRejectsBadCredentials(t *testing.T) {
	svc, _ := newService(t)

	if err := svc.CreateUser(t.Context(), creds("someone@example.com", "good-password")); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	tests := []struct {
		name    string
		auth    requests.AuthenticationRequest
		wantErr string
	}{
		{
			name:    "wrong password",
			auth:    creds("someone@example.com", "wrong-password"),
			wantErr: "crypto/bcrypt: hashedPassword is not the hash of the given password",
		},
		{
			name:    "no such user",
			auth:    creds("nobody@example.com", "irrelevant"),
			wantErr: "user not found",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			token, err := svc.Authenticate(t.Context(), test.auth)
			if err == nil {
				t.Fatalf("Authenticate succeeded, returning %q", token)
			}
			// public_auth.golden pins both of these strings as response bodies.
			if err.Error() != test.wantErr {
				t.Errorf("error = %q, want %q", err.Error(), test.wantErr)
			}
			if token != "" {
				t.Errorf("token = %q on a failed authentication, want empty", token)
			}
		})
	}
}

func TestByID(t *testing.T) {
	svc, store := newService(t)

	if err := svc.CreateUser(t.Context(), creds("byid@example.com", "password")); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	seeded := store.byName["byid@example.com"]

	got, err := svc.ByID(t.Context(), seeded.UserID)
	if err != nil {
		t.Fatalf("ByID: %v", err)
	}
	if got.Username != "byid@example.com" {
		t.Errorf("ByID = %+v, want %q", got, "byid@example.com")
	}

	if _, err = svc.ByID(t.Context(), seeded.UserID+1000); err == nil {
		t.Error("ByID succeeded for an id that does not exist")
	}
}
