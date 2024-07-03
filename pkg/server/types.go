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
	QueryPack           string
	Language            queue.QueryLanguage
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
