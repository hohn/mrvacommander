package agent

import (
	"github.com/hohn/mrvacommander/pkg/artifactstore"
	"github.com/hohn/mrvacommander/pkg/qldbstore"
	"github.com/hohn/mrvacommander/pkg/queue"
)

type Visibles struct {
	Queue         queue.Queue
	Artifacts     artifactstore.Store
	CodeQLDBStore qldbstore.Store
}
