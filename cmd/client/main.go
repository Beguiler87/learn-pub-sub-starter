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
		log.Fatalf("Error encountered: %v", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			log.Printf("Error encountered: %v", err)
		}
	}()
	un, err := gamelogic.ClientWelcome()
	if err != nil {
		log.Fatalf("welcome error: %v", err)
	}
	userN := strings.ToLower(un)
	queueN := "pause." + userN
	ch, q, err := pubsub.DeclareAndBind(conn, routing.ExchangePerilDirect, queueN, routing.PauseKey, pubsub.Transient)
	if err != nil {
		log.Fatalf("declare/bind error: %v", err)
	}
	gs := gamelogic.NewGameState(userN)
	defer ch.Close()
	for {
		parts := gamelogic.GetInput()
		if len(parts) == 0 {
			continue
		}
		cmd, words := parts[0], parts[1:]
		if cmd == "" {
			continue
		}
		switch cmd {
		case "spawn":
			if len(words) != 2 {
				fmt.Println("usage: spawn <location> <unitType>")
				break
			}
			if err := gs.CommandSpawn(append([]string{"spawn"}, words...)); err != nil {
				log.Printf("Error encountered: %v", err)
				break
			}
		case "move":
			if len(words) < 2 {
				fmt.Println("usage: move <destination> <unitID>")
				break
			}
			if _, err := gs.CommandMove(append([]string{"move"}, words...)); err != nil {
				log.Printf("Error encountered: %v", err)
				break
			}
			fmt.Println("Move successful")
		case "status":
			gs.CommandStatus()
		case "help":
			gamelogic.PrintClientHelp()
		case "spam":
			fmt.Println("Spamming is not allowed yet!")
		case "quit":
			gamelogic.PrintQuit()
			return
		default:
			fmt.Println("Unknown command")
		}
	}
	_ = q
	_ = queueN
}
