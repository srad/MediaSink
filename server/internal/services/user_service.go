package services

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/srad/mediasink/server/internal/db"
	"github.com/srad/mediasink/server/internal/models/requests"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// ErrUsernameTaken is the wire-visible duplicate-signup error. It used to live in the
// store's username-existence check, which returned it in place of a boolean; that check
// now answers (bool, error), and this is where the message it produced is preserved.
//
// The text is the response body verbatim — app.Gin.Error JSON-encodes err.Error() —
// and internal/api/testdata/public_auth.golden pins it together with the 500 status.
// Phase 6 changes that status to 409.
var ErrUsernameTaken = errors.New("username already exists")

func CreateUser(store *db.Store, auth requests.AuthenticationRequest) error {
	taken, err := store.Users().Exists(auth.Username)
	if err != nil {
		return err
	}
	if taken {
		return ErrUsernameTaken
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(auth.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user := &db.User{
		Username: auth.Username,
		Password: string(passwordHash),
	}

	return store.Users().Create(user)
}

// AuthenticateUser Returns a JWT string if the authentication was successful.
// The signing secret is a parameter rather than an environment read so the caller
// controls it; see config.Cfg.JWTSecret.
func AuthenticateUser(store *db.Store, auth requests.AuthenticationRequest, secret string) (string, error) {
	user, errUser := store.Users().ByUsername(auth.Username)

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

	return generateToken.SignedString([]byte(secret))
}

func GetUserByID(store *db.Store, userID uint) (*db.User, error) {
	return store.Users().ByID(userID)
}
