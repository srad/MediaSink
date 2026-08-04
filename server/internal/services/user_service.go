package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/srad/mediasink/server/internal/db"
	"github.com/srad/mediasink/server/internal/models/requests"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// ErrUsernameTaken is the wire-visible duplicate-signup error. The store answers
// (bool, error), so "name taken" and "database unavailable" are distinguishable; this is
// where the message the caller renders lives.
//
// The text is the response body verbatim - app.Gin.Error JSON-encodes err.Error() - and
// internal/api/testdata/public_auth.golden pins it together with the 500 status. The API
// redesign phase changes that status to 409.
var ErrUsernameTaken = errors.New("username already exists")

// userStore is the slice of the user aggregate this service actually uses. Declared here
// rather than in internal/db, per ARCHITECTURE.md section 2.
type userStore interface {
	Exists(ctx context.Context, name string) (bool, error)
	Create(ctx context.Context, u *db.User) error
	ByUsername(ctx context.Context, name string) (*db.User, error)
	ByID(ctx context.Context, id uint) (*db.User, error)
}

// Compile-time proof that the real store satisfies what this service needs. Without it
// the mismatch still fails the build, but at the wiring line rather than here, where the
// requirement is stated.
var _ userStore = (*db.UserStore)(nil)

// UserService owns signup and authentication.
type UserService struct {
	users userStore
	// jwtSecret signs the tokens this service issues. Held rather than passed per
	// call: it is configuration, not per-request state.
	jwtSecret string
}

func NewUserService(users userStore, jwtSecret string) *UserService {
	return &UserService{users: users, jwtSecret: jwtSecret}
}

func (s *UserService) CreateUser(ctx context.Context, auth requests.AuthenticationRequest) error {
	taken, err := s.users.Exists(ctx, auth.Username)
	if err != nil {
		return err
	}
	if taken {
		return ErrUsernameTaken
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(auth.Password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	return s.users.Create(ctx, &db.User{
		Username: auth.Username,
		Password: string(passwordHash),
	})
}

// Authenticate returns a JWT string if the authentication was successful.
func (s *UserService) Authenticate(ctx context.Context, auth requests.AuthenticationRequest) (string, error) {
	user, errUser := s.users.ByUsername(ctx, auth.Username)

	if errors.Is(errUser, gorm.ErrRecordNotFound) {
		return "", errors.New("user not found")
	}

	if errUser != nil {
		return "", errUser
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(auth.Password)); err != nil {
		return "", err
	}

	generateToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":  user.UserID,
		"exp": time.Now().Add(time.Hour * 24).Unix(),
	})

	return generateToken.SignedString([]byte(s.jwtSecret))
}

func (s *UserService) ByID(ctx context.Context, userID uint) (*db.User, error) {
	return s.users.ByID(ctx, userID)
}
