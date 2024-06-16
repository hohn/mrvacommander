package queue

import (
	"mrvacommander/pkg/common"
	"mrvacommander/pkg/logger"
)

type QueueSingle struct {
	NumWorkers int
	jobs       chan common.AnalyzeJob
	results    chan common.AnalyzeResult
	modules    *Visibles
}

type Visibles struct {
	Logger logger.Logger
}

func NewQueueSingle(numWorkers int, v *Visibles) *QueueSingle {
	q := QueueSingle{}
	q.jobs = make(chan common.AnalyzeJob, 10)
	q.results = make(chan common.AnalyzeResult, 10)
	q.NumWorkers = numWorkers

	q.modules = v

	return &q
}
