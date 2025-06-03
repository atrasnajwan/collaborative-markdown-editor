package auth

import (
	"collaborative-markdown-editor/internal/config"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)


func GenerateJWT(userID uint) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(time.Hour * 24 * 3).Unix(), // expires in 3 days
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(config.AppConfig.JWTSecret)
}

func VerifyJWT(tokenString string) (*jwt.Token, error) {
	// parse token
	jwtToken, err :=  jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return config.AppConfig.JWTSecret, nil
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