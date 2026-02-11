package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"golang.org/x/exp/slog"
)

type RabbitMQQueue struct {
	jobs    chan AnalyzeJob
	results chan AnalyzeResult
	conn    *amqp.Connection
	channel *amqp.Channel

	mu         sync.Mutex
	connString string
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

	// Enable publisher confirms so that publishJob/publishResult
	// confirmations actually arrive.  (reconnectIfNeeded already does
	// this, but the initial connection skips that path.)
	err = ch.Confirm(false)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to enable publisher confirms: %w", err)
	}

	result := RabbitMQQueue{
		conn:       conn,
		channel:    ch,
		jobs:       make(chan AnalyzeJob),
		results:    make(chan AnalyzeResult),
		mu:         sync.Mutex{},
		connString: rabbitMQURL,
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

func (q *RabbitMQQueue) reconnectIfNeeded() error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.conn != nil && !q.conn.IsClosed() && q.channel != nil {
		return nil // still valid
	}

	// Recreate everything
	conn, err := amqp.Dial(q.connString)
	if err != nil {
		return fmt.Errorf("failed to reconnect: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to open channel: %w", err)
	}

	// Enable publisher confirms on the channel once during setup
	err = ch.Confirm(false)
	if err != nil {
		ch.Close()
		conn.Close()
		return fmt.Errorf("failed to enable publisher confirms: %w", err)
	}

	// Optional: redeclare queues here
	// _, _ = ch.QueueDeclare(...)

	q.conn = conn
	q.channel = ch
	return nil
}

func (q *RabbitMQQueue) invalidateConnection() {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.channel != nil {
		_ = q.channel.Close()
	}
	if q.conn != nil {
		_ = q.conn.Close()
	}

	q.channel = nil
	q.conn = nil
}

func (q *RabbitMQQueue) ConsumeJobs(queueName string) {
	const pollInterval = 5 * time.Second

	// | scenario          | result                                |
	// |-------------------+---------------------------------------|
	// | Queue is empty    | msg = zero, ok = false, err = nil     |
	// | Queue has message | msg = valid, ok = true, err = nil     |
	// | Connection lost   | msg = zero, ok = false, err = non-nil |

	for {

		if err := q.reconnectIfNeeded(); err != nil {
			slog.Error("failed to reconnect", slog.Any("error", err))
			time.Sleep(10 * time.Second)
			continue
		}

		msg, ok, err := q.channel.Get(queueName, false) // false = manual ack
		if err != nil {
			slog.Error("polling error while getting job", slog.Any("error", err))
			q.invalidateConnection()
			time.Sleep(pollInterval)
			continue
		}

		if !ok {
			// No message in queue
			time.Sleep(pollInterval)
			continue
		}

		var job AnalyzeJob
		if err := json.Unmarshal(msg.Body, &job); err != nil {
			slog.Error("failed to unmarshal job", slog.Any("error", err))
			_ = msg.Nack(false, false) // do not requeue
			continue
		}

		// Send job to channel for processing
		q.jobs <- job

		// Acknowledge successful processing
		if err := msg.Ack(false); err != nil {
			slog.Error("failed to ack job message", slog.Any("error", err))
			continue
		}
	}
}

func (q *RabbitMQQueue) PublishResults(queueName string) {
	const pollInterval = 5 * time.Second

	for result := range q.results {
		for {
			if err := q.reconnectIfNeeded(); err != nil {
				slog.Error("failed to reconnect for publishing", slog.Any("error", err))
				time.Sleep(10 * time.Second)
				continue
			}

			if err := q.publishResult(queueName, result); err != nil {
				slog.Error("failed to publish result, will retry", slog.Any("error", err))
				q.invalidateConnection()
				time.Sleep(pollInterval)
				continue
			}

			// Successfully published, move to next result
			break
		}
	}
}

