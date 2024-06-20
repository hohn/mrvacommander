package server

import (
	"log/slog"
	"mrvacommander/pkg/artifactstore"
	"mrvacommander/pkg/common"
	"mrvacommander/pkg/qldbstore"
	"mrvacommander/pkg/queue"
	"mrvacommander/pkg/state"
)

type SessionInfo struct {
	ID             int
	Owner          string
	ControllerRepo string
	// XX: check
	QueryPack           artifactstore.ArtifactLocation
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

type CommanderContainer struct {
	v *Visibles
}

func NewCommanderSingle(st *Visibles) *CommanderSingle {
	slog.Debug("Commander started")
	c := CommanderSingle{v: st}
	setupEndpoints(&c)
	return &c
}

func NewCommanderContainer(st *Visibles) *CommanderContainer {
	c := CommanderContainer{v: st}
	setupEndpoints(&c)
	return &c
}

type Visibles struct {
	Queue         queue.Queue
	State         state.CommonState
	Artifacts     artifactstore.ArtifactStore
	CodeQLDBStore qldbstore.CodeQLDatabaseStore
}
