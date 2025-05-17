package main

import (
	"collaborative-markdown-editor/auth"
	"collaborative-markdown-editor/internal/db"
	"collaborative-markdown-editor/internal/user"
	"collaborative-markdown-editor/redis"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

type FormLogin struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func init() {
	if err := godotenv.Load("../../.env"); err != nil {
		log.Fatal("Error loading .env file\n", err)
	}
}

func main() {
	database, _ := db.ConnectDb()
	defer db.CloseDb(database)
	db.Migrate(database)

	redis.InitRedis()
	router := gin.Default()

	router.POST("/login", func(ctx *gin.Context) {
		var body FormLogin
		if err := ctx.ShouldBind(&body); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		currentUser := &user.User{Email: body.Email, Password: body.Password}

		if err := currentUser.FindByEmail(database); err != nil {
			log.Println(err)
			ctx.JSON(http.StatusUnprocessableEntity, gin.H{"error": "User not found!"})
			return
		}

		// check password
		err := bcrypt.CompareHashAndPassword([]byte(currentUser.PasswordHash), []byte(body.Password))
		if err != nil {
			// invalid password
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Wrong Password"})
			return
		}

		// generate JWT token
		token, err := auth.GenerateJWT(currentUser.ID)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
			return
		}

		// Store in Redis with expiration
		redis.RedisClient.Set(redis.Ctx, token, "1", time.Hour * 24 * 3)

		ctx.JSON(http.StatusOK, gin.H{"token": token})
	})

	router.DELETE("/logout", auth.AuthMiddleWare(), func(ctx *gin.Context) {
		jwtToken, exists := ctx.Get("jwt_token")
		if exists {
			log.Println("delete jwt token")
			redis.RedisClient.Del(redis.Ctx, jwtToken.(string))
		}

		log.Println("logout")
	})

	router.GET("/profile", auth.AuthMiddleWare(), func(ctx *gin.Context) {
		log.Println(ctx.Get("current_user"))
		// currentUser, _ := ctx.Get("current_user")

		// user := currentUser.(*user.User)
		// ctx.JSON(http.StatusOK, gin.H{
		// 	"id": "test",
		// 	// "name":  user.Name,
		// })
	})

	router.Run(":8080")
}
