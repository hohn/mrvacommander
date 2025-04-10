package deploy

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"strconv"

	"github.com/hohn/mrvacommander/pkg/artifactstore"
	"github.com/hohn/mrvacommander/pkg/qldbstore"
	"github.com/hohn/mrvacommander/pkg/queue"
)

func validateEnvVars(requiredEnvVars []string) {
	missing := false

	for _, envVar := range requiredEnvVars {
		if _, ok := os.LookupEnv(envVar); !ok {
			slog.Error("Missing required environment variable", "key", envVar)
			missing = true
		}
	}

	if missing {
		os.Exit(1)
	}
}

func InitRabbitMQ(isAgent bool) (queue.Queue, error) {
	requiredEnvVars := []string{
		"MRVA_RABBITMQ_HOST",
		"MRVA_RABBITMQ_PORT",
		"MRVA_RABBITMQ_USER",
		"MRVA_RABBITMQ_PASSWORD",
	}
	validateEnvVars(requiredEnvVars)

	rmqHost := os.Getenv("MRVA_RABBITMQ_HOST")
	rmqPort := os.Getenv("MRVA_RABBITMQ_PORT")
	rmqUser := os.Getenv("MRVA_RABBITMQ_USER")
	rmqPass := os.Getenv("MRVA_RABBITMQ_PASSWORD")

	rmqPortAsInt, err := strconv.ParseInt(rmqPort, 10, 16)
	if err != nil {
		return nil, fmt.Errorf("failed to parse RabbitMQ port: %v", err)
	}

	log.Println("Initializing RabbitMQ queue")

	rabbitMQQueue, err := queue.NewRabbitMQQueue(rmqHost, int16(rmqPortAsInt), rmqUser, rmqPass, isAgent)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize RabbitMQ: %v", err)
	}

	return rabbitMQQueue, nil
}

func InitMinIOArtifactStore() (artifactstore.Store, error) {
	requiredEnvVars := []string{
		"ARTIFACT_MINIO_ENDPOINT",
		"ARTIFACT_MINIO_ID",
		"ARTIFACT_MINIO_SECRET",
	}
	validateEnvVars(requiredEnvVars)

	endpoint := os.Getenv("ARTIFACT_MINIO_ENDPOINT")
	id := os.Getenv("ARTIFACT_MINIO_ID")
	secret := os.Getenv("ARTIFACT_MINIO_SECRET")

	store, err := artifactstore.NewMinIOArtifactStore(endpoint, id, secret)

	if err != nil {
		return nil, fmt.Errorf("failed to initialize artifact store: %v", err)
	}

	return store, nil
}

func InitMinIOCodeQLDatabaseStore() (qldbstore.Store, error) {
	requiredEnvVars := []string{
		"QLDB_MINIO_ENDPOINT",
		"QLDB_MINIO_ID",
		"QLDB_MINIO_SECRET",
	}
	validateEnvVars(requiredEnvVars)

	endpoint := os.Getenv("QLDB_MINIO_ENDPOINT")
	id := os.Getenv("QLDB_MINIO_ID")
	secret := os.Getenv("QLDB_MINIO_SECRET")

	store, err := qldbstore.NewMinIOCodeQLDatabaseStore(endpoint, id, secret)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize ql database storage: %v", err)
	}

	return store, nil
}

func InitHEPCDatabaseStore() (qldbstore.Store, error) {
	requiredEnvVars := []string{
		"MRVA_HEPC_ENDPOINT",
		"MRVA_HEPC_CACHE_DURATION",
		"MRVA_HEPC_DATAVIACLI",
		"MRVA_HEPC_REFROOT",
		"MRVA_HEPC_OUTDIR",
		"MRVA_HEPC_TOOL",
	}
	validateEnvVars(requiredEnvVars)

	endpoint := os.Getenv("MRVA_HEPC_ENDPOINT")

	store := qldbstore.NewHepcStore(endpoint)

	return store, nil
}
