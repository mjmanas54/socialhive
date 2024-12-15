package helper

import (
	"context"
	"errors"
	"go.mongodb.org/mongo-driver/bson"
	"socialhive/database"
	"socialhive/models"
	"time"
)

func GetUserByEmail(email string) (models.User, error) {
	var user models.User

	filter := bson.M{"email": email}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	userCollection := database.OpenCollection(database.Client, "user-collection")
	err := userCollection.FindOne(ctx, filter).Decode(&user)
	if err != nil {
		return models.User{}, errors.New("user not found")
	}
	return user, nil
}
