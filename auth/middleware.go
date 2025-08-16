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
		var token string
		tokenQuery := ctx.Query("token")

		if authHeader != "" {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		} else if tokenQuery != "" {
			token = tokenQuery
		} else {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization is not found!"})
			return		
		}
		// log.Println("token", token)
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
		userName := ctx.Query("userName")
		ctx.Set("user_name", userName)
		ctx.Set("user_id", userID)
		ctx.Set("jwt_token", token)
		ctx.Next()
	}
}

func AuthWebSocketMiddleWare() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// verify token
		token := ctx.Query("token")
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
		ctx.Set("username", ctx.Query("userName"))
		ctx.Next()
	}
}