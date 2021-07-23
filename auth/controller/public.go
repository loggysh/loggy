package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/tuxcanfly/loggy/auth/database"
	"github.com/tuxcanfly/loggy/auth/jwt"
	"github.com/tuxcanfly/loggy/auth/models"
	"gorm.io/gorm"
	"log"
)

// Signup creates a user in db
func Signup(c *gin.Context) {
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

	err = user.CreateUserRecord()
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

// LoginPayload login body
type LoginPayload struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse token response
type LoginResponse struct {
	Token string `json:"token"`
	UserID string `json:"user_id"`
}

// Login logs users in
func Login(c *gin.Context) {
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

	result := database.GlobalDB.Where("email = ?", payload.Email).First(&user)

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
		SecretKey:       "verysecretkey",
		Issuer:          "AuthService",
		ExpirationHours: 24,
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
		Token: signedToken,
		UserID: user.ID,
	}

	c.JSON(200, tokenResponse)

	return
}
func Verify(c *gin.Context) {
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
		SecretKey:       "verysecretkey",
		Issuer:          "AuthService",
		ExpirationHours: 24,
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
	c.JSON(200, gin.H{
		"message": "token valid",
	})
}