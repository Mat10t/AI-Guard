package events

import (
	"context"
	"encoding/json"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
)

type Publisher struct {
	brokers []string
}

func NewPublisher(brokers string) *Publisher {
	parts := strings.Split(brokers, ",")
	clean := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			clean = append(clean, p)
		}
	}
	return &Publisher{brokers: clean}
}

func (p *Publisher) Publish(ctx context.Context, topic, key string, payload any) error {
	if len(p.brokers) == 0 {
		return nil
	}
	val, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	writeOnce := func(writeCtx context.Context) error {
		w := &kafka.Writer{
			Addr:                   kafka.TCP(p.brokers...),
			Topic:                  topic,
			Balancer:               &kafka.Hash{},
			RequiredAcks:           kafka.RequireOne,
			AllowAutoTopicCreation: true,
			Async:                  false,
		}
		defer w.Close()
		return w.WriteMessages(writeCtx, kafka.Message{
			Key:   []byte(key),
			Value: val,
			Time:  time.Now(),
		})
	}

	writeCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err = writeOnce(writeCtx); err == nil {
		return nil
	}
	if !isTopicProvisioningError(err) {
		return err
	}

	ensureCtx, ensureCancel := context.WithTimeout(ctx, 3*time.Second)
	defer ensureCancel()
	if ensureErr := p.ensureTopic(ensureCtx, topic); ensureErr != nil {
		return err
	}

	retryCtx, retryCancel := context.WithTimeout(ctx, 3*time.Second)
	defer retryCancel()
	return writeOnce(retryCtx)
}

func (p *Publisher) ensureTopic(ctx context.Context, topic string) error {
	if len(p.brokers) == 0 {
		return nil
	}
	conn, err := kafka.DialContext(ctx, "tcp", p.brokers[0])
	if err != nil {
		return err
	}
	defer conn.Close()

	controller, err := conn.Controller()
	if err != nil {
		return err
	}

	controllerAddr := net.JoinHostPort(controller.Host, strconv.Itoa(controller.Port))
	controllerConn, err := kafka.DialContext(ctx, "tcp", controllerAddr)
	if err != nil {
		return err
	}
	defer controllerConn.Close()

	err = controllerConn.CreateTopics(kafka.TopicConfig{
		Topic:             topic,
		NumPartitions:     1,
		ReplicationFactor: 1,
	})
	if err != nil {
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "already exists") || strings.Contains(msg, "topic with this name already exists") {
			return nil
		}
		return err
	}
	return nil
}

func isTopicProvisioningError(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unknown topic") ||
		strings.Contains(msg, "unknown topic or partition") ||
		strings.Contains(msg, "leader not available")
}
