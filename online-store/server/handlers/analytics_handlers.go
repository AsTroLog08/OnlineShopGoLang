package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"online-store/server/analytics"
	"text/template"
	"time"
)

func GenerateOrdersHandler(w http.ResponseWriter, r *http.Request) {
	err := analytics.GenerateRandomOrders()
	if err != nil {
		http.Error(w, "Не вдалося згенерувати замовлення", http.StatusInternalServerError)
		return
	}
	w.Write([]byte("100 замовлень успішно згенеровано"))
}

func AnalyticsHandler(w http.ResponseWriter, r *http.Request) {
	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	// Обчислюємо аналітику з контекстом
	results, err := analytics.CalculateAnalytics(ctx)
	if err != nil {
		http.Error(w, "Не вдалося отримати аналітику: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Серіалізуємо DailyPurchases у JSON
	dailyPurchasesJSON, err := json.Marshal(results.DailyPurchases)
	if err != nil {
		http.Error(w, "Помилка серіалізації DailyPurchases", http.StatusInternalServerError)
		return
	}

	// Передаємо дані у шаблон
	data := struct {
		*analytics.AnalyticsResult
		DailyPurchasesJSON string
	}{
		AnalyticsResult:    results,
		DailyPurchasesJSON: string(dailyPurchasesJSON),
	}

	// Рендеримо HTML-шаблон
	tmpl, err := template.ParseFiles("web/templates/analytics.html")
	if err != nil {
		http.Error(w, "Не вдалося завантажити шаблон", http.StatusInternalServerError)
		return
	}

	tmpl.Execute(w, data)
}