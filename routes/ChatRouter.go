package routes

import (
	"github.com/gin-gonic/gin"
	"socialhive/controllers"
	"socialhive/middlewares"
)

func ChatRouter(incomingRoutes *gin.Engine) {
	incomingRoutes.Use(middlewares.RequireAuth)
	s := controllers.NewServer()
	incomingRoutes.GET("/ws", s.HandleWS)
}
