package server

import (
	"mrvacommander/pkg/agent"
	"mrvacommander/pkg/common"
	"mrvacommander/pkg/logger"
	"mrvacommander/pkg/queue"
	"mrvacommander/pkg/storage"
)

type SessionInfo struct {
	ID             int
	Owner          string
	ControllerRepo string

	QueryPack    string
	Language     string
	Repositories []common.OwnerRepo

	AccessMismatchRepos []common.OwnerRepo
	NotFoundRepos       []common.OwnerRepo
	NoCodeqlDBRepos     []common.OwnerRepo
	OverLimitRepos      []common.OwnerRepo

	AnalysisRepos *map[common.OwnerRepo]storage.DBLocation
}

type CommanderSingle struct {
	st    *State
	queue queue.Queue
}

func NewCommanderSingle(s *State, q queue.Queue) *CommanderSingle {
	c := CommanderSingle{s, q}
	return &c
}

type State struct {
	Commander Commander
	Logger    logger.Logger
	Queue     queue.Queue
	Storage   storage.Storage
	Runner    agent.Runner
}
