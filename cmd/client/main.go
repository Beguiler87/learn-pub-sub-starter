package main

import (
	"fmt"
	"log"
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
		log.Fatalf("dial error: %v", err)
	}
	defer conn.Close()
	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("channel error: %v", err)
	}
	defer ch.Close()
	un, err := gamelogic.ClientWelcome()
	if err != nil {
		log.Fatalf("welcome error: %v", err)
	}
	username := strings.ToLower(un)
	gs := gamelogic.NewGameState(username)
	queueName := "pause." + username
	if err := pubsub.SubscribeJSON[routing.PlayingState](
		conn,
		routing.ExchangePerilDirect,
		queueName,
		routing.PauseKey,
		pubsub.SimpleQueueTransient,
		handlerPause(gs),
	); err != nil {
		log.Fatalf("subscribe error: %v", err)
	}
	if err = pubsub.SubscribeJSON[gamelogic.ArmyMove](
		conn,
		routing.ExchangePerilTopic,
		"army_moves."+username,
		"army_moves.*",
		pubsub.SimpleQueueTransient,
		handlerMove(gs, ch),
	); err != nil {
		log.Fatalf("subscribe error: %v", err)
	}
	if err = pubsub.SubscribeJSON[gamelogic.RecognitionOfWar](
		conn,
		routing.ExchangePerilTopic,
		"war",
		routing.WarRecognitionsPrefix+".*",
		pubsub.SimpleQueueDurable,
		func(m gamelogic.RecognitionOfWar) pubsub.AckType {
			return handlerWar(gs, ch, routing.ExchangePerilTopic, m)
		},
	); err != nil {
		log.Fatalf("subscribe error: %v", err)
	}
	for {
		parts := gamelogic.GetInput()
		if len(parts) == 0 {
			continue
		}
		cmd, words := parts[0], parts[1:]
		switch cmd {
		case "spawn":
			if len(words) != 2 {
				fmt.Println("usage: spawn <location> <unitType>")
				break
			}
			if err := gs.CommandSpawn(append([]string{"spawn"}, words...)); err != nil {
				log.Printf("error: %v", err)
			}
		case "move":
			if len(words) < 2 {
				fmt.Println("usage: move <destination> <unitID>")
				break
			}
			mv, err := gs.CommandMove(append([]string{"move"}, words...))
			if err != nil {
				log.Printf("error: %v", err)
				break
			}
			if err := pubsub.PublishJSON(ch, routing.ExchangePerilTopic, "army_moves."+username, mv); err != nil {
				log.Printf("publish error: %v", err)
			} else {
				log.Printf("published move to %s", "army_moves."+username)
			}
			fmt.Println("Move successful")
		case "status":
			gs.CommandStatus()
		case "help":
			gamelogic.PrintClientHelp()
		case "quit":
			gamelogic.PrintQuit()
			return
		default:
			fmt.Println("Unknown command")
		}
	}
}
