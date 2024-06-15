package queue

import (
	"mrvacommander/pkg/common"
	"mrvacommander/pkg/storage"

	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"golang.org/x/exp/slog"
)

type RabbitMQQueue struct {
	jobs    chan common.AnalyzeJob
	results chan common.AnalyzeResult
	conn    *amqp.Connection
	channel *amqp.Channel
}

func InitializeRabbitMQQueue(
	host string,
	port int16,
	user string,
	password string,
) (*RabbitMQQueue, error) {
	const (
		tryCount         = 5
		retryDelaySec    = 3
		jobsQueueName    = "tasks"
		resultsQueueName = "results"
	)

	var conn *amqp.Connection
	var err error

	rabbitMQURL := fmt.Sprintf("amqp://%s:%s@%s:%d/", user, password, host, port)

	for i := 0; i < tryCount; i++ {
		slog.Info("Attempting to connect to RabbitMQ", slog.Int("attempt", i+1))
		conn, err = amqp.Dial(rabbitMQURL)
		if err != nil {
			slog.Warn("Failed to connect to RabbitMQ", "error", err)
			if i < tryCount-1 {
				slog.Info("Retrying", "seconds", retryDelaySec)
				time.Sleep(retryDelaySec * time.Second)
			}
		} else {
			// successfully connected to RabbitMQ
			break
		}
	}
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	slog.Info("Connected to RabbitMQ")

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open a channel: %w", err)
	}

	_, err = ch.QueueDeclare(jobsQueueName, false, false, false, true, nil)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to declare tasks queue: %w", err)
	}

	_, err = ch.QueueDeclare(resultsQueueName, false, false, false, true, nil)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to declare results queue: %w", err)
	}

	err = ch.Qos(1, 0, false)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to set QoS: %w", err)
	}

	result := RabbitMQQueue{
		conn:    conn,
		channel: ch,
		jobs:    make(chan common.AnalyzeJob),
		results: make(chan common.AnalyzeResult),
	}

	slog.Info("Starting tasks consumer")
	go result.ConsumeJobs(jobsQueueName)

	slog.Info("Starting results publisher")
	go result.PublishResults(resultsQueueName)

	return &result, nil
}

func (q *RabbitMQQueue) Jobs() chan common.AnalyzeJob {
	return q.jobs
}

func (q *RabbitMQQueue) Results() chan common.AnalyzeResult {
	return q.results
}

func (q *RabbitMQQueue) StartAnalyses(analysis_repos *map[common.NameWithOwner]storage.DBLocation, session_id int, session_language string) {
	// TODO: Implement
	log.Fatal("unimplemented")
}

func (q *RabbitMQQueue) Close() {
	q.channel.Close()
	q.conn.Close()
}

func (q *RabbitMQQueue) ConsumeJobs(queueName string) {
	msgs, err := q.channel.Consume(queueName, "", true, false, false, false, nil)
	if err != nil {
		slog.Error("failed to register a consumer", slog.Any("error", err))
	}

	for msg := range msgs {
		job := common.AnalyzeJob{}
		err := json.Unmarshal(msg.Body, &job)
		if err != nil {
			slog.Error("failed to unmarshal job", slog.Any("error", err))
			continue
		}
		q.jobs <- job
	}
	close(q.jobs)
}

func (q *RabbitMQQueue) PublishResults(queueName string) {
	for result := range q.results {
		q.publishResult(queueName, result)
	}
}

func (q *RabbitMQQueue) publishResult(queueName string, result interface{}) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	resultBytes, err := json.Marshal(result)
	if err != nil {
		slog.Error("failed to marshal result", slog.Any("error", err))
		return
	}

	slog.Info("Publishing result", slog.String("result", string(resultBytes)))
	err = q.channel.PublishWithContext(ctx, "", queueName, false, false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        resultBytes,
		})
	if err != nil {
		slog.Error("failed to publish result", slog.Any("error", err))
	}
}
