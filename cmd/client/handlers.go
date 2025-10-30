package main

import (
	"fmt"
	"time"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
	amqp "github.com/rabbitmq/amqp091-go"
)

func handlerPause(gs *gamelogic.GameState) func(routing.PlayingState) pubsub.AckType {
	return func(state routing.PlayingState) pubsub.AckType {
		defer fmt.Print("> ")
		gs.HandlePause(state)
		return pubsub.Ack
	}
}

func handlerMove(gs *gamelogic.GameState, ch *amqp.Channel) func(gamelogic.ArmyMove) pubsub.AckType {
	return func(msg gamelogic.ArmyMove) pubsub.AckType {
		defer fmt.Print("> ")
		retvalue := gs.HandleMove(msg)
		if retvalue == gamelogic.MoveOutComeSafe {
			return pubsub.Ack
		} else if retvalue == gamelogic.MoveOutcomeMakeWar {
			if err := pubsub.PublishJSON(
				ch,
				routing.ExchangePerilTopic,
				routing.WarRecognitionsPrefix+"."+gs.Player.Username,
				gamelogic.RecognitionOfWar{
					Attacker: msg.Player,
					Defender: gs.GetPlayerSnap(),
				},
			); err != nil {
				fmt.Printf("error publishing war: %v\n", err)
				return pubsub.NackRequeue
			}
			return pubsub.Ack
		} else if retvalue == gamelogic.MoveOutcomeSamePlayer {
			return pubsub.NackDiscard
		} else {
			return pubsub.NackDiscard
		}
	}
}

func handlerWar(gs *gamelogic.GameState, ch *amqp.Channel, exchange string, msg gamelogic.RecognitionOfWar) pubsub.AckType {
	defer fmt.Print("> ")
	outcome, winner, loser := gs.HandleWar(msg)
	var message string
	switch outcome {
	case gamelogic.WarOutcomeNotInvolved:
		return pubsub.NackRequeue
	case gamelogic.WarOutcomeNoUnits:
		return pubsub.NackDiscard
	case gamelogic.WarOutcomeYouWon, gamelogic.WarOutcomeOpponentWon:
		message = fmt.Sprintf("%s won a war against %s", winner, loser)
	case gamelogic.WarOutcomeDraw:
		message = fmt.Sprintf("A war between %s and %s resulted in a draw", winner, loser)
	default:
		fmt.Println("error: unknown war outcome")
		return pubsub.NackDiscard
	}
	log := routing.GameLog{
		CurrentTime: time.Now(),
		Message:     message,
		Username:    msg.Attacker.Username,
	}
	key := routing.GameLogSlug + "." + msg.Attacker.Username
	if err := pubsub.PublishGob(ch, routing.ExchangePerilTopic, key, log); err != nil {
		return pubsub.NackRequeue
	}
	return pubsub.Ack
}
