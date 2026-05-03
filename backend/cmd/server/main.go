package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"umamusume-fan-point/backend/internal/api"
	"umamusume-fan-point/backend/internal/excel"
	"umamusume-fan-point/backend/internal/mongodb"
)

func main() {
	dataFile := getenv("DATA_FILE", filepath.Join("..", "source.xlsx"))
	addr := listenAddr()
	staticDir := os.Getenv("STATIC_DIR")
	mongoURI := os.Getenv("MONGO_URI")
	mongoDatabase := getenv("MONGO_DATABASE", "umamusume_fan_point")

	parser := excel.NewParser(dataFile)
	loader := api.Loader(parser)
	if mongoURI != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		store, err := mongodb.New(ctx, mongoURI, mongoDatabase)
		if err != nil {
			cancel()
			log.Fatal(err)
		}
		if err := store.SeedIfEmpty(ctx, parser); err != nil {
			cancel()
			log.Fatal(err)
		}
		cancel()
		defer func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := store.Close(ctx); err != nil {
				log.Printf("close mongo: %v", err)
			}
		}()
		loader = store
		log.Printf("MongoDB enabled, database=%s", mongoDatabase)
	}
	handler := api.NewHandler(loader, 30*time.Second)

	mux := http.NewServeMux()
	handler.Register(mux)

	if staticDir != "" {
		mux.Handle("/", http.FileServer(http.Dir(staticDir)))
	}

	log.Printf("fan point API listening on %s, data=%s", addr, dataFile)
	if err := http.ListenAndServe(addr, withCORS(mux)); err != nil {
		log.Fatal(err)
	}
}

func getenv(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func listenAddr() string {
	if port := os.Getenv("PORT"); port != "" {
		return ":" + port
	}
	return getenv("ADDR", ":8080")
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
