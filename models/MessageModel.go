package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

type Message struct {
	ID        primitive.ObjectID `json:"_id" bson:"_id"`
	Sender    User               `json:"sender" bson:"sender"`
	Recipient User               `json:"recipient" bson:"recipient"`
	Content   string             `json:"content" bson:"content"`
	Timestamp time.Time          `json:"timestamp" bson:"timestamp"`
	Status    string             `json:"status" bson:"status"`
}
