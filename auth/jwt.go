package auth

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var secret = []byte(os.Getenv("JWT_SECRET"))

func GenerateJWT(userID uint) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(time.Hour * 24 * 3).Unix(), // expires in 3 days
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

func VerifyJWT(tokenString string) (*jwt.Token, error) {
	// parse token
	jwtToken, err :=  jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return secret, nil
	})
	
	if err != nil {
		return nil, err
	}
	
	// isValid
	if !jwtToken.Valid {
		return nil, errors.New("token invalid")
	}

	return jwtToken, nil
} 