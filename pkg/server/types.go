package server

import (
	"mrvacommander/pkg/artifactstore"
	"mrvacommander/pkg/common"
	"mrvacommander/pkg/qldbstore"
	"mrvacommander/pkg/queue"
	"mrvacommander/pkg/state"
)

type SessionInfo struct {
	ID                  int
	Owner               string
	ControllerRepo      string
	QueryPack           string
	Language            string
	Repositories        []common.NameWithOwner
	AccessMismatchRepos []common.NameWithOwner
	NotFoundRepos       []common.NameWithOwner
	NoCodeqlDBRepos     []common.NameWithOwner
	OverLimitRepos      []common.NameWithOwner
	AnalysisRepos       *map[common.NameWithOwner]qldbstore.CodeQLDatabaseLocation
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
