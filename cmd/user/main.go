package main

import (
	"github.com/tuxcanfly/loggy/auth/controller"
	"github.com/tuxcanfly/loggy/auth/models"
	"log"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/driver/sqlite"
)

func main() {
	//create database
	db, err := gorm.Open(sqlite.Open("db/test.db"), &gorm.Config{})
	if err != nil {
		log.Fatalln("could not create database", err)
	}

	err = db.AutoMigrate(&models.User{})
	if err != nil {
		log.Fatalf("user database migration failed", err)
	}

	userServer := controller.UserServer {
		DB: db,
	}

	router := gin.Default()

	router.GET("/ping", func(c *gin.Context) {
		c.String(200, "pong")
	})

	api := router.Group("/api")
	public := api.Group("/public")
	public.POST("/login", userServer.Login)
	public.POST("/signup", userServer.Signup)
	public.POST("/verify", userServer.Verify)

	err = router.Run(":8080")
	if err != nil {
		log.Fatalf("Gin engine run failed", err)
	}

}
