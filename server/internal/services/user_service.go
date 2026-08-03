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

func CreateUser(auth requests.AuthenticationRequest) error {
	if err := db.ExistsUsername(auth.Username); err != nil {
		return err
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(auth.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user := &db.User{
		Username: auth.Username,
		Password: string(passwordHash),
	}

	return db.CreateUser(user)
}

// AuthenticateUser Returns a JWT string if the authentication was successful.
// The signing secret is a parameter rather than an environment read so the caller
// controls it; see config.Cfg.JWTSecret.
func AuthenticateUser(auth requests.AuthenticationRequest, secret string) (string, error) {
	user, errUser := db.FindUserByUsername(auth.Username)

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

func GetUserByID(userID uint) (*db.User, error) {
	return db.FindUserByID(userID)
}
