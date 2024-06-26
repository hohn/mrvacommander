package server

import (
	"mrvacommander/pkg/artifactstore"
	"mrvacommander/pkg/common"
	"mrvacommander/pkg/qldbstore"
	"mrvacommander/pkg/queue"
	"mrvacommander/pkg/state"
)

type SessionInfo struct {
	// TODO verify: these fields are never used
	// Owner               string
	// ControllerRepo      string
	// Repositories        []common.NameWithOwner
	// OverLimitRepos      []common.NameWithOwner
	// AnalysisRepos       *map[common.NameWithOwner]qldbstore.CodeQLDatabaseLocation

	ID        int
	QueryPack string
	Language  string

	AccessMismatchRepos []common.NameWithOwner
	NotFoundRepos       []common.NameWithOwner
	NoCodeqlDBRepos     []common.NameWithOwner
}

type CommanderSingle struct {
	v *Visibles
}

func NewCommanderSingle(st *Visibles) *CommanderSingle {
	c := CommanderSingle{v: st}
	setupEndpoints(&c)
	go c.ConsumeResults()
	return &c
}

type Visibles struct {
	Queue         queue.Queue
	State         state.ServerState
	Artifacts     artifactstore.Store
	CodeQLDBStore qldbstore.Store
}
