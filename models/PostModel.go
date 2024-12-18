package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

type Post struct {
	ID        primitive.ObjectID   `json:"_id" bson:"_id"`
	Uploader  primitive.ObjectID   `json:"uploader" bson:"uploader"`
	Text      string               `json:"text" bson:"text"`
	CreatedAt time.Time            `json:"createdAt" bson:"createdAt"`
	LikedBy   []primitive.ObjectID `json:"likedBy" bson:"likedBy"`
	Images    []string             `json:"images" bson:"images"`
	Comments  []Comment            `json:"comments" bson:"comments"`
}
