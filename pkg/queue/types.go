package queue

import "mrvacommander/pkg/common"

type QueueSingle struct {
	NumWorkers int
	jobs       chan common.AnalyzeJob
	results    chan common.AnalyzeResult
}

func NewQueueSingle(numWorkers int) *QueueSingle {
	q := QueueSingle{}
	q.jobs = make(chan common.AnalyzeJob, 10)
	q.results = make(chan common.AnalyzeResult, 10)
	q.NumWorkers = numWorkers
	return &q
}
