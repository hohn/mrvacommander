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

type DBSpec struct {
	Host     string
	Port     int
	User     string
	Password string
	DBname   string
}

type DBInfo struct {
	// Database version of
	// info map[common.JobSpec]common.JobInfo       = make(map[common.JobSpec]common.JobInfo)
	gorm.Model
	JobSpec common.JobSpec `gorm:"type:jsonb"`
	JobInfo common.JobInfo `gorm:"type:jsonb"`
}

type DBJobs struct {
	// Database version of
	// jobs   map[int][]common.AnalyzeJob             = make(map[int][]common.AnalyzeJob)
	gorm.Model
	JobKey     int
	AnalyzeJob common.AnalyzeJob `gorm:"type:jsonb"`
}

type DBResult struct {
	// Database version of
	// result map[common.JobSpec]common.AnalyzeResult = make(map[common.JobSpec]common.AnalyzeResult)
	gorm.Model
	JobSpec       common.JobSpec       `gorm:"type:jsonb"`
	AnalyzeResult common.AnalyzeResult `gorm:"type:jsonb"`
}

type DBStatus struct {
	// Database version of
	// status map[common.JobSpec]common.Status        = make(map[common.JobSpec]common.Status)
	gorm.Model
	JobSpec common.JobSpec `gorm:"type:jsonb"`
	Status  common.Status  `gorm:"type:jsonb"`
}

type StorageContainer struct {
	// Database version of StorageSingle
	RequestID int
	DB        *gorm.DB
}
