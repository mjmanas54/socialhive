package main

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"log"
	"os"
	"socialhive/intializers"
	"socialhive/routes"
)

func init() {
	intializers.LoadEnvVariables()
}

func main() {
	router := gin.Default()

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://127.0.0.1:5500", "http://localhost:3000"}, // Add allowed origins
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Authorization"},
		AllowCredentials: true, // Allow cookies if needed
	}))

	routes.AuthRouter(router)

	// middleware using routes
	routes.ChatRouter(router)
	routes.HomeRoutes(router)
	routes.MessageRouter(router)
	routes.PostRouter(router)

	PORT := os.Getenv("PORT")

	if err := router.Run(":" + PORT); err != nil {
		log.Fatal(err)
	}
}
