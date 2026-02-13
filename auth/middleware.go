package auth

import (
	"collaborative-markdown-editor/internal/domain"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type UserProvider interface {
	GetUserByID(id uint64) (*domain.User, error)
}

type Middleware struct {
	UserService UserProvider
	InternalSecret string
}

func (m *Middleware) AuthMiddleWare() gin.HandlerFunc {
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
		
		userID, tokenVersion, err := GetDataFromToken(parsedToken)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		user, err := m.UserService.GetUserByID(userID)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		// Check token version
		if user.TokenVersion != tokenVersion {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization failed!"})
			return
		}

		userName := ctx.Query("userName")
		ctx.Set("user_name", userName)
		ctx.Set("user_id", userID)
		ctx.Set("jwt_token", token)
		ctx.Next()
	}
}

func (m *Middleware) InternalAuthMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		token := strings.TrimPrefix(
			ctx.GetHeader("Authorization"),
			"Bearer ",
		)

		if token != m.InternalSecret {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized internal call!"})
			return
		}

		ctx.Next()
	}
}