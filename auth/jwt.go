package auth

import (
	"collaborative-markdown-editor/internal/config"
	"collaborative-markdown-editor/internal/errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)


func GenerateAccessToken(userID uint64, tokenVersion int64) (string, error) {
	claims := jwt.MapClaims{
		"user_id": 			userID,
		"token_version": 	tokenVersion,
		"exp":     			time.Now().Add(time.Minute * 30).Unix(), // expires in 30 minutes
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(config.AppConfig.JWTSecret))
}

func GenerateRefreshToken(userID uint64, tokenVersion int64) (string, error) {
	claims := jwt.MapClaims{
		"user_id":       userID,
		"token_version": tokenVersion,
		"exp":           time.Now().Add(7 * 24 * time.Hour).Unix(), // 7 days
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

func GetDataFromToken(token *jwt.Token) (uint64, int64, error) {
    claims, ok := token.Claims.(jwt.MapClaims)
    if !ok {
        return 0, 0, errors.ErrUnauthorized(nil).WithMessage("invalid token claims")
    }
    
	userIDFloat, ok := claims["user_id"].(float64)
    if !ok {
        return 0, 0, errors.ErrUnauthorized(nil).WithMessage("user_id not found in token")
    }
	
	tokenVersionFloat, ok := claims["token_version"].(float64)
    if !ok {
        return 0, 0, errors.ErrUnauthorized(nil).WithMessage("token_version not found in token")
    }

    return uint64(userIDFloat), int64(tokenVersionFloat), nil
}