package middleware

import (
	"collaborative-markdown-editor/internal/auth"
	"collaborative-markdown-editor/internal/domain"
	"collaborative-markdown-editor/internal/errors"
	"collaborative-markdown-editor/redis"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type UserProvider interface {
	GetUserByID(ctx context.Context, id uint64) (*domain.User, error)
}

type Auth struct {
	UserService UserProvider
	InternalSecret string
	Cache *redis.Cache
}

func (m *Auth) AuthMiddleWare() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		authHeader := ctx.GetHeader("Authorization")
		var token string
		tokenQuery := ctx.Query("token")

		if authHeader != "" {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		} else if tokenQuery != "" {
			token = tokenQuery
		} else {
			ctx.Error(errors.Unauthorized("Authorization is not found!", nil))
			ctx.Abort()
			return		
		}

		parsedToken, err := auth.VerifyJWT(token)
		if err != nil {
			ctx.Error(errors.Unauthorized("Invalid token!", err))
			ctx.Abort()
			return
		}
		
		userID, tokenVersion, err := auth.GetDataFromToken(parsedToken)
		if err != nil {
			ctx.Error(errors.Unauthorized("Invalid token!", err))
			ctx.Abort()
			return
		}

		// try get token version from cache/redis
		cacheKey := fmt.Sprintf("user:version:%d", userID)
		var tokenVersionDB int64
		found, err := m.Cache.Get(ctx.Request.Context(), cacheKey, &tokenVersionDB)

		if err != nil || !found {
			user, err := m.UserService.GetUserByID(ctx.Request.Context(), userID)
			if err != nil {
				ctx.Error(errors.Unauthorized("Invalid User ID!", err))
				ctx.Abort()
				return
			}
			tokenVersionDB = user.TokenVersion
			// set to cache/redis
			m.Cache.Set(ctx.Request.Context(), cacheKey, tokenVersionDB, 15*time.Minute)
		}

		// Check token version
		if tokenVersionDB != tokenVersion {
			ctx.Error(errors.Unauthorized("Session expired. Please log in again.", nil))
			ctx.Abort()
			return
		}

		ctx.Set("user_id", userID)
		ctx.Set("jwt_token", token)
		ctx.Next()
	}
}

func (m *Auth) InternalAuthMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		token := strings.TrimPrefix(
			ctx.GetHeader("Authorization"),
			"Bearer ",
		)

		if token != m.InternalSecret {
			ctx.Error(errors.Unauthorized("Unauthorized internal call!", nil))
			ctx.Abort()
			return
		}

		ctx.Next()
	}
}