package server

import (
	"github.com/hohn/mrvacommander/pkg/artifactstore"
	"github.com/hohn/mrvacommander/pkg/common"
	"github.com/hohn/mrvacommander/pkg/qldbstore"
	"github.com/hohn/mrvacommander/pkg/queue"
	"github.com/hohn/mrvacommander/pkg/state"
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
