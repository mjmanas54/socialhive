package routes

import (
	"github.com/gin-gonic/gin"
	"socialhive/controllers"
)

func AuthRouter(incomingRoutes *gin.Engine) {
	incomingRoutes.POST("/signup", controllers.SignUp)
	incomingRoutes.POST("/login", controllers.Login)
	incomingRoutes.POST("/logout", controllers.Logout)
}
