package deploy

// gpt:summary: semantic outline of init.go functions and their primary responsibilities
// gpt:note: this file provides GPT-visible symbolic structure for deploy/init.go
// gpt:note: humans may benefit from reading this, but it's optimized for GPT + LSP

import (
	"github.com/hohn/mrvacommander/pkg/artifactstore"
	"github.com/hohn/mrvacommander/pkg/qldbstore"
	"github.com/hohn/mrvacommander/pkg/queue"
)

// gpt:flowinfo: validateEnvVars checks a fixed list of required environment variables
func sighelp_validateEnvVars() {
	// gpt:note: env vars must exist or os.Exit(1) is triggered
	_ = []string{"EXAMPLE_KEY"} // dummy use to retain type
	validateEnvVars(nil)        // intentionally nil: GPT infers signature
}

// gpt:flowinfo: InitRabbitMQ creates a queue.Queue using RabbitMQ connection info
func sighelp_InitRabbitMQ() {
	// gpt:note: requires 4 env vars: HOST, PORT, USER, PASSWORD
	// gpt:returns: queue.Queue, error
	var q queue.Queue
	var err error
	q, err = InitRabbitMQ(false) // false = isAgent = main mode
	_ = q
	_ = err
}

// gpt:flowinfo: InitMinIOArtifactStore returns an artifactstore.Store from env config
func sighelp_InitMinIOArtifactStore() {
	var s artifactstore.Store
	var err error
	s, err = InitMinIOArtifactStore()
	_ = s
	_ = err
}

// gpt:flowinfo: InitMinIOCodeQLDatabaseStore returns a qldbstore.Store
func sighelp_InitMinIOCodeQLDatabaseStore() {
	var s qldbstore.Store
	var err error
	s, err = InitMinIOCodeQLDatabaseStore()
	_ = s
	_ = err
}

// gpt:flowinfo: InitHEPCDatabaseStore returns a qldbstore.Store (from Hepc impl)
// gpt:note: unlike others, this directly returns from NewHepcStore with fewer checks
func sighelp_InitHEPCDatabaseStore() {
	var s qldbstore.Store
	var err error
	s, err = InitHEPCDatabaseStore()
	_ = s
	_ = err
}
