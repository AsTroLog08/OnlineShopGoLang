package analytics

import (
	"context"
	"math/rand"
	"time"

	"online-store/server/db"
	"online-store/server/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func GenerateRandomOrders() error {
	collection := db.Client.Database("store").Collection("orders")
	productsCollection := db.Client.Database("store").Collection("products")

	// Отримуємо всі товари
	var products []models.Product
	cursor, err := productsCollection.Find(context.TODO(), bson.M{})
	if err != nil {
		return err
	}
	defer cursor.Close(context.TODO())

	for cursor.Next(context.TODO()) {
		var product models.Product
		cursor.Decode(&product)
		products = append(products, product)
	}

	if len(products) == 0 {
		return nil // Якщо немає товарів, генерація неможлива
	}

	// Генерація випадкових замовлень
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < 100; i++ {
		var orderProducts []models.OrderProduct
		usedProducts := make(map[string]bool) // Щоб уникнути дублювання товарів у замовленні

		// Генеруємо випадкову кількість товарів у замовленні (1-5)
		numProducts := rand.Intn(5) + 1
		for j := 0; j < numProducts; j++ {
			product := products[rand.Intn(len(products))]
			if usedProducts[product.ID] {
				continue // Уникаємо дублювання
			}
			usedProducts[product.ID] = true

			orderProducts = append(orderProducts, models.OrderProduct{
				ProductID: product.ID,
				Quantity:  rand.Intn(5) + 1, // Кількість товару (1-5)
				Price:     product.Price,
			})
		}

		// Розрахунок загальної суми
		total := 0.0
		for _, op := range orderProducts {
			total += float64(op.Quantity) * op.Price
		}

		order := models.Order{
			ID:       primitive.NewObjectID(),
			UserID:   primitive.NewObjectID().Hex(),
			Date:     time.Now().AddDate(0, 0, -rand.Intn(30)).Format("2006-01-02"),
			Products: orderProducts,
			Total:    total,
		}

		_, err := collection.InsertOne(context.TODO(), order)
		if err != nil {
			return err
		}
	}

	return nil
}
