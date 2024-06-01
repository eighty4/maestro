package main

import (
	"fmt"
	"github.com/eighty4/maestro/composable"
	"log"
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
	var pkgCompositions []*composable.Composition
	for _, pkg := range j.cfg.Packages {
		pkgCompositions = append(pkgCompositions, j.createPackageComposition(pkg))
	}
	var composables []composable.Composable
	j.composition = composable.NewComposition(composables)
	j.composition.Start()
	startApiEndpoint(j.composition)
	return nil
}

func (j *OrchestrateProjectJob) createPackageComposition(pkg *Package) *composable.Composition {
	var composables []composable.Composable
	for _, cmd := range pkg.commands {
		if cmd.Archetype == "docker.compose" {
			log.Fatalln("Orchestrating docker.compose has not been implemented")
		}
		composables = append(composables, cmd.Exec.Process())
	}
	return composable.NewComposition(composables)
}
