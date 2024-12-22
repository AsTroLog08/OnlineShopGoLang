package handlers

import (
	"context"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"online-store/server/db"
	"online-store/server/models"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var client *mongo.Client

// Ініціалізація клієнта MongoDB
func InitDB(c *mongo.Client) {
	client = c
}

func GetProductsHandler(w http.ResponseWriter, r *http.Request) {
    collection := db.Client.Database("store").Collection("products")
    cursor, err := collection.Find(context.TODO(), bson.D{})
    if err != nil {
        http.Error(w, "Не вдалося отримати товари", http.StatusInternalServerError)
        return
    }
    defer cursor.Close(context.TODO())

    var products []models.Product
    for cursor.Next(context.TODO()) {
        var product models.Product
        cursor.Decode(&product)
        products = append(products, product)
    }

    if len(products) == 0 {
        // Логування, якщо товари не знайдено
        log.Println("Товари не знайдено у базі даних")
    } else {
        log.Printf("Знайдено %d товарів\n", len(products))
    }

    tmpl, err := template.ParseFiles("web/templates/catalog.html")
    if err != nil {
        http.Error(w, "Не вдалося завантажити шаблон", http.StatusInternalServerError)
        return
    }

    tmpl.Execute(w, products)
}
func GetProductHandler(w http.ResponseWriter, r *http.Request) {
    id := mux.Vars(r)["id"]

    // Конвертація рядкового ID у MongoDB ObjectID
    objectID, err := primitive.ObjectIDFromHex(id)
    if err != nil {
        http.Error(w, "Невірний формат ID", http.StatusBadRequest)
        return
    }

    collection := db.Client.Database("store").Collection("products")
    var product models.Product

    // Пошук документа за ObjectID
    err = collection.FindOne(context.TODO(), bson.M{"_id": objectID}).Decode(&product)
    if err != nil {
        http.Error(w, "Товар не знайдено", http.StatusNotFound)
        return
    }

    tmpl, err := template.ParseFiles("web/templates/product.html")
    if err != nil {
        http.Error(w, "Не вдалося завантажити шаблон", http.StatusInternalServerError)
        return
    }

    tmpl.Execute(w, product)
}



func AddProductHandler(w http.ResponseWriter, r *http.Request) {
	if db.Client == nil {
		http.Error(w, "База даних не підключена", http.StatusInternalServerError)
		return
	}

	var product models.Product
	err := json.NewDecoder(r.Body).Decode(&product)
	if err != nil {
		http.Error(w, "Невірний формат даних", http.StatusBadRequest)
		return
	}

	collection := db.Client.Database("store").Collection("products")
	_, err = collection.InsertOne(context.TODO(), product)
	if err != nil {
		http.Error(w, "Не вдалося додати товар", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("Товар успішно додано"))
}

// Видалення товару
func DeleteProductHandler(w http.ResponseWriter, r *http.Request) {
	// Отримуємо ID товару з параметрів запиту
	id := r.URL.Query().Get("id")
    objectId, err := primitive.ObjectIDFromHex(id)
    if err != nil {
        http.Error(w, "Невірний формат ID", http.StatusBadRequest)
        return
    }
    collection := db.Client.Database("store").Collection("products")
    _, err = collection.DeleteOne(context.TODO(), bson.M{"_id": objectId})
	if err != nil {
		http.Error(w, "Не вдалося видалити товар", http.StatusInternalServerError)
		return
	}

	w.Write([]byte("Товар успішно видалено"))
}