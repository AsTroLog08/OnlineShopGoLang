package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"text/template"
	"time"

	"online-store/server/db"
	"online-store/server/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Обробник для відображення форми оформлення замовлення
func CheckoutPageHandler(w http.ResponseWriter, r *http.Request) {
    // Отримуємо дані з кошика
    cartLock.Lock()
    defer cartLock.Unlock()

    // Підраховуємо кількість товарів та загальну суму
    var totalPrice float64
    var itemCount int
    for id, quantity := range Cart {
        // Отримуємо товар з бази даних
        collection := db.Client.Database("store").Collection("products")
        var product models.Product
        err := collection.FindOne(context.TODO(), bson.M{"_id": id}).Decode(&product)
        if err != nil {
            continue
        }
        totalPrice += product.Price * float64(quantity)
        itemCount += quantity
    }

    // Рендеримо шаблон order.html і передаємо дані
    tmpl, err := template.ParseFiles("web/templates/order.html")
    if err != nil {
        http.Error(w, "Не вдалося завантажити шаблон", http.StatusInternalServerError)
        return
    }

    data := struct {
        ItemCount  int
        TotalPrice float64
    }{
        ItemCount:  itemCount,
        TotalPrice: totalPrice,
    }

    tmpl.Execute(w, data)
}
// CreateOrderHandler обробляє створення замовлення
func CreateOrderHandler(w http.ResponseWriter, r *http.Request) {
    var orderData struct {
        FullName       string `json:"fullName"`
        Email          string `json:"email"`
        Phone          string `json:"phone"`
        DeliveryMethod string `json:"deliveryMethod"`
        Address        string `json:"address"`
        PaymentMethod  string `json:"paymentMethod"`
    }

    // Декодуємо запит
    err := json.NewDecoder(r.Body).Decode(&orderData)
    if err != nil {
        http.Error(w, "Невірний формат даних", http.StatusBadRequest)
        return
    }

    // Перевіряємо, чи є товари в кошику
    cartLock.Lock()
    defer cartLock.Unlock()

    if len(Cart) == 0 {
        http.Error(w, "Кошик порожній", http.StatusBadRequest)
        return
    }

    var order models.Order
    order.UserID = "test_user" // Призначити користувача
    order.Date = time.Now().Format("2006-01-02")
    order.Products = []models.OrderProduct{}
    order.Total = 0

    collection := db.Client.Database("store").Collection("products")

    for id, quantity := range Cart {
        objectID, err := primitive.ObjectIDFromHex(id)
        if err != nil {
            log.Println("Невірний формат ID:", id)
            continue
        }

        var product models.Product
        err = collection.FindOne(context.TODO(), bson.M{"_id": objectID}).Decode(&product)
        if err != nil {
            log.Println("Товар не знайдений, id:", id)
            continue
        }

        // Перевіряємо, чи достатньо товару на складі
        if product.Stock < quantity {
            http.Error(w, "Недостатньо товару на складі: "+product.Name, http.StatusBadRequest)
            return
        }

        // Оновлюємо кількість товару на складі
        _, err = collection.UpdateOne(
            context.TODO(),
            bson.M{"_id": objectID},
            bson.M{"$inc": bson.M{"stock": -quantity}}, // Зменшуємо кількість на складі
        )
        if err != nil {
            log.Println("Не вдалося оновити склад для товару, id:", id)
            http.Error(w, "Не вдалося оновити склад для товару: "+product.Name, http.StatusInternalServerError)
            return
        }

        log.Printf("Товар оновлено: %s, новий залишок: %d\n", product.Name, product.Stock-quantity)

        // Додаємо товар у масив продуктів замовлення
        order.Products = append(order.Products, models.OrderProduct{
            ProductID: id,
            Quantity:  quantity,
            Price:     product.Price,
        })

        // Оновлюємо загальну суму замовлення
        order.Total += product.Price * float64(quantity)
    }

    // Додаємо інформацію про замовлення
    order.FullName = orderData.FullName
    order.Email = orderData.Email
    order.Phone = orderData.Phone
    order.DeliveryMethod = orderData.DeliveryMethod
    order.Address = orderData.Address
    order.PaymentMethod = orderData.PaymentMethod

    // Зберігаємо замовлення в базі даних
    ordersCollection := db.Client.Database("store").Collection("orders")
    _, err = ordersCollection.InsertOne(context.TODO(), order)
    if err != nil {
        http.Error(w, "Не вдалося створити замовлення", http.StatusInternalServerError)
        return
    }

    // Очищаємо кошик після оформлення замовлення
    Cart = make(map[string]int)

    w.WriteHeader(http.StatusCreated)
    w.Write([]byte("Замовлення успішно створено"))
}

// Отримання всіх замовлень
func GetOrdersHandler(w http.ResponseWriter, r *http.Request) {
	collection := db.Client.Database("store").Collection("orders")

	cursor, err := collection.Find(context.TODO(), bson.D{})
	if err != nil {
		http.Error(w, "Не вдалося отримати замовлення", http.StatusInternalServerError)
		return
	}
	defer cursor.Close(context.TODO())

	var orders []models.Order
	for cursor.Next(context.TODO()) {
		var order models.Order
		cursor.Decode(&order)
		orders = append(orders, order)
	}

	json.NewEncoder(w).Encode(orders)
}
// Обробник для сторінки оплати
func PaymentPageHandler(w http.ResponseWriter, r *http.Request) {
    tmpl, err := template.ParseFiles("web/templates/payment.html")
    if err != nil {
        http.Error(w, "Не вдалося завантажити шаблон сторінки оплати", http.StatusInternalServerError)
        return
    }

    tmpl.Execute(w, nil)
}
