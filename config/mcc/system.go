package mcc

import (
	"log/slog"
	"os"

	"github.com/BurntSushi/toml"
)

type System struct {
	Commander Commander
	Logger    Logger
	Queue     Queue
	Storage   Storage
	Runner    Runner
}

func LoadConfig(fname string) *System {
	if _, err := os.Stat(fname); err != nil {
		slog.Warn("Configuration file not found", "name", fname)
		return &System{}
	}

	var config System

	_, err := toml.DecodeFile(fname, &config)
	if err != nil {
		slog.Error("Error decoding configuration file", err)
		os.Exit(1)
	}

	return &config
}
