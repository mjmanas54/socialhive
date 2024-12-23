package routes

import (
	"github.com/gin-gonic/gin"
	"socialhive/controllers"
	"socialhive/middlewares"
)

func ConnectionRouter(incomingRoutes *gin.Engine) {
	incomingRoutes.Use(middlewares.RequireAuth)

	incomingRoutes.POST("/follow-request", controllers.SendFollowRequest)
	incomingRoutes.PUT("/follow-request/:request_id/accept", controllers.AcceptFollowRequest)
	incomingRoutes.DELETE("/follow-request/receiver/:request_id", controllers.DeleteFollowRequestByReceiver)
	incomingRoutes.DELETE("/follow-request/sender/:request_id", controllers.DeleteFollowRequestBySender)
	incomingRoutes.GET("/followers/:user_id", controllers.GetAllFollowers)
	incomingRoutes.GET("/following/:user_id", controllers.GetAllFollowing)
	incomingRoutes.DELETE("/unfollow/:user_id", controllers.UnFollow)
}
