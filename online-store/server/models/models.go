package models

import "go.mongodb.org/mongo-driver/bson/primitive"

// Модель для товару
type Product struct {
	ID          string  `bson:"_id,omitempty" json:"ID"`
	Name        string  `bson:"name" json:"Name"`
	Image       string  `bson:"image" json:"Image"`
	Description string  `bson:"description" json:"Description"`
	Price       float64 `bson:"price" json:"Price"`
	Stock       int     `bson:"stock" json:"Stock"`
}

// Модель для елементів замовлення
type OrderItem struct {
	ProductID string  `bson:"productId" json:"ProductID"`
	Quantity  int     `bson:"quantity" json:"Quantity"`
	Price     float64 `bson:"price" json:"Price"`
}

// Модель для замовлення
type Order struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"ID"`
	UserID         string             `bson:"user_id" json:"UserID"`
	Products       []OrderProduct     `bson:"products" json:"Products"`
	Total          float64            `bson:"total" json:"Total"`
	DeliveryMethod string             `bson:"delivery_method" json:"DeliveryMethod"`
	Address        string             `bson:"address" json:"Address"`
	PaymentMethod  string             `bson:"payment_method" json:"PaymentMethod"`
	FullName       string             `bson:"full_name" json:"FullName"`
	Phone          string             `bson:"phone" json:"Phone"`
	Email          string             `bson:"email" json:"Email"` // Додано поле Email
	Date           string             `bson:"date" json:"Date"`
}

// Модель для товару в замовленні
type OrderProduct struct {
	ProductID string  `bson:"product_id" json:"ProductID"`
	Quantity  int     `bson:"quantity" json:"Quantity"`
	Price     float64 `bson:"price" json:"Price"`
}
