package server

import (
	"mrvacommander/pkg/common"
	"mrvacommander/pkg/logger"
	"mrvacommander/pkg/qpstore"
	"mrvacommander/pkg/queue"
	"mrvacommander/pkg/storage"
)

type SessionInfo struct {
	ID             int
	Owner          string
	ControllerRepo string

	QueryPack    string
	Language     string
	Repositories []common.NameWithOwner

	AccessMismatchRepos []common.NameWithOwner
	NotFoundRepos       []common.NameWithOwner
	NoCodeqlDBRepos     []common.NameWithOwner
	OverLimitRepos      []common.NameWithOwner

	AnalysisRepos *map[common.NameWithOwner]storage.DBLocation
}

type CommanderSingle struct {
	vis *Visibles
}

func NewCommanderSingle(st *Visibles) *CommanderSingle {
	c := CommanderSingle{}

	setupEndpoints(&c)

	return &c
}

// type State struct {
// 	Commander Commander
// 	Logger    logger.Logger
// 	Queue     queue.Queue
// 	Storage   storage.Storage
// 	Runner    agent.Runner
// }

type Visibles struct {
	Logger      logger.Logger
	Queue       queue.Queue
	ServerStore storage.Storage
	// TODO extra package for query pack storage
	QueryPackStore qpstore.Storage
	// TODO extra package for ql db storage
	QLDBStore storage.Storage
}
