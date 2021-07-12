package main

import (
	"github.com/tuxcanfly/loggy/auth/controller"
	"github.com/tuxcanfly/loggy/auth/database"
	"github.com/tuxcanfly/loggy/auth/models"
	"log"

	"github.com/gin-gonic/gin"
)

func setupRouter() *gin.Engine {
	r := gin.Default()

	r.GET("/ping", func(c *gin.Context) {
		c.String(200, "pong")
	})

	//here
	api := r.Group("/api")
	{
		public := api.Group("/public")
		{
			public.POST("/login", controller.Login)
			public.POST("/signup", controller.Signup)
			public.POST("/verify", controller.Verify)
		}
	}

	return r
}

func main() {
	err := database.InitDatabase()
	if err != nil {
		log.Fatalln("could not create database", err)
	}

	err = database.GlobalDB.AutoMigrate(&models.User{})
	if err != nil {
		log.Fatalf("%v\n", err)
	}

	r := setupRouter()
	err = r.Run(":8080")
	if err != nil {
		log.Fatalf("%v\n", err)
	}
}