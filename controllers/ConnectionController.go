package controllers

import (
	"context"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"net/http"
	"socialhive/helper"
	"socialhive/models"
	"time"
)

func SendFollowRequest(c *gin.Context) {
	senderIDHex := c.PostForm("sender")
	receiverIDHex := c.PostForm("receiver")

	senderID, err := primitive.ObjectIDFromHex(senderIDHex)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	receiverID, err := primitive.ObjectIDFromHex(receiverIDHex)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// check if sender is logged-in user
	loggedInUserEmail := helper.ExtractEmail(c)

	user, err := helper.GetUserByEmail(loggedInUserEmail)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if user.ID != senderID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sender is not the same as the logged-in user"})
		return
	}

	// check if already followed the receiver

	filter := bson.M{
		"_id":       user.ID,
		"following": bson.M{"$in": []primitive.ObjectID{receiverID}},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var result models.User
	err = userCollection.FindOne(ctx, filter).Decode(&result)
	if err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sender has already followed receiver"})
		return
	}

	if err != mongo.ErrNoDocuments {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var followRequest models.FollowRequest

	followRequest = models.FollowRequest{
		ID:        primitive.NewObjectID(),
		From:      senderID,
		To:        receiverID,
		CreatedAt: time.Now(),
	}

	filter = bson.M{"_id": receiverID}
	update := bson.M{"$push": bson.M{"requestsReceived": followRequest}}

	_, err = userCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	filter = bson.M{"_id": senderID}
	update = bson.M{"$push": bson.M{"requestsSent": followRequest}}

	_, err = userCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"requestId": followRequest.ID})

}

func AcceptFollowRequest(c *gin.Context) {
	requestIDHex := c.Param("request_id")
	requestID, err := primitive.ObjectIDFromHex(requestIDHex)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	loggedInUserEmail := helper.ExtractEmail(c)
	user, err := helper.GetUserByEmail(loggedInUserEmail)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	filter := bson.M{"_id": user.ID}
	projection := bson.M{
		"requestsReceived": bson.M{
			"$elemMatch": bson.M{
				"_id": requestID,
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	type Result struct {
		FollowRequests []models.FollowRequest `bson:"requestsReceived"`
	}

	var result Result

	err = userCollection.FindOne(ctx, filter, options.FindOne().SetProjection(projection)).Decode(&result)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		} else {
			c.JSON(500, gin.H{"error": err.Error()})
		}
		return
	}

	// Check and print the found FollowRequest
	if len(result.FollowRequests) <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No such request found"})
		return
	}

	var followRequest models.FollowRequest = result.FollowRequests[0]

	filter = bson.M{"_id": user.ID}
	update := bson.M{
		"$pull": bson.M{
			"requestsReceived": bson.M{"_id": requestID},
		},
		"$push": bson.M{
			"followers": followRequest.From,
		},
	}

	_, err = userCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	filter = bson.M{"_id": followRequest.From}
	update = bson.M{
		"$pull": bson.M{
			"requestsSent": bson.M{"_id": requestID},
		},
		"$push": bson.M{
			"following": followRequest.To,
		},
	}

	_, err = userCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "request accepted successfully"})
}

func DeleteFollowRequestBySender(c *gin.Context) {
	requestIDHex := c.Param("request_id")
	requestID, err := primitive.ObjectIDFromHex(requestIDHex)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	loggedInUserEmail := helper.ExtractEmail(c)
	user, err := helper.GetUserByEmail(loggedInUserEmail)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	filter := bson.M{"_id": user.ID}
	projection := bson.M{
		"requestsSent": bson.M{
			"$elemMatch": bson.M{
				"_id": requestID,
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	type Result struct {
		FollowRequests []models.FollowRequest `bson:"requestsSent"`
	}

	var result Result

	err = userCollection.FindOne(ctx, filter, options.FindOne().SetProjection(projection)).Decode(&result)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		} else {
			c.JSON(500, gin.H{"error": err.Error()})
		}
		return
	}

	// Check and print the found FollowRequest
	if len(result.FollowRequests) <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No such request found"})
		return
	}

	var followRequest models.FollowRequest = result.FollowRequests[0]

	filter = bson.M{"_id": user.ID}
	update := bson.M{
		"$pull": bson.M{
			"requestsSent": bson.M{"_id": requestID},
		},
	}

	_, err = userCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	filter = bson.M{"_id": followRequest.To}
	update = bson.M{
		"$pull": bson.M{
			"requestsReceived": bson.M{"_id": requestID},
		},
	}

	_, err = userCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "request deleted successfully"})
}

func DeleteFollowRequestByReceiver(c *gin.Context) {
	requestIDHex := c.Param("request_id")
	requestID, err := primitive.ObjectIDFromHex(requestIDHex)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	loggedInUserEmail := helper.ExtractEmail(c)
	user, err := helper.GetUserByEmail(loggedInUserEmail)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	filter := bson.M{"_id": user.ID}
	projection := bson.M{
		"requestsReceived": bson.M{
			"$elemMatch": bson.M{
				"_id": requestID,
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	type Result struct {
		FollowRequests []models.FollowRequest `bson:"requestsReceived"`
	}

	var result Result

	err = userCollection.FindOne(ctx, filter, options.FindOne().SetProjection(projection)).Decode(&result)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		} else {
			c.JSON(500, gin.H{"error": err.Error()})
		}
		return
	}

	// Check and print the found FollowRequest
	if len(result.FollowRequests) <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No such request found"})
		return
	}

	var followRequest models.FollowRequest = result.FollowRequests[0]

	filter = bson.M{"_id": user.ID}
	update := bson.M{
		"$pull": bson.M{
			"requestsReceived": bson.M{"_id": requestID},
		},
	}

	_, err = userCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	filter = bson.M{"_id": followRequest.From}
	update = bson.M{
		"$pull": bson.M{
			"requestsSent": bson.M{"_id": requestID},
		},
	}

	_, err = userCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "request deleted successfully"})
}

func GetAllFollowers(c *gin.Context) {
	userIDHex := c.Param("user_id")
	userID, err := primitive.ObjectIDFromHex(userIDHex)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := helper.GetUserById(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	filter := bson.M{
		"_id": bson.M{
			"$in": user.Followers,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := userCollection.Find(ctx, filter)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	defer cursor.Close(ctx)
	var users []models.User
	err = cursor.All(ctx, &users)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, users)
}

func GetAllFollowing(c *gin.Context) {
	userIDHex := c.Param("user_id")
	userID, err := primitive.ObjectIDFromHex(userIDHex)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := helper.GetUserById(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	filter := bson.M{
		"_id": bson.M{
			"$in": user.Following,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := userCollection.Find(ctx, filter)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	defer cursor.Close(ctx)
	var users []models.User
	err = cursor.All(ctx, &users)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, users)
}

func UnFollow(c *gin.Context) {
	userIDHex := c.Param("user_id")
	userID, err := primitive.ObjectIDFromHex(userIDHex)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	loggedInUserEmail := helper.ExtractEmail(c)
	loggedInUser, err := helper.GetUserByEmail(loggedInUserEmail)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	filter := bson.M{"_id": userID}
	update := bson.M{
		"$pull": bson.M{
			"followers": loggedInUser.ID,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	result, err := userCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if result.MatchedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	if result.ModifiedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	filter = bson.M{"_id": loggedInUser.ID}
	update = bson.M{
		"$pull": bson.M{
			"following": userID,
		},
	}

	result, err = userCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if result.MatchedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	if result.ModifiedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user un-followed successfully"})
}
