package analytics

import (
	"context"
	"math"
	"sort"
	"sync"
	"time"

	"online-store/server/db"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type AnalyticsResult struct {
	DailyPurchases     map[string]int `json:"dailyPurchases"`
	MinPurchaseDay     string         `json:"minPurchaseDay"`
	MaxPurchaseDay     string         `json:"maxPurchaseDay"`
	AverageCheck       float64        `json:"averageCheck"`
	MedianCheck        float64        `json:"medianCheck"`
	MostPurchased      string         `json:"mostPurchased"`
	MostFrequentCombo  string         `json:"mostFrequentCombo"`
	LeastFrequentCombo string         `json:"leastFrequentCombo"`
}

func CalculateAnalytics(ctx context.Context) (*AnalyticsResult, error) {
	// Set timeout
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Concurrent data retrieval channels
	ordersChan := make(chan []bson.M, 1)
	productsChan := make(chan map[string]string, 1)
	errChan := make(chan error, 2)

	// Concurrent goroutines for data retrieval
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		orders, err := retrieveOrders(ctx)
		if err != nil {
			errChan <- err
			return
		}
		ordersChan <- orders
	}()

	go func() {
		defer wg.Done()
		products, err := retrieveProducts(ctx)
		if err != nil {
			errChan <- err
			return
		}
		productsChan <- products
	}()

	// Wait for goroutines and close error channel
	go func() {
		wg.Wait()
		close(errChan)
		close(ordersChan)
		close(productsChan)
	}()

	// Check for errors
	select {
	case err := <-errChan:
		if err != nil {
			return nil, err
		}
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Retrieve data from channels
	orders := <-ordersChan
	productMap := <-productsChan

	// Concurrent analytics calculation
	var (
		dailyPurchasesMu sync.Mutex
		checksMu         sync.Mutex
		productCountMu   sync.Mutex
		comboCountMu     sync.Mutex
	)

	dailyPurchases := make(map[string]int)
	checks := []float64{}
	productCount := make(map[string]int)
	comboCount := make(map[string]int)

	// Process orders concurrently
	processChan := make(chan bson.M, 10)
	var processWg sync.WaitGroup

	// Multiple worker goroutines for order processing
	for i := 0; i < 4; i++ {
		processWg.Add(1)
		go func() {
			defer processWg.Done()
			for order := range processChan {
				processOrder(
					order, 
					productMap, 
					&dailyPurchasesMu, &checksMu, &productCountMu, &comboCountMu,
					&dailyPurchases, &checks, &productCount, &comboCount,
				)
			}
		}()
	}

	// Send orders to processing channel
	for _, order := range orders {
		processChan <- order
	}
	close(processChan)
	processWg.Wait()

	// Calculate statistics
	averageCheck := calculateAverageCheck(checks)
	medianCheck := calculateMedianCheck(checks)
	mostPurchased := findMostPurchasedProduct(productCount, productMap)
	mostFrequentCombo, leastFrequentCombo := findFrequentCombos(comboCount)

	return &AnalyticsResult{
		DailyPurchases:     dailyPurchases,
		MinPurchaseDay:     findMinKey(dailyPurchases),
		MaxPurchaseDay:     findMaxKey(dailyPurchases),
		AverageCheck:       averageCheck,
		MedianCheck:        medianCheck,
		MostPurchased:      mostPurchased,
		MostFrequentCombo:  mostFrequentCombo,
		LeastFrequentCombo: leastFrequentCombo,
	}, nil
}

func retrieveOrders(ctx context.Context) ([]bson.M, error) {
	ordersCollection := db.Client.Database("store").Collection("orders")
	findOptions := options.Find().SetLimit(10000)
	cursor, err := ordersCollection.Find(ctx, bson.M{}, findOptions)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var orders []bson.M
	if err = cursor.All(ctx, &orders); err != nil {
		return nil, err
	}
	return orders, nil
}

func retrieveProducts(ctx context.Context) (map[string]string, error) {
	productsCollection := db.Client.Database("store").Collection("products")
	findOptions := options.Find().SetLimit(5000)
	cursor, err := productsCollection.Find(ctx, bson.M{}, findOptions)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	productMap := make(map[string]string)
	var product bson.M
	for cursor.Next(ctx) {
		if err := cursor.Decode(&product); err != nil {
			return nil, err
		}
		productMap[product["_id"].(primitive.ObjectID).Hex()] = product["name"].(string)
	}
	return productMap, nil
}

func processOrder(
	order bson.M, 
	productMap map[string]string,
	dailyPurchasesMu, checksMu, productCountMu, comboCountMu *sync.Mutex,
	dailyPurchases *map[string]int, 
	checks *[]float64, 
	productCount *map[string]int, 
	comboCount *map[string]int,
) {
	// Process daily purchases
	date := order["date"].(string)
	dailyPurchasesMu.Lock()
	(*dailyPurchases)[date]++
	dailyPurchasesMu.Unlock()

	// Process check amount
	check := order["total"].(float64)
	checksMu.Lock()
	*checks = append(*checks, check)
	checksMu.Unlock()

	// Process product combinations
	products := order["products"].(primitive.A)
	var combo []string
	productCountMu.Lock()
	for _, p := range products {
		product := p.(bson.M)
		productID := product["product_id"].(string)
		(*productCount)[productID]++
		combo = append(combo, productMap[productID])
	}
	productCountMu.Unlock()

	// Process product combinations
	if len(combo) >= 2 {
		sort.Strings(combo)
		comboKey := ""
		for _, c := range combo {
			comboKey += c + ","
		}
		
		comboCountMu.Lock()
		(*comboCount)[comboKey]++
		comboCountMu.Unlock()
	}
}

func calculateAverageCheck(checks []float64) float64 {
	var totalCheck float64
	for _, check := range checks {
		totalCheck += check
	}
	return totalCheck / float64(len(checks))
}

func calculateMedianCheck(checks []float64) float64 {
	sort.Float64s(checks)
	midIndex := len(checks) / 2
	if len(checks)%2 == 0 {
		return (checks[midIndex-1] + checks[midIndex]) / 2
	}
	return checks[midIndex]
}

func findMostPurchasedProduct(productCount map[string]int, productMap map[string]string) string {
	var mostPurchased string
	maxCount := 0
	for productID, count := range productCount {
		if count > maxCount {
			maxCount = count
			if productMap[productID] != "" {
				mostPurchased = productMap[productID]
			}
		}
	}
	return mostPurchased
}

func findFrequentCombos(comboCount map[string]int) (string, string) {
	var mostFrequentCombo, leastFrequentCombo string
	maxComboCount := 0
	minComboCount := math.MaxInt

	for combo, count := range comboCount {
		if count > maxComboCount {
			maxComboCount = count
			mostFrequentCombo = combo
		}
		if count < minComboCount {
			minComboCount = count
			leastFrequentCombo = combo
		}
	}

	return mostFrequentCombo, leastFrequentCombo
}

// Existing findMinKey and findMaxKey functions remain unchanged
func findMinKey(data map[string]int) string {
	minKey := ""
	minValue := math.MaxInt
	for key, value := range data {
		if value < minValue {
			minKey = key
			minValue = value
		}
	}
	return minKey
}

func findMaxKey(data map[string]int) string {
	maxKey := ""
	maxValue := 0
	for key, value := range data {
		if value > maxValue {
			maxKey = key
			maxValue = value
		}
	}
	return maxKey
}