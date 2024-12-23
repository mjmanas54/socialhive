package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

type FollowRequest struct {
	ID        primitive.ObjectID `json:"_id" bson:"_id"`
	From      primitive.ObjectID `json:"from" bson:"from"`
	To        primitive.ObjectID `json:"to" bson:"to"`
	CreatedAt time.Time          `json:"created_at" bson:"created_at"`
}
