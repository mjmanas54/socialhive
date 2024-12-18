package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

type User struct {
	ID        primitive.ObjectID `json:"_id" bson:"_id"`
	Name      string             `json:"name" bson:"name" validate:"required"`
	Email     string             `json:"email" bson:"email" validate:"required"`
	Password  string             `json:"password" bson:"password" validate:"required"`
	CreatedAt time.Time          `json:"createdAt" bson:"createdAt"`
	Dp        string             `json:"dp" bson:"dp"`
	Posts     []Post             `json:"posts" bson:"posts"`
}
