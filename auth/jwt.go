package auth

import (
	"collaborative-markdown-editor/internal/config"
	"collaborative-markdown-editor/internal/errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)


func GenerateJWT(userID uint64) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(time.Hour * 24 * 3).Unix(), // expires in 3 days
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(config.AppConfig.JWTSecret))
}

func VerifyJWT(tokenString string) (*jwt.Token, error) {
	// parse token
	jwtToken, err :=  jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(config.AppConfig.JWTSecret), nil
	})
	
	if err != nil {
		return nil, err
	}
	
	// isValid
	if !jwtToken.Valid {
		return nil, errors.ErrUnauthorized(nil).WithMessage("token invalid")
	}

	return jwtToken, nil
} 

func GetUserIDFromToken(token *jwt.Token) (uint64, error) {
    claims, ok := token.Claims.(jwt.MapClaims)
    if !ok {
        return 0, errors.ErrUnauthorized(nil).WithMessage("invalid token claims")
    }
    userIDFloat, ok := claims["user_id"].(float64)
    if !ok {
        return 0, errors.ErrUnauthorized(nil).WithMessage("user_id not found in token")
    }
    return uint64(userIDFloat), nil
}