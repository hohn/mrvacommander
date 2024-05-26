package agent

import (
	"bufio"
	"bytes"
	"log/slog"

	"mrvacommander/pkg/common"
	"mrvacommander/pkg/queue"
	"mrvacommander/pkg/storage"

	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
)

type RunnerSingle struct {
	queue queue.Queue
}

func NewRunnerSingle(numWorkers int, queue queue.Queue) *RunnerSingle {
	r := RunnerSingle{queue: queue}

	for id := 1; id <= numWorkers; id++ {
		go r.worker(id)
	}
	return &r
}

func (r *RunnerSingle) worker(wid int) {
	var job common.AnalyzeJob

	for {
		job = <-r.queue.Jobs()

		slog.Debug("Picking up job", "job", job, "worker", wid)

		cwd, err := os.Getwd()
		if err != nil {
			slog.Error("RunJob: cwd problem: ", "error", err)
			continue
		}

		slog.Debug("Analysis: running", "job", job)
		storage.SetStatus(job.QueryPackId, job.ORL, common.StatusQueued)
		cmd := exec.Command(path.Join(cwd, "bin", "run-analysis.sh"),
			strconv.FormatInt(int64(job.QueryPackId), 10),
			job.QueryLanguage, job.ORL.Owner, job.ORL.Repo)

		out, err := cmd.CombinedOutput()
		if err != nil {
			slog.Error("Analysis command failed: exit code: ", "error", err, "job", job)
			slog.Error("Analysis command failed: ", "job", job, "output", out)
			storage.SetStatus(job.QueryPackId, job.ORL, common.StatusError)
			continue
		}
		slog.Debug("Analysis run finished", "job", job)

		// Get the SARIF ouput location
		sr := bufio.NewScanner(bytes.NewReader(out))
		sr.Split(bufio.ScanLines)
		for {
			more := sr.Scan()
			if !more {
				slog.Error("Analysis run failed to report result: ", "output", out)
				break
			}
			fields := strings.Fields(sr.Text())
			if len(fields) >= 3 {
				if fields[0] == "run-analysis-output" {
					slog.Debug("Analysis run successful: ", "job", job, "location", fields[2])
					res := common.AnalyzeResult{
						RunAnalysisSARIF: fields[2], // Abs. path from run-analysis.sh
						RunAnalysisBQRS:  "",        // FIXME? see note in run-analysis.sh
					}
					r.queue.Results() <- res
					storage.SetStatus(job.QueryPackId, job.ORL, common.StatusSuccess)
					storage.SetResult(job.QueryPackId, job.ORL, res)
					break
				}
			}
		}
	}
}
