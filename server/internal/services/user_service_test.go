package services

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/golang-jwt/jwt/v4"
	"github.com/srad/mediasink/server/config"
	"github.com/srad/mediasink/server/internal/db"
	"github.com/srad/mediasink/server/internal/models/requests"
	"golang.org/x/crypto/bcrypt"
)

const testSecret = "user-service-test-secret"

// userStore opens a throwaway database and returns the store the functions under test
// are handed. Note that nothing here assigns the package-level handle: these tests
// reach the database only through the value they pass in, which is the point of the
// slice.
func userStore(t *testing.T) *db.Store {
	t.Helper()

	store, err := db.Open(config.Cfg{DbFileName: filepath.Join(t.TempDir(), "user_service.db")})
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	return store
}

func creds(username, password string) requests.AuthenticationRequest {
	return requests.AuthenticationRequest{Username: username, Password: password}
}

// The message is the response body verbatim: app.Gin.Error JSON-encodes err.Error(),
// and internal/api/testdata/public_auth.golden pins the same bytes. It used to come
// from the store's username-existence check; if the wording drifts here, that golden
// fails.
func TestCreateUserRejectsATakenUsername(t *testing.T) {
	store := userStore(t)

	if err := CreateUser(store, creds("taken@example.com", "first-password")); err != nil {
		t.Fatalf("first CreateUser: %v", err)
	}

	err := CreateUser(store, creds("taken@example.com", "second-password"))
	if !errors.Is(err, ErrUsernameTaken) {
		t.Fatalf("CreateUser on a taken username = %v, want ErrUsernameTaken", err)
	}
	if err.Error() != "username already exists" {
		t.Errorf("error text = %q, want %q — public_auth.golden pins this string",
			err.Error(), "username already exists")
	}
}

func TestCreateUserStoresABcryptHashNotThePlaintext(t *testing.T) {
	store := userStore(t)

	if err := CreateUser(store, creds("hashed@example.com", "plaintext-password")); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}

	user, err := store.Users().ByUsername("hashed@example.com")
	if err != nil {
		t.Fatalf("ByUsername: %v", err)
	}
	if user.Password == "plaintext-password" {
		t.Fatal("the password was stored in plaintext")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte("plaintext-password")); err != nil {
		t.Errorf("stored value is not a bcrypt hash of the password: %v", err)
	}
}

func TestAuthenticateUserReturnsATokenBoundToTheUserAndSecret(t *testing.T) {
	store := userStore(t)

	if err := CreateUser(store, creds("login@example.com", "good-password")); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	seeded, err := store.Users().ByUsername("login@example.com")
	if err != nil {
		t.Fatalf("ByUsername: %v", err)
	}

	tokenString, err := AuthenticateUser(store, creds("login@example.com", "good-password"), testSecret)
	if err != nil {
		t.Fatalf("AuthenticateUser: %v", err)
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

	// The secret is a parameter, so a token signed with one must not verify under
	// another. This is the unit-level half of what docker-test.sh proves by restarting
	// the server under a different SECRET.
	if _, err = jwt.Parse(tokenString, func(*jwt.Token) (any, error) {
		return []byte("a-different-secret"), nil
	}); err == nil {
		t.Error("the token verified under a secret it was not signed with")
	}
}

func TestAuthenticateUserRejectsBadCredentials(t *testing.T) {
	store := userStore(t)

	if err := CreateUser(store, creds("someone@example.com", "good-password")); err != nil {
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
			token, err := AuthenticateUser(store, test.auth, testSecret)
			if err == nil {
				t.Fatalf("AuthenticateUser succeeded, returning %q", token)
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

func TestGetUserByID(t *testing.T) {
	store := userStore(t)

	if err := CreateUser(store, creds("byid@example.com", "password")); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	seeded, err := store.Users().ByUsername("byid@example.com")
	if err != nil {
		t.Fatalf("ByUsername: %v", err)
	}

	got, err := GetUserByID(store, seeded.UserID)
	if err != nil {
		t.Fatalf("GetUserByID: %v", err)
	}
	if got.Username != "byid@example.com" {
		t.Errorf("GetUserByID = %+v, want %q", got, "byid@example.com")
	}

	if _, err = GetUserByID(store, seeded.UserID+1000); err == nil {
		t.Error("GetUserByID succeeded for an id that does not exist")
	}
}

// Every function here must read and write the store it is handed. db.Open still assigns
// the package global until phase 2b.6, so one that reached for it instead would pass
// the tests above while using the most recently opened database.
func TestUserServiceUsesTheStoreItIsGivenNotTheGlobal(t *testing.T) {
	first := userStore(t)
	second := userStore(t) // db.Open reassigns the global to this one.

	if err := CreateUser(first, creds("scoped@example.com", "password")); err != nil {
		t.Fatalf("CreateUser against the first store: %v", err)
	}

	if _, err := AuthenticateUser(second, creds("scoped@example.com", "password"), testSecret); err == nil {
		t.Error("a user created through the first store authenticated against the second")
	}
}
