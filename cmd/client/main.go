package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"

	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	fmt.Println("Starting Peril client...")
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		log.Fatalf("Error encountered: %v", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			log.Printf("Error encountered: %v", err)
		}
	}()
	un, err := gamelogic.ClientWelcome()
	userN := strings.ToLower(un)
	queueN := "pause." + userN
	ch, q, err := pubsub.DeclareAndBind(conn, routing.ExchangePerilDirect, queueN, routing.PauseKey, pubsub.Transient)
	if err != nil {
		log.Fatalf("declare/bind error: %v", err)
	}
	defer ch.Close()
	_ = q
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	<-sig
}
