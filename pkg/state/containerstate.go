package state

import (
	"sync"

	"mrvacommander/pkg/common"
)

type ContainerState struct {
	jobs      map[int][]common.AnalyzeJob
	info      map[common.JobSpec]common.JobInfo
	status    map[common.JobSpec]common.Status
	result    map[common.JobSpec]common.AnalyzeResult
	mutex     sync.Mutex
	currentID int
}

func NewContainerState(startingID int) *ContainerState {
	return &ContainerState{
		jobs:      make(map[int][]common.AnalyzeJob),
		info:      make(map[common.JobSpec]common.JobInfo),
		status:    make(map[common.JobSpec]common.Status),
		result:    make(map[common.JobSpec]common.AnalyzeResult),
		currentID: startingID,
	}
}

func (s *ContainerState) NextID() int {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.currentID++
	return s.currentID
}

func (s *ContainerState) GetArtifactURL(js common.JobSpec, vaid int) (string, error) {
	// TODO: have the server convert an artifact to a URL temporarily hosted on the
	return "", nil
}

func (s *ContainerState) GetResult(js common.JobSpec) common.AnalyzeResult {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.result[js]
}

func (s *ContainerState) SetResult(jobID int, nwo common.NameWithOwner, analyzeResult common.AnalyzeResult) {
	s.mutex.Lock()
	s.result[common.JobSpec{JobID: jobID, NameWithOwner: nwo}] = analyzeResult
	s.mutex.Unlock()
}

func (s *ContainerState) GetJobList(jobID int) []common.AnalyzeJob {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.jobs[jobID]
}

func (s *ContainerState) GetJobInfo(js common.JobSpec) common.JobInfo {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.info[js]
}

func (s *ContainerState) SetJobInfo(js common.JobSpec, ji common.JobInfo) {
	s.mutex.Lock()
	s.info[js] = ji
	s.mutex.Unlock()
}

func (s *ContainerState) GetStatus(jobID int, nwo common.NameWithOwner) common.Status {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.status[common.JobSpec{JobID: jobID, NameWithOwner: nwo}]
}

func (s *ContainerState) SetStatus(jobID int, nwo common.NameWithOwner, status common.Status) {
	s.mutex.Lock()
	s.status[common.JobSpec{JobID: jobID, NameWithOwner: nwo}] = status
	s.mutex.Unlock()
}

func (s *ContainerState) AddJob(jobID int, job common.AnalyzeJob) {
	s.mutex.Lock()
	s.jobs[jobID] = append(s.jobs[jobID], job)
	s.mutex.Unlock()
}
