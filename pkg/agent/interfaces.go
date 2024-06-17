package agent

import (
	"mrvacommander/pkg/artifactstore"
	"mrvacommander/pkg/qldbstore"
	"mrvacommander/pkg/queue"
)

type Visibles struct {
	Queue         queue.Queue
	Artifacts     artifactstore.Store
	CodeQLDBStore qldbstore.Store
}
