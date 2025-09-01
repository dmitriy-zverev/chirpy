package auth

import (
	"fmt"
	"testing"
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
			fmt.Println(hashedPassword)
			if err != nil {
				t.Errorf("cannot hash password 1")
				return
			}
			result := CheckPasswordHash(c, hashedPassword)
			fmt.Println(result)
			if result != nil {
				t.Errorf("invalid check")
				return
			}
		})
	}
}
