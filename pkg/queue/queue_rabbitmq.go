package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"golang.org/x/exp/slog"
)

type RabbitMQQueue struct {
	jobs    chan AnalyzeJob
	results chan AnalyzeResult
	conn    *amqp.Connection
	channel *amqp.Channel
}

// NewRabbitMQQueue initializes a RabbitMQ queue.
// It returns a pointer to a RabbitMQQueue and an error.
//
// If isAgent is true, the queue is initialized to be used by an agent.
// Otherwise, the queue is initialized to be used by the server.
// The difference in behaviour is that the agent consumes jobs and publishes results,
// while the server publishes jobs and consumes results.
func NewRabbitMQQueue(
	host string,
	port int16,
	user string,
	password string,
	isAgent bool,
) (*RabbitMQQueue, error) {
	const (
		tryCount      = 5
		retryDelaySec = 3
		// XX: static typing?
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
		jobs:    make(chan AnalyzeJob),
		results: make(chan AnalyzeResult),
	}

	if isAgent {
		slog.Info("Starting tasks consumer")
		go result.ConsumeJobs(jobsQueueName)

		slog.Info("Starting results publisher")
		go result.PublishResults(resultsQueueName)
	} else {
		slog.Info("Starting jobs publisher")
		go result.PublishJobs(jobsQueueName)

		slog.Info("Starting results consumer")
		go result.ConsumeResults(resultsQueueName)
	}

	return &result, nil
}

func (q *RabbitMQQueue) Jobs() chan AnalyzeJob {
	return q.jobs
}

func (q *RabbitMQQueue) Results() chan AnalyzeResult {
	return q.results
}

func (q *RabbitMQQueue) Close() {
	q.channel.Close()
	q.conn.Close()
}

func (q *RabbitMQQueue) ConsumeJobs(queueName string) {
	autoAck := false
	msgs, err := q.channel.Consume(queueName, "", autoAck, false, false, false, nil)

	if err != nil {
		slog.Error("failed to consume from queue", slog.Any("error", err))
	}

	for msg := range msgs {
		// Process message
		job := AnalyzeJob{}
		err := json.Unmarshal(msg.Body, &job)
		if err != nil {
			slog.Error("failed to unmarshal job", slog.Any("error", err))
			continue
		}
		q.jobs <- job

		// Acknowledge the message after successful processing
		err = msg.Ack(false)
		if err != nil {
			slog.Error("Failed to acknowledge job consumption message",
				slog.Any("error", err))
			continue
		}

	}
	close(q.jobs)
}

func (q *RabbitMQQueue) PublishResults(queueName string) {
	for result := range q.results {
		q.publishResult(queueName, result)
	}
}

func (q *RabbitMQQueue) publishResult(queueName string, result AnalyzeResult) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	resultBytes, err := json.Marshal(result)
	if err != nil {
		slog.Error("failed to marshal result", slog.Any("error", err))
		return
	}

	// Enable publisher confirms on the channel
	err = q.channel.Confirm(false)
	if err != nil {
		slog.Error("Failed to enable publisher confirms", slog.Any("error", err))
	}

	// Set up a confirmation channel.  This uses a large capacity to avoid blocking.
	confirmChannelSize := 99999
	confirmations := q.channel.NotifyPublish(make(chan amqp.Confirmation, confirmChannelSize))

	// Publish the message
	slog.Debug("Publishing result", slog.String("result", string(resultBytes)))
	err = q.channel.PublishWithContext(ctx, "", queueName, false, false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        resultBytes,
		})
	if err != nil {
		slog.Error("failed to publish result", slog.Any("error", err))
	}

	// Wait for the confirmation
	confirm := <-confirmations
	if !confirm.Ack {
		slog.Error("Publish result message confirmation failed")
	}

}

func (q *RabbitMQQueue) publishJob(queueName string, job AnalyzeJob) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	jobBytes, err := json.Marshal(job)
	if err != nil {
		slog.Error("failed to marshal job", slog.Any("error", err))
		return
	}

	// Enable publisher confirms on the channel
	err = q.channel.Confirm(false)
	if err != nil {
		slog.Error("Failed to enable publisher confirms", slog.Any("error", err))
	}

	// Set up a confirmation channel.  This uses a large capacity to avoid
	// blocking server requests.
	confirmChannelSize := 99999
	confirmations := q.channel.NotifyPublish(make(chan amqp.Confirmation, confirmChannelSize))

	// Publish the job
	slog.Debug("Publishing job", slog.String("job", string(jobBytes)))
	err = q.channel.PublishWithContext(ctx, "", queueName, false, false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        jobBytes,
		})
	if err != nil {
		slog.Error("failed to publish job", slog.Any("error", err))
	}

	// Wait for the confirmation
	confirm := <-confirmations
	if !confirm.Ack {
		slog.Error("Publish result message confirmation failed")
	}

}

func (q *RabbitMQQueue) PublishJobs(queueName string) {
	for job := range q.jobs {
		q.publishJob(queueName, job)
	}
}

func (q *RabbitMQQueue) ConsumeResults(queueName string) {
	autoAck := false
	msgs, err := q.channel.Consume(queueName, "", autoAck, false, false, false, nil)
	if err != nil {
		slog.Error("failed to register a consumer", slog.Any("error", err))
	}

	for msg := range msgs {
		// Process message
		result := AnalyzeResult{}
		err := json.Unmarshal(msg.Body, &result)
		if err != nil {
			slog.Error("failed to unmarshal result", slog.Any("error", err))
			continue
		}
		q.results <- result

		// Acknowledge the message after successful processing
		err = msg.Ack(false)
		if err != nil {
			slog.Error("Failed to acknowledge result consumption message",
				slog.Any("error", err))
			continue
		}

	}
	close(q.results)
}
