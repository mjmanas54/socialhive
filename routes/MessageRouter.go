package routes

import (
	"github.com/gin-gonic/gin"
	"socialhive/controllers"
	"socialhive/middlewares"
)

func MessageRouter(incomingRoutes *gin.Engine) {
	incomingRoutes.Use(middlewares.RequireAuth)

	incomingRoutes.GET("/users", controllers.GetAllUsers)
	incomingRoutes.GET("/messages/:user1/:user2", controllers.GetMessagesByUsers)
	incomingRoutes.DELETE("/delete_message/:message_id", controllers.DeleteMessage)
}
