package queue

type Queue interface {
	Jobs() chan AnalyzeJob
	Results() chan AnalyzeResult
	Close()
}
