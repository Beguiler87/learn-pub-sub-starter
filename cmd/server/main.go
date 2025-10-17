package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"

	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	fmt.Println("Starting Peril server...")
	conString := "amqp://guest:guest@localhost:5672/"
	conn, err := amqp.Dial(conString)
	if err != nil {
		log.Fatalf("Error encountered: %v", err)
	}
	defer func() {
		err := conn.Close()
		if err != nil {
			log.Printf("Error encountered: %v", err)
		}
	}()
	fmt.Println("Connection successful.")
	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("channel error: %v", err)
	}
	defer ch.Close()
	err = ch.ExchangeDeclare(
		"peril_direct",
		"direct",
		true,
		false,
		false,
		false,
		nil,
	)
	payload := routing.PlayingState{IsPaused: true}
	if err := pubsub.PublishJSON(ch, routing.ExchangePerilDirect, routing.PauseKey, payload); err != nil {
		log.Printf("publish error: %v", err)
	}
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	<-sigChan
	fmt.Println("Closing program...")
}
