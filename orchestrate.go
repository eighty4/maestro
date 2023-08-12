package main

import (
	"fmt"
	"github.com/eighty4/maestro/composable"
	"os"
)

func orchestrateProject(cfg *Config) error {
	j, err := NewOrchestrateProjectJob(cfg)
	if err != nil {
		return err
	}
	j.start()
	return nil
}

type OrchestrateProjectJob struct {
	cfg         *Config
	composition *composable.Composition
}

func NewOrchestrateProjectJob(cfg *Config) (OrchestrateProjectJob, error) {
	return OrchestrateProjectJob{
		cfg: cfg,
	}, nil
}

func (j *OrchestrateProjectJob) start() {
	var composables []composable.Composable
	for _, pkg := range j.cfg.Packages {
		for _, cmd := range pkg.commands {
			if cmd.Archetype == "docker.compose" {
				fmt.Println("Orchestrating docker.compose has not been implemented")
				os.Exit(1)
			}
			composables = append(composables, composable.NewExecComposable(cmd.Exec))
		}
	}
	j.composition = composable.NewComposition(composables)
	j.composition.Start()
	for {
		select {}
	}
}
