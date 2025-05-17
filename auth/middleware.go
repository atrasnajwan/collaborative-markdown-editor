package auth

import (
	"collaborative-markdown-editor/redis"
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
		_, err := VerifyJWT(token)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}
		
		// check on redis
		exists, err := redis.RedisClient.Exists(redis.Ctx, token).Result()
		if err != nil || exists == 0 {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token expired or not found"})
			return
		}

		ctx.Set("jwt_token", token)
		ctx.Next()
	}
}