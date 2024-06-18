package state

import (
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"mrvacommander/pkg/common"
)

type LocalState struct {
	jobs      map[int][]common.AnalyzeJob
	info      map[common.JobSpec]common.JobInfo
	status    map[common.JobSpec]common.Status
	result    map[common.JobSpec]common.AnalyzeResult
	mutex     sync.Mutex
	currentID int
}

func NewLocalState(startingID int) *LocalState {
	return &LocalState{
		jobs:      make(map[int][]common.AnalyzeJob),
		info:      make(map[common.JobSpec]common.JobInfo),
		status:    make(map[common.JobSpec]common.Status),
		result:    make(map[common.JobSpec]common.AnalyzeResult),
		currentID: startingID,
	}
}

func (s *LocalState) NextID() int {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.currentID++
	return s.currentID
}

func (s *LocalState) GetArtifactURL(js common.JobSpec, vaid int) (string, error) {
	// TODO: have the server convert an artifact to a URL temporarily hosted on the
	return "", nil
}

func (s *LocalState) GetResult(js common.JobSpec) common.AnalyzeResult {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.result[js]
}

func (s *LocalState) SetResult(jobID int, nwo common.NameWithOwner, analyzeResult common.AnalyzeResult) {
	s.mutex.Lock()
	s.result[common.JobSpec{JobID: jobID, NameWithOwner: nwo}] = analyzeResult
	s.mutex.Unlock()
}

func (s *LocalState) GetJobList(jobID int) []common.AnalyzeJob {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.jobs[jobID]
}

func (s *LocalState) GetJobInfo(js common.JobSpec) common.JobInfo {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.info[js]
}

func (s *LocalState) SetJobInfo(js common.JobSpec, ji common.JobInfo) {
	s.mutex.Lock()
	s.info[js] = ji
	s.mutex.Unlock()
}

func (s *LocalState) GetStatus(jobID int, nwo common.NameWithOwner) common.Status {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.status[common.JobSpec{JobID: jobID, NameWithOwner: nwo}]
}

func (s *LocalState) SetStatus(jobID int, nwo common.NameWithOwner, status common.Status) {
	s.mutex.Lock()
	s.status[common.JobSpec{JobID: jobID, NameWithOwner: nwo}] = status
	s.mutex.Unlock()
}

func (s *LocalState) AddJob(jobID int, job common.AnalyzeJob) {
	s.mutex.Lock()
	s.jobs[jobID] = append(s.jobs[jobID], job)
	s.mutex.Unlock()
}

// TODO: @hohn
func ResultAsFile(path string) (string, []byte, error) {
	fpath := path
	if !filepath.IsAbs(path) {
		fpath = "/" + path
	}

	file, err := os.ReadFile(fpath)
	if err != nil {
		slog.Warn("Failed to read results file", fpath, err)
		return "", nil, err
	}

	return fpath, file, nil
}
