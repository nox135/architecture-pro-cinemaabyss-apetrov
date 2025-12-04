package main

import (
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"
)

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// Прокси запрос
func proxyRequest(target string, w http.ResponseWriter, r *http.Request) {
	// Создаем новый HTTP запрос
	req, err := http.NewRequest(r.Method, target, r.Body)
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	// копируем query параметры!!!
	req.URL.RawQuery = r.URL.RawQuery

	// Копируем заголовки
	for name, values := range r.Header {
		for _, v := range values {
			req.Header.Add(name, v)
		}
	}

	// Пробрасываем запрос
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, "Failed to proxy request", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Прокидываем статус
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func main() {
	rand.Seed(time.Now().UnixNano())

	monolithURL := getenv("MONOLITH_URL", "http://monolith:8080")
	moviesURL := getenv("MOVIES_SERVICE_URL", "http://movies-service:8081")
	eventsURL := getenv("EVENTS_SERVICE_URL", "http://events-service:8082")
	migrationPercentStr := getenv("MOVIES_MIGRATION_PERCENT", "50")
	migrationPercent, _ := strconv.Atoi(migrationPercentStr)

	// Health check
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": true}`))
	})

	// Users -> монолит
	http.HandleFunc("/api/users", func(w http.ResponseWriter, r *http.Request) {
		proxyRequest(monolithURL+"/api/users", w, r)
	})

	// Movies с миграцией
	http.HandleFunc("/api/movies", func(w http.ResponseWriter, r *http.Request) {
		if rand.Intn(100) < migrationPercent {
			proxyRequest(moviesURL+"/api/movies", w, r)
		} else {
			proxyRequest(monolithURL+"/api/movies", w, r)
		}
	})

	// Events -> events-service
	http.HandleFunc("/api/events/user", func(w http.ResponseWriter, r *http.Request) {
		proxyRequest(eventsURL+"/api/events/user", w, r)
	})
	http.HandleFunc("/api/events/payment", func(w http.ResponseWriter, r *http.Request) {
		proxyRequest(eventsURL+"/api/events/payment", w, r)
	})
	http.HandleFunc("/api/events/movie", func(w http.ResponseWriter, r *http.Request) {
		proxyRequest(eventsURL+"/api/events/movie", w, r)
	})

	port := getenv("PORT", "8000")
	log.Println("Proxy service listening on port", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
