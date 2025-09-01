package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return string(hashedPassword), nil
}

func CheckPasswordHash(password, hash string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

func GetBearerToken(headers http.Header) (string, error) {
	authHeader := headers.Get("Authorization")
	splitToken := strings.Split(authHeader, "Bearer ")
	if len(splitToken) != 2 {
		return "", errors.New("invalid authorization header format")
	}

	reqToken := strings.TrimSpace(splitToken[1])
	return reqToken, nil
}

func MakeRefreshToken() (string, error) {
	randBytes := make([]byte, 256)
	if _, err := rand.Read(randBytes); err != nil {
		return "", err
	}

	return hex.EncodeToString(randBytes), nil
}

func GetAPIKey(headers http.Header) (string, error) {
	authHeader := headers.Get("Authorization")
	splitToken := strings.Split(authHeader, "ApiKey ")
	if len(splitToken) != 2 {
		return "", errors.New("invalid api authorization header format")
	}

	return strings.TrimSpace(splitToken[1]), nil
}