func (q *RabbitMQQueue) publishResult(queueName string, result AnalyzeResult) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	resultBytes, err := json.Marshal(result)
	if err != nil {
		slog.Error("publishResult: failed to marshal result", slog.Any("error", err))
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	// Check channel validity with mutex protection
	q.mu.Lock()
	ch := q.channel
	if ch == nil {
		q.mu.Unlock()
		slog.Error("publishResult: channel is nil")
		return fmt.Errorf("channel is nil")
	}
	q.mu.Unlock()

	// Set up a confirmation channel. This uses a large capacity to avoid blocking.
	confirmChannelSize := 99999
	confirmations := ch.NotifyPublish(make(chan amqp.Confirmation, confirmChannelSize))

	// Publish the message
	slog.Debug("Publishing result", slog.String("result", string(resultBytes)))
	err = ch.PublishWithContext(ctx, "", queueName, false, false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        resultBytes,
		})
	if err != nil {
		slog.Error("failed to publish result", slog.Any("error", err))
		return fmt.Errorf("failed to publish result: %w", err)
	}

	// Wait for the confirmation
	confirm := <-confirmations
	if !confirm.Ack {
		slog.Error("Publish result message confirmation failed")
		return fmt.Errorf("publish result message confirmation failed")
	}

	return nil
}

func (q *RabbitMQQueue) publishJob(queueName string, job AnalyzeJob) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	jobBytes, err := json.Marshal(job)
	if err != nil {
		slog.Error("failed to marshal job", slog.Any("error", err))
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	// Check channel validity with mutex protection
	q.mu.Lock()
	ch := q.channel
	if ch == nil {
		q.mu.Unlock()
		slog.Error("publishJob: channel is nil")
		return fmt.Errorf("channel is nil")
	}
	q.mu.Unlock()

	// Set up a confirmation channel. This uses a large capacity to avoid
	// blocking server requests.
	confirmChannelSize := 99999
	confirmations := ch.NotifyPublish(make(chan amqp.Confirmation, confirmChannelSize))

	// Publish the job
	slog.Debug("Publishing job", slog.String("job", string(jobBytes)))
	err = ch.PublishWithContext(ctx, "", queueName, false, false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        jobBytes,
		})
	if err != nil {
		slog.Error("failed to publish job", slog.Any("error", err))
		return fmt.Errorf("failed to publish job: %w", err)
	}

	// Wait for the confirmation
	confirm := <-confirmations
	if !confirm.Ack {
		slog.Error("Publish job message confirmation failed")
		return fmt.Errorf("publish job message confirmation failed")
	}

	return nil
}

func (q *RabbitMQQueue) PublishJobs(queueName string) {
	const pollInterval = 5 * time.Second

	for job := range q.jobs {
		for {
			if err := q.reconnectIfNeeded(); err != nil {
				slog.Error("failed to reconnect for publishing", slog.Any("error", err))
				time.Sleep(10 * time.Second)
				continue
			}

			if err := q.publishJob(queueName, job); err != nil {
				slog.Error("failed to publish job, will retry", slog.Any("error", err))
				q.invalidateConnection()
				time.Sleep(pollInterval)
				continue
			}

			// Successfully published, move to next job
			break
		}
	}
}

func (q *RabbitMQQueue) ConsumeResults(queueName string) {
	autoAck := false // false = manual ack
	sleepFor := 5    // polling interval

	for {

		if err := q.reconnectIfNeeded(); err != nil {
			slog.Error("failed to reconnect", slog.Any("error", err))
			time.Sleep(10 * time.Second)
			continue
		}

		msg, ok, err := q.channel.Get(queueName, autoAck)
		if err != nil {
			slog.Error("poll error", slog.Any("err", err))
			time.Sleep(time.Duration(sleepFor) * time.Second)
			continue
		}
		if !ok {
			// no message
			time.Sleep(time.Duration(sleepFor) * time.Second)
			continue
		}

		var result AnalyzeResult
		if err := json.Unmarshal(msg.Body, &result); err != nil {
			slog.Error("unmarshal error", slog.Any("err", err))
			_ = msg.Nack(false, false) // finish .Get() with nack
			continue
		}

		q.results <- result
		_ = msg.Ack(false) // finish .Get() with nack
	}

}
