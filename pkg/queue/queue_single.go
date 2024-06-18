package queue

import (
	"mrvacommander/pkg/common"
)

type QueueSingle struct {
	NumWorkers int
	jobs       chan common.AnalyzeJob
	results    chan common.AnalyzeResult
}

func NewQueueSingle(numWorkers int) *QueueSingle {
	q := QueueSingle{
		NumWorkers: numWorkers,
		jobs:       make(chan common.AnalyzeJob, 10),
		results:    make(chan common.AnalyzeResult, 10),
	}
	return &q
}

func (q *QueueSingle) Jobs() chan common.AnalyzeJob {
	return q.jobs
}

func (q *QueueSingle) Results() chan common.AnalyzeResult {
	return q.results
}
