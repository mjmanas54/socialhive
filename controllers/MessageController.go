package controllers

import (
	"context"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"net/http"
	"socialhive/database"
	"socialhive/helper"
	"socialhive/models"
	"time"
)

var msgCollection *mongo.Collection = database.OpenCollection(database.Client, "message-collection")

func GetMessagesByUsers(c *gin.Context) {
	user1Email := c.Param("user1")
	user2Email := c.Param("user2")

	// check whether client is requesting its own data
	if !checkUserEmails(c, user1Email, user2Email) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "you are not allowed to request this data"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Check if both users exist
	filter := bson.M{"email": bson.M{"$in": []string{user1Email, user2Email}}}
	count, err := userCollection.CountDocuments(ctx, filter)
	if err != nil || count < 2 {
		c.JSON(http.StatusNotFound, gin.H{"error": "One or both users not found"})
		return
	}

	filter = bson.M{
		"$or": []bson.M{
			{"sender.email": user1Email, "recipient.email": user2Email},
			{"sender.email": user2Email, "recipient.email": user1Email},
		},
	}

	cursor, err := msgCollection.Find(ctx, filter)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching messages"})
		return
	}
	defer cursor.Close(ctx)

	var messages []models.Message
	if err := cursor.All(ctx, &messages); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error decoding messages"})
		return
	}

	c.JSON(http.StatusOK, messages)
}

func checkUserEmails(c *gin.Context, user1Email string, user2Email string) bool {
	loggedInUserEmail := helper.ExtractEmail(c)

	if loggedInUserEmail == user1Email || loggedInUserEmail == user2Email {
		return true
	} else {
		return false
	}
}

func DeleteMessage(c *gin.Context) {
	messageId := c.Param("message_id")
	var message models.Message

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	objectId, err := primitive.ObjectIDFromHex(messageId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error decoding message_id"})
		return
	}

	cursor, err := msgCollection.Find(ctx, bson.M{"_id": objectId})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error searching message_id"})
		return
	}

	if !cursor.Next(ctx) {
		c.JSON(http.StatusNotFound, gin.H{"error": "There is no such message in database."})
		return
	}

	if err := cursor.Decode(&message); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error decoding message_id"})
		return
	}

	// check the message is sent by the logged-in user
	loggedInUserEmail := helper.ExtractEmail(c)
	senderEmail := message.Sender.Email

	if loggedInUserEmail != senderEmail {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not allowed to delete this message"})
		return
	}

	msgCollection.FindOneAndDelete(ctx, bson.M{"_id": objectId})
	c.JSON(http.StatusOK, message)

}
