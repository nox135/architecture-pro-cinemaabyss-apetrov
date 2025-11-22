package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
)

var (
	moviesServiceURL   string
	monolithServiceURL string
	migrationPercent   int
)

func init() {
	// Берём адреса сервисов из переменных окружения
	moviesServiceURL = os.Getenv("MOVIES_SERVICE_URL")
	monolithServiceURL = os.Getenv("MONOLITH_SERVICE_URL")
	migrationPercent = 0
	if v := os.Getenv("MOVIES_MIGRATION_PERCENT"); v != "" {
		fmt.Sscanf(v, "%d", &migrationPercent)
	}
}

func proxyHandler(w http.ResponseWriter, r *http.Request) {
	targetURL := monolithServiceURL

	// Решаем, куда отправлять запрос на основе процента миграции
	if rand.Intn(100) < migrationPercent {
		targetURL = moviesServiceURL
	}

	proxyReq, err := http.NewRequest(r.Method, targetURL+r.RequestURI, r.Body)
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	proxyReq.Header = r.Header
	client := &http.Client{}
	resp, err := client.Do(proxyReq)
	if err != nil {
		http.Error(w, "Failed to call target service", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	for k, v := range resp.Header {
		for _, vv := range v {
			w.Header().Add(k, vv)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = w.ReadFrom(resp.Body)
}

func main() {
	http.HandleFunc("/api/movies", proxyHandler)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Proxy service running on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
