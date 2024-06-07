package queue

import (
	"mrvacommander/pkg/common"
	"mrvacommander/pkg/logger"
)

type QueueSingle struct {
	NumWorkers int
	jobs       chan common.AnalyzeJob
	results    chan common.AnalyzeResult
	modules    *QueueVisibles
}

type QueueVisibles struct {
	Logger logger.Logger
}

func (q *QueueSingle) Setup(v *QueueVisibles) {
	q.modules = v
}

func NewQueueSingle(numWorkers int) *QueueSingle {
	q := QueueSingle{}
	q.jobs = make(chan common.AnalyzeJob, 10)
	q.results = make(chan common.AnalyzeResult, 10)
	q.NumWorkers = numWorkers
	return &q
}
