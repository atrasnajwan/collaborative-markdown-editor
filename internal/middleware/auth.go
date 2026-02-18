package middleware

import (
	"collaborative-markdown-editor/internal/auth"
	"collaborative-markdown-editor/internal/domain"
	"collaborative-markdown-editor/internal/errors"
	"strings"

	"github.com/gin-gonic/gin"
)

type UserProvider interface {
	GetUserByID(id uint64) (*domain.User, error)
}

type Auth struct {
	UserService UserProvider
	InternalSecret string
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

		user, err := m.UserService.GetUserByID(userID)
		if err != nil {
			ctx.Error(errors.Unauthorized("Invalid User ID!", err))
			ctx.Abort()
			return
		}

		// Check token version
		if user.TokenVersion != tokenVersion {
			ctx.Error(errors.Unauthorized("Invalid token version!", nil))
			ctx.Abort()
			return
		}

		userName := ctx.Query("userName")
		ctx.Set("user_name", userName)
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