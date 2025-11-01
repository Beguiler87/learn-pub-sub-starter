package pubsub

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type AckType string

const (
	Ack         AckType = "acknowledge"
	NackRequeue AckType = "negative_acknowledge_requeue"
	NackDiscard AckType = "negative_acknowledge_discard"
)

type SimpleQueueType int

const (
	SimpleQueueDurable SimpleQueueType = iota
	SimpleQueueTransient
)

func DeclareAndBind(
	conn *amqp.Connection,
	exchange,
	queueName,
	key string,
	queueType SimpleQueueType,
) (*amqp.Channel, amqp.Queue, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, amqp.Queue{}, fmt.Errorf("could not create channel: %v", err)
	}
	args := amqp.Table{}
	args["x-dead-letter-exchange"] = "peril_dlx"
	queue, err := ch.QueueDeclare(
		queueName,
		queueType == SimpleQueueDurable,
		queueType != SimpleQueueDurable,
		queueType != SimpleQueueDurable,
		false,
		args,
	)
	if err != nil {
		return nil, amqp.Queue{}, fmt.Errorf("could not declare queue: %v", err)
	}

	if err := ch.QueueBind(queue.Name, key, exchange, false, nil); err != nil {
		return nil, amqp.Queue{}, fmt.Errorf("could not bind queue: %v", err)
	}
	return ch, queue, nil
}

func SubscribeJSON[T any](conn *amqp.Connection, exchange, queueName, key string, queueType SimpleQueueType, handler func(T) AckType) error {
	return subscribe[T](
		conn, exchange, queueName, key, queueType,
		handler,
		"application/json",
		func(b []byte) (T, error) {
			var v T
			err := json.Unmarshal(b, &v)
			return v, err
		},
	)
}

func SubscribeGob[T any](conn *amqp.Connection, exchange, queueName, key string, queueType SimpleQueueType, handler func(T) AckType) error {
	return subscribe[T](
		conn, exchange, queueName, key, queueType,
		handler,
		"application/gob",
		func(b []byte) (T, error) {
			var v T
			err := gob.NewDecoder(bytes.NewReader(b)).Decode(&v)
			return v, err
		},
	)
}

func subscribe[T any](conn *amqp.Connection, exchange, queueName, key string, queueType SimpleQueueType, handler func(T) AckType, expectedContentType string, unmarshal func([]byte) (T, error)) error {
	ch, queue, err := DeclareAndBind(conn, exchange, queueName, key, queueType)
	if err != nil {
		return fmt.Errorf("could not declare and bind queue: %v", err)
	}
	if err := ch.Qos(10, 0, false); err != nil {
		return fmt.Errorf("set qos prefetch: %w", err)
	}
	deliveries, err := ch.Consume(
		queue.Name,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("could not consume messages: %v", err)
	}
	go func() {
		defer ch.Close()
		for d := range deliveries {
			if d.ContentType != expectedContentType {
				_ = d.Ack(false)
				continue
			}
			v, err := unmarshal(d.Body)
			if err != nil {
				_ = d.Ack(false)
				continue
			}
			switch handler(v) {
			case Ack:
				_ = d.Ack(false)
			case NackRequeue:
				_ = d.Nack(false, true)
			case NackDiscard:
				_ = d.Nack(false, false)
			}
		}
	}()
	return nil
}
