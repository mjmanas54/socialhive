package routes

import (
	"github.com/gin-gonic/gin"
	"socialhive/controllers"
	"socialhive/middlewares"
)

func PostRouter(incomingRoutes *gin.Engine) {
	incomingRoutes.Use(middlewares.RequireAuth)

	incomingRoutes.POST("/create_post", controllers.CreatePost)
	incomingRoutes.GET("/posts/:user_id", controllers.GetPostsByUserId)
	incomingRoutes.GET("images/:image_id", controllers.GetImage)
	incomingRoutes.GET("/update_likes/:action/:post_id/:user_id", controllers.UpdateLikes)
	incomingRoutes.POST("/add_comment", controllers.AddComment)
}
