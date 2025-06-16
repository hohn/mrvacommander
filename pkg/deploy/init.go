package deploy

import (
	"fmt"
	"log"
	"log/slog"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/hohn/mrvacommander/pkg/artifactstore"
	"github.com/hohn/mrvacommander/pkg/qldbstore"
	"github.com/hohn/mrvacommander/pkg/queue"
	"github.com/minio/minio-go/v7"
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
		"MRVA_MINIO_VIRTUAL_HOST",
	}
	validateEnvVars(requiredEnvVars)

	endpoint := os.Getenv("ARTIFACT_MINIO_ENDPOINT")
	id := os.Getenv("ARTIFACT_MINIO_ID")
	secret := os.Getenv("ARTIFACT_MINIO_SECRET")
	useVirtual := os.Getenv("MRVA_MINIO_VIRTUAL_HOST") == "1"

	var lookup minio.BucketLookupType
	var bucketName string

	if useVirtual {
		parsedURL, err := url.Parse(endpoint)
		if err != nil {
			return nil, fmt.Errorf("failed to parse ARTIFACT_MINIO_ENDPOINT: %w", err)
		}
		hostParts := strings.Split(parsedURL.Hostname(), ".")
		if len(hostParts) < 2 {
			return nil, fmt.Errorf("unable to extract bucket from host: %s", parsedURL.Hostname())
		}
		bucketName = hostParts[0]
		lookup = minio.BucketLookupDNS
	} else {
		bucketName = "mrvabucket"
		lookup = minio.BucketLookupPath
	}
	// TODO: unify into one. clean up state handling.
	artifactstore.AF_BUCKETNAME_RESULTS = bucketName
	artifactstore.AF_BUCKETNAME_PACKS = bucketName

	store, err := artifactstore.NewMinIOArtifactStore(endpoint, id, secret, lookup)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize artifact store: %v", err)
	}
	return store, nil

}
func InitHEPCDatabaseStore() (qldbstore.Store, error) {
	requiredEnvVars := []string{
		"MRVA_HEPC_ENDPOINT",
		"MRVA_HEPC_CACHE_DURATION",
		"MRVA_HEPC_DATAVIACLI",
		"MRVA_HEPC_OUTDIR",
		"MRVA_HEPC_TOOL",
	}
	validateEnvVars(requiredEnvVars)

	endpoint := os.Getenv("MRVA_HEPC_ENDPOINT")

	store := qldbstore.NewHepcStore(endpoint)

	return store, nil
}
