package routes

import (
	"github.com/gin-gonic/gin"
	"socialhive/controllers"
	"socialhive/middlewares"
)

func HomeRoutes(incomingRoutes *gin.Engine) {
	incomingRoutes.Use(middlewares.RequireAuth)
	incomingRoutes.GET("/validate", controllers.ValidateUser)
}
