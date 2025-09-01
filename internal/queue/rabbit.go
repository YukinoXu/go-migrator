package queue

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Client interface {
	Publish(ctx context.Context, id string) error
	Consume(ctx context.Context) (<-chan string, error)
	Close() error
}

type rabbitClient struct {
	conn *amqp.Connection
	q    amqp.Queue
}

// NewRabbitClient connects to RabbitMQ and declares a queue with the given name.
func NewRabbitClient(url string, queueName string) (Client, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, err
	}
	q, err := ch.QueueDeclare(
		queueName,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, err
	}
	// close channel; we'll open new channels for publish/consume
	ch.Close()
	return &rabbitClient{conn: conn, q: q}, nil
}

func (r *rabbitClient) Publish(ctx context.Context, id string) error {
	ch, err := r.conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()
	err = ch.PublishWithContext(ctx,
		"", r.q.Name, false, false,
		amqp.Publishing{ContentType: "text/plain", Body: []byte(id)},
	)
	return err
}

func (r *rabbitClient) Consume(ctx context.Context) (<-chan string, error) {
	ch, err := r.conn.Channel()
	if err != nil {
		return nil, err
	}
	msgs, err := ch.Consume(r.q.Name, "", false, false, false, false, nil)
	if err != nil {
		ch.Close()
		return nil, err
	}
	out := make(chan string)
	go func() {
		defer ch.Close()
		defer close(out)
		for d := range msgs {
			select {
			case out <- string(d.Body):
				// ack
				d.Ack(false)
			case <-ctx.Done():
				d.Nack(false, true)
				return
			}
		}
	}()
	// return channel of ids
	return out, nil
}

func (r *rabbitClient) Close() error {
	if r.conn == nil {
		return nil
	}
	return r.conn.Close()
}
