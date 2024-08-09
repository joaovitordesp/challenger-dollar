package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

	_ "modernc.org/sqlite"
)

type ExchangeRate struct {
	Bid string `json:"bid"`
}

func main() {
	db, err := sql.Open("sqlite", "./exchange.db")
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}
	defer db.Close()

	createTableSQL := `CREATE TABLE IF NOT EXISTS exchange (id INTEGER PRIMARY KEY AUTOINCREMENT, bid TEXT, timestamp DATETIME DEFAULT CURRENT_TIMESTAMP)`
	_, err = db.Exec(createTableSQL)
	if err != nil {
		log.Fatalf("Error creating table: %v", err)
	}

	http.HandleFunc("/cotacao", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Received request for /cotacao")

		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, "GET", "https://economia.awesomeapi.com.br/json/last/USD-BRL", nil)
		if err != nil {
			log.Printf("Error creating API request: %v", err)
			http.Error(w, "Failed to create request", http.StatusInternalServerError)
			return
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Printf("Error making API request: %v", err)
			http.Error(w, "Failed to fetch exchange rate", http.StatusRequestTimeout)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Printf("API request failed with status: %d", resp.StatusCode)
			http.Error(w, "API request failed", http.StatusInternalServerError)
			return
		}

		var result map[string]ExchangeRate
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			log.Printf("Error decoding API response: %v", err)
			http.Error(w, "Failed to decode response", http.StatusInternalServerError)
			return
		}

		bid := result["USDBRL"].Bid
		log.Printf("Received exchange rate: %s", bid)

		saveCtx, saveCancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer saveCancel()

		_, err = db.ExecContext(saveCtx, "INSERT INTO exchange (bid) VALUES (?)", bid)
		if err != nil {
			log.Printf("Error saving exchange rate to database: %v", err)
			http.Error(w, "Failed to save exchange rate", http.StatusInternalServerError)
			return
		}

		log.Println("Exchange rate saved to database successfully")

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(bid); err != nil {
			log.Printf("Error encoding response: %v", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}

		log.Println("Response sent to client successfully")
	})

	log.Println("Server started at :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
