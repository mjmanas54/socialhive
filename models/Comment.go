package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

type Comment struct {
	ID          primitive.ObjectID `json:"_id" bson:"_id"`
	CommenterID primitive.ObjectID `json:"commenter_id" bson:"commenter_id"`
	CommentText string             `json:"comment_text" bson:"comment_text"`
	CommentedAt time.Time          `json:"commented_at" bson:"commented_at"`
}
