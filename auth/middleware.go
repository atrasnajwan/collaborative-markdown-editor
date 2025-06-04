package auth

import (
	"collaborative-markdown-editor/redis"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func AuthMiddleWare() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		authHeader := ctx.GetHeader("Authorization")
		if authHeader == "" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization is not found!"})
			return
		}

		// verify token
		token := strings.TrimPrefix(authHeader, "Bearer ")
		parsedToken, err := VerifyJWT(token)
		if err != nil {
			log.Println(err)
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		
		userID, err := GetUserIDFromToken(parsedToken)
		if err != nil {
			log.Println(err)
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		// check on redis
		exists, err := redis.RedisClient.Exists(redis.Ctx, token).Result()
		if err != nil || exists == 0 {
			log.Println(err)
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token expired or not found"})
			return
		}

		ctx.Set("user_id", userID)
		ctx.Set("jwt_token", token)
		ctx.Next()
	}
}