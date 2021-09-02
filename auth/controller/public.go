package controller

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/tuxcanfly/loggy/auth/jwt"
	"github.com/tuxcanfly/loggy/auth/models"
	"gorm.io/gorm"
)

const authSecretKey = "c8b7b19b-19a0-4201-bc42-dfe6111d8819"
const authService = "AuthService"
const authExpirationInHours = 24

type UserServer struct {
	DB *gorm.DB
}

// LoginPayload login body
type LoginPayload struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse token response
type LoginResponse struct {
	Token  string `json:"token"`
	UserID string `json:"user_id"`
}

// Signup creates a user in db
func (u *UserServer) Signup(c *gin.Context) {
	var user models.User

	err := c.ShouldBindJSON(&user)
	if err != nil {
		log.Println(err)

		c.JSON(400, gin.H{
			"error": err.Error(),
		})
		c.Abort()

		return
	}

	err = user.HashPassword(user.Password)
	if err != nil {
		log.Println(err.Error())

		c.JSON(400, gin.H{
			"error": err.Error(),
		})
		c.Abort()

		return
	}

	err = user.CreateUserRecord(u.DB)
	if err != nil {
		log.Println(err)

		c.JSON(400, gin.H{
			"error": err.Error(),
		})
		c.Abort()

		return
	}

	c.JSON(200, user)
}

// Login logs users in
func (u *UserServer) Login(c *gin.Context) {
	var payload LoginPayload
	var user models.User

	err := c.ShouldBindJSON(&payload)
	if err != nil {
		c.JSON(400, gin.H{
			"error": err.Error(),
		})
		c.Abort()
		return
	}

	result := u.DB.Where("email = ?", payload.Email).First(&user)

	if result.Error == gorm.ErrRecordNotFound {
		c.JSON(401, gin.H{
			"msg": "invalid user credentials",
		})
		c.Abort()
		return
	}

	err = user.CheckPassword(payload.Password)
	if err != nil {
		log.Println(err)
		c.JSON(401, gin.H{
			"error": err.Error(),
		})
		c.Abort()
		return
	}

	jwtWrapper := jwt.Wrapper{
		SecretKey:       authSecretKey,
		Issuer:          authService,
		ExpirationHours: authExpirationInHours,
	}

	signedToken, err := jwtWrapper.GenerateToken(user.Email)
	if err != nil {
		log.Println(err)
		c.JSON(400, gin.H{
			"error": err.Error(),
		})
		c.Abort()
		return
	}

	tokenResponse := LoginResponse{
		Token:  signedToken,
		UserID: user.ID,
	}

	c.JSON(200, tokenResponse)
}

func (u *UserServer) Verify(c *gin.Context) {
	var payload LoginResponse
	err := c.ShouldBindJSON(&payload)
	if err != nil {
		log.Println(err)

		c.JSON(400, gin.H{
			"message": err.Error(),
		})
		c.Abort()

		return
	}

	jwtWrapper := jwt.Wrapper{
		SecretKey:       authSecretKey,
		Issuer:          authService,
		ExpirationHours: authExpirationInHours,
	}

	_, err = jwtWrapper.ValidateToken(payload.Token)
	if err != nil {
		log.Println(err)
		c.JSON(400, gin.H{
			"message": err.Error(),
		})
		c.Abort()
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "token valid",
	})
}

func (u *UserServer) VerifyAPIKey(c *gin.Context) {
	var user models.User

	apiKey := c.Query("api_key")

	result := u.DB.Where("api_key = ?", apiKey).First(&user)

	if result.Error == gorm.ErrRecordNotFound {
		c.JSON(401, gin.H{
			"msg": "invalid api key credentials",
		})
		c.Abort()
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id": user.ID,
	})
}
