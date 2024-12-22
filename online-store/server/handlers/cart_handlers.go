package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"online-store/server/db"
	"online-store/server/models"
	"strconv"
	"sync"
	"text/template"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type CartItem struct {
	ProductID string `json:"productId"`
	Quantity  int    `json:"quantity"`
}

var Cart = make(map[string]int) // Ключ: ID товару, значення: кількість
var cartLock sync.Mutex         // Для синхронізації доступу

func AddToCartHandler(w http.ResponseWriter, r *http.Request) {

    cartLock.Lock()
    defer cartLock.Unlock()

    // Отримуємо ID товару та кількість
    id := r.URL.Query().Get("id")
    log.Printf("Product ID received: %s", id)

    if id == "" {
        http.Error(w, "ID товару не вказано", http.StatusBadRequest)
        return
    }

    quantity := 1
    if q := r.URL.Query().Get("quantity"); q != "" {
        parsedQuantity, err := strconv.Atoi(q)
        if err != nil || parsedQuantity <= 0 {
            http.Error(w, "Невірна кількість", http.StatusBadRequest)
            return
        }
        quantity = parsedQuantity
    }

    // Отримуємо товар з бази даних
    collection := db.Client.Database("store").Collection("products")
    
    // Convert string ID to MongoDB ObjectID
    objectID, err := primitive.ObjectIDFromHex(id)
    if err != nil {
        log.Printf("Invalid ObjectID: %v", err)
        http.Error(w, "Невірний формат ID", http.StatusBadRequest)
        return
    }

    var product models.Product
    err = collection.FindOne(context.TODO(), bson.M{"_id": objectID}).Decode(&product)
    if err != nil {
        log.Printf("Error finding product: %v", err)
        http.Error(w, "Товар не знайдено", http.StatusNotFound)
        return
    }

    // Перевіряємо, чи можна додати в кошик
    if Cart[id]+quantity > product.Stock {
        http.Error(w, "Недостатньо товару на складі", http.StatusBadRequest)
        return
    }

    Cart[id] += quantity
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("Товар додано до кошика"))
}

func ViewCartHandler(w http.ResponseWriter, r *http.Request) {
    cartLock.Lock()
    defer cartLock.Unlock()

    // Розширена структура для відображення
    type CartViewItem struct {
        ProductID   string
        Name        string
        Image       string
        Price       float64
        Quantity    int
        TotalPrice  float64
    }
    type CartView struct {
        Items      []CartViewItem
        TotalPrice float64
    }

    cartView := CartView{Items: []CartViewItem{}, TotalPrice: 0}

    // Отримуємо інформацію про товари
    collection := db.Client.Database("store").Collection("products")
    for id, quantity := range Cart {
        // Конвертуємо рядковий ID в ObjectID
        objectID, err := primitive.ObjectIDFromHex(id)
        if err != nil {
            log.Printf("Invalid ObjectID: %v", err)
            continue
        }

        var product models.Product
        err = collection.FindOne(context.TODO(), bson.M{"_id": objectID}).Decode(&product)
        if err != nil {
            log.Printf("Error finding product: %v", err)
            continue
        }

        totalPrice := float64(quantity) * product.Price
        cartView.Items = append(cartView.Items, CartViewItem{
            ProductID:  id,
            Name:       product.Name,
            Image:      product.Image,
            Price:      product.Price,
            Quantity:   quantity,
            TotalPrice: totalPrice,
        })
        cartView.TotalPrice += totalPrice
    }

    // Рендеринг шаблону
    tmpl, err := template.ParseFiles("web/templates/cart.html")
    if err != nil {
        http.Error(w, "Не вдалося завантажити шаблон", http.StatusInternalServerError)
        return
    }
    tmpl.Execute(w, cartView)
}
func UpdateCartItemHandler(w http.ResponseWriter, r *http.Request) {
    cartLock.Lock()
    defer cartLock.Unlock()

    // Отримання ID товару з запиту
    id := r.URL.Query().Get("id")
    if id == "" {
        http.Error(w, "ID товару не вказано", http.StatusBadRequest)
        return
    }

    // Отримання кількості з запиту
    quantity, err := strconv.Atoi(r.URL.Query().Get("quantity"))
    if err != nil || quantity <= 0 {
        http.Error(w, "Невірна кількість", http.StatusBadRequest)
        return
    }

    // Отримуємо товар з бази даних
    collection := db.Client.Database("store").Collection("products")
    objectID, err := primitive.ObjectIDFromHex(id)
    if err != nil {
        log.Printf("Invalid ObjectID: %v", err)
        http.Error(w, "Невірний формат ID", http.StatusBadRequest)
        return
    }

    var product models.Product
    err = collection.FindOne(context.TODO(), bson.M{"_id": objectID}).Decode(&product)
    if err != nil {
        log.Printf("Error finding product: %v", err)
        http.Error(w, "Товар не знайдено", http.StatusNotFound)
        return
    }

    // Якщо кількість перевищує доступну, обмежуємо її
    if quantity > product.Stock {
        quantity = product.Stock
    }

    // Оновлення кількості товару в кошику
    Cart[id] = quantity

    // Обчислення нової загальної ціни для товару
    totalPrice := float64(quantity) * product.Price

    // Обчислення загальної вартості кошика
    cartTotalPrice := 0.0
    for cartID, cartQuantity := range Cart {
        // Отримання кожного товару з бази даних
        cartObjectID, err := primitive.ObjectIDFromHex(cartID)
        if err != nil {
            log.Printf("Invalid ObjectID in cart: %v", err)
            continue
        }

        var cartProduct models.Product
        err = collection.FindOne(context.TODO(), bson.M{"_id": cartObjectID}).Decode(&cartProduct)
        if err != nil {
            log.Printf("Error finding product in cart: %v", err)
            continue
        }

        cartTotalPrice += float64(cartQuantity) * cartProduct.Price
    }

    // Якщо кількість перевищує доступну, обмежуємо її
    message := ""
    if quantity > product.Stock {
        quantity = product.Stock
        message = "Вказана кількість перевищує доступний залишок. Кількість змінено на максимальну доступну."
    }


    // Повернення оновлених даних про товар та кошик
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "productId":      id,
        "quantity":       quantity,
        "price":          product.Price,
        "totalPrice":     totalPrice,
        "cartTotalPrice": cartTotalPrice,
        "availableStock": product.Stock,
        "message":        message, // Передаємо повідомлення
    })

}



func DeleteCartItemHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodDelete {
        http.Error(w, "Метод не підтримується", http.StatusMethodNotAllowed)
        return
    }

    id := r.URL.Query().Get("id")
    if id == "" {
        http.Error(w, "ID товару не вказано", http.StatusBadRequest)
        return
    }

    cartLock.Lock()
    defer cartLock.Unlock()

    if _, exists := Cart[id]; !exists {
        http.Error(w, "Товар не знайдено в кошику", http.StatusNotFound)
        return
    }

    delete(Cart, id)
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("Товар успішно видалено з кошика"))
}