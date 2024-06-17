package queue

type QueueSingle struct {
	NumWorkers int
	jobs       chan AnalyzeJob
	results    chan AnalyzeResult
}

func NewQueueSingle(numWorkers int) Queue {
	q := QueueSingle{
		NumWorkers: numWorkers,
		jobs:       make(chan AnalyzeJob, 10),
		results:    make(chan AnalyzeResult, 10),
	}
	return q
}

func (q QueueSingle) Jobs() chan AnalyzeJob {
	return q.jobs
}

func (q QueueSingle) Results() chan AnalyzeResult {
	return q.results
}

func (q QueueSingle) Close() {
	close(q.jobs)
	close(q.results)
}
