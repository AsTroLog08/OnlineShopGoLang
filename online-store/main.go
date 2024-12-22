package main

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"online-store/server/db"
	"online-store/server/handlers"
)

func main() {
	// Підключення до бази даних
	db.Connect("mongodb://localhost:27017")

	// Ініціалізація маршрутизатора
	router := mux.NewRouter()

	// Маршрути
	router.HandleFunc("/", handlers.HomeHandler).Methods("GET")
	router.HandleFunc("/products", handlers.GetProductsHandler).Methods("GET")
	router.HandleFunc("/products/add", handlers.AddProductHandler).Methods("POST")
	router.HandleFunc("/products/delete", handlers.DeleteProductHandler).Methods("DELETE")
	router.HandleFunc("/product/{id}", handlers.GetProductHandler).Methods("GET")
	router.HandleFunc("/cart/add", handlers.AddToCartHandler).Methods("POST")
	router.HandleFunc("/cart", handlers.ViewCartHandler).Methods("GET")	
	router.HandleFunc("/orders", handlers.GetOrdersHandler).Methods("GET")
	router.HandleFunc("/cart/checkout", handlers.CreateOrderHandler).Methods("POST")
	router.HandleFunc("/cart/checkout", handlers.CheckoutPageHandler).Methods("GET")
	router.HandleFunc("/cart/update", handlers.UpdateCartItemHandler)
	router.HandleFunc("/cart/delete", handlers.DeleteCartItemHandler)

	router.HandleFunc("/payment", handlers.PaymentPageHandler)

	router.HandleFunc("/analytics", handlers.AnalyticsHandler).Methods("GET")
	router.HandleFunc("/analytics/generate", handlers.GenerateOrdersHandler).Methods("POST")
	

	// Запуск сервера
	log.Println("Сервер запущено на http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}
