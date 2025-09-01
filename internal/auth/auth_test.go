package auth

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCheckPasswordHash(t *testing.T) {
	cases := []string{
		"1234567890",
		"moredifficultpassword",
		"another1092password",
	}

	for i, c := range cases {
		t.Run(fmt.Sprintf("Test case %v", i), func(t *testing.T) {
			hashedPassword, err := HashPassword(c)
			if err != nil {
				t.Errorf("cannot hash password 1")
				return
			}
			result := CheckPasswordHash(c, hashedPassword)
			if result != nil {
				t.Errorf("invalid check")
				return
			}
		})
	}
}

func TestValidateJWT(t *testing.T) {
	jwtSecret := os.Getenv("JWT_SECRET")
	expiresIn, _ := time.ParseDuration("1s")

	userID1 := uuid.New()
	userID2 := uuid.New()
	userID3 := uuid.New()

	cases := []uuid.UUID{
		userID1,
		userID2,
		userID3,
	}

	for i, c := range cases {
		t.Run(fmt.Sprintf("Test case %v", i), func(t *testing.T) {
			jwtString, err := MakeJWT(c, jwtSecret, expiresIn)
			if err != nil {
				t.Errorf("cannot make jwt: %v", err)
				return
			}

			userID, err := ValidateJWT(jwtString, jwtSecret)
			if err != nil {
				t.Errorf("cannot validate jwt: %v", err)
				return
			}

			if userID != c {
				t.Errorf("User ID must be %v but got %v", c, userID)
				t.Fail()
			}
		})
	}
}

func TestValidateJWTExpired(t *testing.T) {
	jwtSecret := os.Getenv("JWT_SECRET")
	expiresIn, _ := time.ParseDuration("1ms")

	userID1 := uuid.New()
	userID2 := uuid.New()
	userID3 := uuid.New()

	cases := []uuid.UUID{
		userID1,
		userID2,
		userID3,
	}

	for i, c := range cases {
		t.Run(fmt.Sprintf("Test case %v", i), func(t *testing.T) {
			jwtString, err := MakeJWT(c, jwtSecret, expiresIn)
			if err != nil {
				t.Errorf("cannot make jwt: %v", err)
				return
			}

			time.Sleep(time.Millisecond * 2)
			_, err = ValidateJWT(jwtString, jwtSecret)
			if !strings.Contains(fmt.Sprintf("%v", err), "token is expired") {
				t.Errorf("jwt token expiration failed: %v", err)
				return
			}
		})
	}
}
