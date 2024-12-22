package db

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var Client *mongo.Client

// Підключення до MongoDB
func Connect(uri string) {
	var err error
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	Client, err = mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatalf("Помилка підключення до MongoDB: %v", err)
	}

	err = Client.Ping(ctx, nil)
	if err != nil {
		log.Fatalf("Помилка при перевірці підключення до MongoDB: %v", err)
	}

	log.Println("Успішне підключення до MongoDB")
}
