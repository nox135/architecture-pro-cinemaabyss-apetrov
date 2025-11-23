package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	kafka "github.com/segmentio/kafka-go"
)

var (
	kafkaBroker  = getenv("KAFKA_BROKERS", "kafka:9092")
	topicUser    = "user-events"
	topicPayment = "payment-events"
	topicMovie   = "movie-events"
)

func getenv(key, def string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return def
}

// Producer
func produceEvent(topic, eventType string) {
	w := kafka.NewWriter(kafka.WriterConfig{
		Brokers:  []string{kafkaBroker},
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	})
	defer w.Close()

	msg := kafka.Message{
		Key:   []byte(fmt.Sprintf("%d", time.Now().UnixNano())),
		Value: []byte(fmt.Sprintf("Event %s at %s", eventType, time.Now())),
	}

	if err := w.WriteMessages(context.Background(), msg); err != nil {
		log.Println("Failed to write message:", err)
	} else {
		log.Println("Produced:", string(msg.Value))
	}
}

// Consumer
func consumeEvents(topic string) {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{kafkaBroker},
		Topic:   topic,
		GroupID: "events-service-group",
	})
	go func() {
		for {
			m, err := r.ReadMessage(context.Background())
			if err != nil {
				log.Println("Error reading message:", err)
				continue
			}
			log.Println("Consumed:", string(m.Value))
		}
	}()
}

func eventHandler(topic, eventType string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		produceEvent(topic, eventType)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"status": "success"}`))
	}
}

func main() {
	// Запускаем consumers для всех топиков
	consumeEvents(topicUser)
	consumeEvents(topicPayment)
	consumeEvents(topicMovie)

	port := getenv("PORT", "8082")
	http.HandleFunc("/api/events/user", eventHandler(topicUser, "User"))
	http.HandleFunc("/api/events/payment", eventHandler(topicPayment, "Payment"))
	http.HandleFunc("/api/events/movie", eventHandler(topicMovie, "Movie"))
	http.HandleFunc("/api/events/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": true}`))
	})
	//todo исправить тесты
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": true}`))
	})

	log.Println("Events service listening on port", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
