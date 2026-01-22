package main

import (
	"log"
	"net/http"
	"os"

	"email-verification/internal/api"
	"email-verification/internal/config"
)

func main() {
	config.LoadEnvFile(".env")

	port := os.Getenv("API_PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", api.HealthHandler)
	mux.HandleFunc("/verify", api.VerifyHandler)

	handler := api.Logging(api.Recovery(mux))

	log.Printf("Starting API server on :%s", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
