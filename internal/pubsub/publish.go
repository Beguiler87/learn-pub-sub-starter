package pubsub

import (
	"context"
	"encoding/json"

	amqp "github.com/rabbitmq/amqp091-go"
)

func PublishJSON[T any](ch *amqp.Channel, exchange, key string, val T) error {
	jsonData, err := json.Marshal(val)
	if err != nil {
		return err
	}
	return ch.PublishWithContext(
		context.Background(),
		exchange,
		key,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        jsonData,
		},
	)
}

type SimpleQueueType int

const (
	Durable SimpleQueueType = iota
	Transient
)

func DeclareAndBind(conn *amqp.Connection, exchange, queueName, key string, queueType SimpleQueueType) (*amqp.Channel, amqp.Queue, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, amqp.Queue{}, err
	}
	durable := (queueType == Durable)
	autoDelete := (queueType == Transient)
	exclusive := (queueType == Transient)
	q, err := ch.QueueDeclare(
		queueName,
		durable,
		autoDelete,
		exclusive,
		false,
		nil)
	if err != nil {
		ch.Close()
		return nil, amqp.Queue{}, err
	}
	if err := ch.QueueBind(queueName, key, exchange, false, nil); err != nil {
		ch.Close()
		return nil, amqp.Queue{}, err
	}
	return ch, q, nil
}
