package state

import (
	"fmt"
	"log/slog"
	"github.com/hohn/mrvacommander/pkg/common"
	"github.com/hohn/mrvacommander/pkg/queue"
	"sync"
)

type LocalState struct {
	jobs                 map[int][]queue.AnalyzeJob
	info                 map[common.JobSpec]common.JobInfo
	status               map[common.JobSpec]common.Status
	result               map[common.JobSpec]queue.AnalyzeResult
	sessionToJobIdToSpec map[int]map[int]common.JobSpec
	mutex                sync.Mutex
	currentID            int
}

func NewLocalState(startingID int) *LocalState {
	state := &LocalState{
		jobs:                 make(map[int][]queue.AnalyzeJob),
		info:                 make(map[common.JobSpec]common.JobInfo),
		status:               make(map[common.JobSpec]common.Status),
		result:               make(map[common.JobSpec]queue.AnalyzeResult),
		sessionToJobIdToSpec: make(map[int]map[int]common.JobSpec),
		currentID:            startingID,
	}
	state.sessionToJobIdToSpec[startingID] = make(map[int]common.JobSpec)
	state.jobs[startingID] = []queue.AnalyzeJob{}
	return state
}

func (s *LocalState) NextID() int {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.currentID++
	s.jobs[s.currentID] = []queue.AnalyzeJob{}
	return s.currentID
}

func (s *LocalState) GetResult(js common.JobSpec) (queue.AnalyzeResult, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if _, ok := s.result[js]; !ok {
		return queue.AnalyzeResult{}, fmt.Errorf("result not found for job spec %v", js)
	}
	return s.result[js], nil
}

func (s *LocalState) GetJobSpecByRepoId(sessionId, jobRepoId int) (common.JobSpec, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	idToSpec, ok := s.sessionToJobIdToSpec[sessionId]
	if !ok {
		return common.JobSpec{}, fmt.Errorf("job ids not found for session %v", sessionId)
	}
	spec, ok := idToSpec[jobRepoId]
	if !ok {
		return common.JobSpec{}, fmt.Errorf("job spec not found for job repo id %v", jobRepoId)
	}
	return spec, nil
}

func (s *LocalState) SetResult(js common.JobSpec, analyzeResult queue.AnalyzeResult) {
	s.mutex.Lock()
	s.result[js] = analyzeResult
	s.mutex.Unlock()
}

func (s *LocalState) GetJobList(sessionID int) ([]queue.AnalyzeJob, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if _, ok := s.jobs[sessionID]; !ok {
		return nil, fmt.Errorf("job list not found for session %v", sessionID)
	}
	return s.jobs[sessionID], nil
}

func (s *LocalState) GetJobInfo(js common.JobSpec) (common.JobInfo, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if _, ok := s.info[js]; !ok {
		return common.JobInfo{}, fmt.Errorf("job info not found for job spec %v", js)
	}
	return s.info[js], nil
}

func (s *LocalState) SetJobInfo(js common.JobSpec, ji common.JobInfo) {
	s.mutex.Lock()
	s.info[js] = ji
	s.mutex.Unlock()
}

func (s *LocalState) GetStatus(js common.JobSpec) (common.Status, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if _, ok := s.status[js]; !ok {
		return common.StatusError, fmt.Errorf("status not found for job spec %v", js)
	}
	return s.status[js], nil
}

func (s *LocalState) SetStatus(js common.JobSpec, status common.Status) {
	s.mutex.Lock()
	s.status[js] = status
	s.mutex.Unlock()
}

func (s *LocalState) AddJob(job queue.AnalyzeJob) {
	s.mutex.Lock()
	sessionID := job.Spec.SessionID
	s.jobs[sessionID] = append(s.jobs[sessionID], job)
	// Map the job index to JobSpec for quick result lookup
	if _, ok := s.sessionToJobIdToSpec[sessionID]; !ok {
		s.sessionToJobIdToSpec[sessionID] = make(map[int]common.JobSpec)
	}
	s.sessionToJobIdToSpec[sessionID][len(s.sessionToJobIdToSpec[sessionID])] = job.Spec
	if len(s.jobs[sessionID]) != len(s.sessionToJobIdToSpec[sessionID]) {
		msg := fmt.Sprintf("Unequal job list and job id map length. Session ID: %v", sessionID)
		slog.Error(msg)
		panic(msg)
	}
	s.mutex.Unlock()
}
