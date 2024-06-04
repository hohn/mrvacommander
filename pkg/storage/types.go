package storage

import (
	"mrvacommander/pkg/common"

	"gorm.io/gorm"
)

type DBLocation struct {
	Prefix string
	File   string
}

type StorageSingle struct {
	currentID int
}

type DBInfo struct {
	// Database version of
	// info map[common.JobSpec]common.JobInfo       = make(map[common.JobSpec]common.JobInfo)
	gorm.Model
	Key     common.JobSpec
	JobInfo common.JobInfo
}

type DBJobs struct {
	// Database version of
	// jobs   map[int][]common.AnalyzeJob             = make(map[int][]common.AnalyzeJob)
	gorm.Model
	Key        int
	AnalyzeJob common.AnalyzeJob
}

type DBResult struct {
	// Database version of
	// result map[common.JobSpec]common.AnalyzeResult = make(map[common.JobSpec]common.AnalyzeResult)
	gorm.Model
	Key           common.JobSpec
	AnalyzeResult common.AnalyzeResult
}

type DBStatus struct {
	// Database version of
	// status map[common.JobSpec]common.Status        = make(map[common.JobSpec]common.Status)
	gorm.Model
	Key    common.JobSpec
	Status common.Status
}

type StorageContainer struct {
	// Database version of StorageSingle
	RequestID int
	DB        *gorm.DB
}
