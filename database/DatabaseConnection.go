package database

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/gridfs"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"os"
	"socialhive/intializers"
	"time"
)

func DBinstance() *mongo.Client {
	// Load environment variables
	intializers.LoadEnvVariables()

	MongoUri := os.Getenv("MONGO_URI")
	client, err := mongo.NewClient(options.Client().ApplyURI(MongoUri))
	if err != nil {
		panic(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = client.Connect(ctx)
	if err != nil {
		panic(err)
	}
	fmt.Println("Connected to MongoDB!")
	return client
}

var Client *mongo.Client = DBinstance()

func OpenCollection(client *mongo.Client, collectionName string) *mongo.Collection {
	collection := client.Database(os.Getenv("DB_NAME")).Collection(collectionName)
	return collection
}

func BucketInstance() *gridfs.Bucket {
	gridFsBucket, err := gridfs.NewBucket(Client.Database(os.Getenv("DB_NAME")))
	if err != nil {
		log.Fatal(err)
	}
	return gridFsBucket
}

var GridFSBucket *gridfs.Bucket = BucketInstance()
