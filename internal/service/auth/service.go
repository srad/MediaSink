package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/srad/mediasink/internal/db"
	"github.com/srad/mediasink/internal/store"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type Service struct {
	users  store.UserStore
	secret []byte
}

func NewService(users store.UserStore, secret string) *Service {
	return &Service{
		users:  users,
		secret: []byte(secret),
	}
}

func (s *Service) CreateUser(ctx context.Context, username, password string) error {
	exists, err := s.users.ExistsUsername(ctx, username)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("username already exists")
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	return s.users.Create(ctx, &db.User{
		Username: username,
		Password: string(passwordHash),
	})
}

func (s *Service) Authenticate(ctx context.Context, username, password string) (string, error) {
	user, err := s.users.FindByUsername(ctx, username)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", errors.New("user not found")
	}
	if err != nil {
		return "", err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return "", err
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":  user.UserID,
		"exp": time.Now().Add(24 * time.Hour).Unix(),
	})
	return token.SignedString(s.secret)
}

func (s *Service) ParseToken(tokenString string) (uint, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.secret, nil
	})
	if err != nil {
		return 0, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return 0, errors.New("invalid token")
	}

	exp, ok := claims["exp"].(float64)
	if !ok || float64(time.Now().Unix()) > exp {
		return 0, errors.New("token expired or invalid")
	}

	idFloat, ok := claims["id"].(float64)
	if !ok {
		return 0, errors.New("invalid token payload")
	}

	return uint(idFloat), nil
}
