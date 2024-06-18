package queue

import (
	"mrvacommander/pkg/common"
)

type Queue interface {
	Jobs() chan common.AnalyzeJob
	Results() chan common.AnalyzeResult
}
