package main

import (
	"fmt"
	"log"
	"time"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
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
	if err != nil {
		log.Fatalf("exchange declare error: %v", err)
	}
	err = ch.ExchangeDeclare(
		"peril_topic",
		"topic",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("exchange declare error: %v", err)
	}
	err = pubsub.SubscribeGob[routing.GameLog](
		conn,
		routing.ExchangePerilTopic,
		"game_logs",
		routing.GameLogSlug+".*",
		pubsub.SimpleQueueDurable,
		func(gl routing.GameLog) pubsub.AckType {
			defer fmt.Print("> ")
			gamelogic.WriteLog(gl)
			return pubsub.Ack
		},
	)
	if err != nil {
		log.Fatalf("subscribe error: %v", err)
	}
	gamelogic.PrintServerHelp()
	quit := false
	for {
		words := gamelogic.GetInput()
		if len(words) == 0 {
			continue
		}
		switch words[0] {
		case "pause":
			gamelogic.WriteLog(routing.GameLog{
				CurrentTime: time.Now(),
				Message:     "sending pause",
				Username:    "server",
			})
			payload := routing.PlayingState{IsPaused: true}
			if err := pubsub.PublishJSON(ch, routing.ExchangePerilDirect, routing.PauseKey, payload); err != nil {
				log.Printf("publish error: %v", err)
			}
		case "resume":
			gamelogic.WriteLog(routing.GameLog{
				CurrentTime: time.Now(),
				Message:     "sending resume",
				Username:    "server",
			})
			payload := routing.PlayingState{IsPaused: false}
			if err := pubsub.PublishJSON(ch, routing.ExchangePerilDirect, routing.PauseKey, payload); err != nil {
				log.Printf("publish error: %v", err)
			}
		case "quit":
			gamelogic.WriteLog(routing.GameLog{
				CurrentTime: time.Now(),
				Message:     "sending quit",
				Username:    "server",
			})
			quit = true
		default:
			gamelogic.WriteLog(routing.GameLog{
				CurrentTime: time.Now(),
				Message:     "sending unknown command",
				Username:    "server",
			})
		}
		if quit {
			break
		}
	}
}
