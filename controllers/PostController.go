package controllers

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/gridfs"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"socialhive/database"
	"socialhive/models"
	"time"
)

var postCollection *mongo.Collection = database.OpenCollection(database.Client, "posts-collection")

func CreatePost(c *gin.Context) {
	// uploader id
	hexId := c.PostForm("uploader")
	uploader, err := primitive.ObjectIDFromHex(hexId)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	text := c.PostForm("text")
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	files := form.File["files"]

	var imageIds []string

	bucket := database.GridFSBucket

	for _, file := range files {
		fileId, err := uploadToGridFS(bucket, file)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		imageIds = append(imageIds, fileId)
	}

	post := models.Post{
		ID:        primitive.NewObjectID(),
		Text:      text,
		Uploader:  uploader,
		CreatedAt: time.Now(),
		Images:    imageIds,
		LikedBy:   []primitive.ObjectID{},
		Comments:  []models.Comment{},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err = postCollection.InsertOne(ctx, post)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	filter := bson.M{"_id": uploader}
	update := bson.M{"$push": bson.M{"posts": post}}

	userCollection := database.OpenCollection(database.Client, "user-collection")

	_, err = userCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"post": post})
}

func uploadToGridFS(bucket *gridfs.Bucket, file *multipart.FileHeader) (string, error) {
	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	fileID := primitive.NewObjectID()
	uploadStream, err := bucket.OpenUploadStreamWithID(fileID, file.Filename)
	if err != nil {
		return "", err
	}
	defer uploadStream.Close()

	if _, err := io.Copy(uploadStream, src); err != nil {
		return "", err
	}

	return fileID.Hex(), nil
}

func GetPostsByUserId(c *gin.Context) {
	userIdHex := c.Param("user_id")
	userId, err := primitive.ObjectIDFromHex(userIdHex)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
		return
	}

	//userCollection := database.OpenCollection(database.Client, "user-collection")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var posts []models.Post

	cursor, err := postCollection.Find(ctx, bson.M{"uploader": userId})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	defer cursor.Close(ctx)
	for cursor.Next(ctx) {
		var post models.Post
		cursor.Decode(&post)
		posts = append(posts, post)
	}

	for i := range posts {
		for j := range posts[i].Images {
			posts[i].Images[j] = os.Getenv("HOST_NAME") + "images/" + posts[i].Images[j] // Construct the URL
		}
	}

	c.JSON(http.StatusOK, gin.H{"posts": posts})
}

func GetImage(c *gin.Context) {
	imageId := c.Param("image_id")

	// Convert string to ObjectID
	fileID, err := primitive.ObjectIDFromHex(imageId)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid image ID"})
		return
	}

	// Retrieve the image from GridFS
	file, err := database.GridFSBucket.OpenDownloadStream(fileID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Image not found"})
		return
	}
	defer file.Close()

	// Set the appropriate content type (you may want to make this dynamic)
	c.Header("Content-Type", "image/png")

	// Stream the file content to the response
	_, err = io.Copy(c.Writer, file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to stream image"})
		return
	}
}

func UpdateLikes(c *gin.Context) {
	postID := c.Param("post_id") // Get post ID from URL params
	userID := c.Param("user_id") // Get user ID from URL params
	action := c.Param("action")  // Get action (increment or decrement) from query params

	// Convert IDs to ObjectID
	postObjectID, err := primitive.ObjectIDFromHex(postID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid post ID"})
		return
	}

	userObjectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	postCollection := database.OpenCollection(database.Client, "posts-collection")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var update bson.M

	// Determine the update action based on the query parameter
	if action == "increment" {
		update = bson.M{"$addToSet": bson.M{"likedBy": userObjectID}} // Avoids duplicates
	} else if action == "decrement" {
		update = bson.M{"$pull": bson.M{"likedBy": userObjectID}}
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid action. Use 'increment' or 'decrement'"})
		return
	}

	// Log the update query for debugging
	fmt.Println("Update Query:", update)

	result, err := postCollection.UpdateOne(ctx, bson.M{"_id": postObjectID}, update)
	if err != nil {
		// Detailed error logging
		fmt.Println("MongoDB Update Error:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update likes: " + err.Error()})
		return
	}

	// Log the result of the update for debugging
	fmt.Println("Update Result:", result)

	if result.ModifiedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Post not found or no changes made"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Like status updated successfully"})
}

func AddComment(c *gin.Context) {
	postIDHex := c.PostForm("post_id")
	userIDHex := c.PostForm("user_id")
	text := c.PostForm("text")

	userID, err := primitive.ObjectIDFromHex(userIDHex)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
		return
	}

	postID, err := primitive.ObjectIDFromHex(postIDHex)

	comment := models.Comment{
		ID:          primitive.NewObjectID(),
		CommenterID: userID,
		CommentText: text,
		CommentedAt: time.Now(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	update := bson.M{"$push": bson.M{"comments": comment}}

	_, err = postCollection.UpdateOne(ctx, bson.M{"_id": postID}, update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add comment: " + err.Error()})
		fmt.Println(err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": comment})
}
