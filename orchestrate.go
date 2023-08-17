package main

import (
	"fmt"
	"github.com/eighty4/maestro/composable"
	"os"
)

func orchestrateProject(cfg *Config) error {
	if cfg == nil || !cfg.FileExists {
		fmt.Println("This directory is missing a maestro.yaml. Compose a project with `maestro -c`.")
		return nil
	} else if !cfgHasCommands(cfg) {
		fmt.Println("This directory's maestro.yaml does not have any commands configured.")
		return nil
	} else if j, err := NewOrchestrateProjectJob(cfg); err != nil {
		return err
	} else {
		return j.start()
	}
}

func cfgHasCommands(cfg *Config) bool {
	if cfg != nil && cfg.FileExists {
		for _, pkg := range cfg.Packages {
			if len(pkg.commands) > 0 {
				return true
			}
		}
	}
	return false
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

func (j *OrchestrateProjectJob) start() error {
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
