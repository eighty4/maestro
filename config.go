package main

import (
	"errors"
	"fmt"
	"github.com/eighty4/maestro/git"
	"github.com/eighty4/maestro/util"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
)

type Config struct {
	Dir          string
	FileExists   bool
	Filename     string
	Packages     []*Package
	Repositories []*git.Repository
}

type config struct {
	Project   *configProject
	Workspace *configWorkspace
}

type configProject struct {
	Packages []*configPackage
}

type configPackage struct {
	Name     string
	Path     string
	Commands []*configCommand
}

type configCommand struct {
	Desc string
	Exec string
	Id   string
	Name string
}

type configWorkspace struct {
	Repositories []*configRepository
}

type configRepository struct {
	Name string
	Path string
	Git  *git.RemoteDetails
}

func (r *configRepository) mapToExternalType(parentDir string) (*git.Repository, error) {
	if r.Git == nil || r.Git.Url == "" {
		return nil, errors.New("missing git.url")
	}
	repoName := r.Name
	repoPath := r.Path
	if repoPath == "" {
		repoPath = git.RepoNameFromUrl(r.Git.Url)
	}
	if repoName == "" {
		repoName = util.TrimRelativePathPrefix(repoPath)
	}
	repoDir := filepath.Join(parentDir, repoPath)
	et := &git.Repository{
		Name: repoName,
		Dir:  repoDir,
		Git:  r.Git,
	}
	return et, nil
}

func parseConfigFile(dir string) (*Config, error) {
	for _, filename := range []string{"maestro.yaml", "maestro.yml"} {
		if content, err := os.ReadFile(filepath.Join(dir, filename)); err != nil {
			if os.IsNotExist(err) {
				continue
			} else {
				return nil, err
			}
		} else {
			return parseConfigBytes(dir, filename, content)
		}
	}
	return &Config{Dir: dir, FileExists: false}, nil
}

func parseConfigBytes(dir string, filename string, bytes []byte) (*Config, error) {
	var c config
	if err := yaml.Unmarshal(bytes, &c); err != nil {
		return nil, err
	}

	var packages []*Package
	if c.Project != nil {
		for cfgPkgI, cfgPkg := range c.Project.Packages {
			if len(cfgPkg.Path) == 0 {
				return nil, fmt.Errorf("$.project.packages[%d] missing path", cfgPkgI)
			}
			pkgDir := filepath.Join(dir, cfgPkg.Path)
			if !util.IsDir(pkgDir) {
				return nil, fmt.Errorf("$.project.packages[%d] uses non-existing path %s", cfgPkgI, cfgPkg.Path)
			}
			if len(cfgPkg.Commands) == 0 {
				return nil, fmt.Errorf("$.project.packages[%d] missing configured commands", cfgPkgI)
			}
			var commands []*Command
			for cfgCmdI, cfgCmd := range cfgPkg.Commands {
				command, err := NewCommand(&CommandOptions{
					Desc: cfgCmd.Desc,
					Dir:  pkgDir,
					Exec: cfgCmd.Exec,
					Id:   cfgCmd.Id,
					Name: cfgCmd.Name,
				})
				if err != nil {
					return nil, fmt.Errorf("$.project.packages[%d].commands[%d] %s", cfgPkgI, cfgCmdI, err.Error())
				} else {
					commands = append(commands, command)
				}
			}
			name := cfgPkg.Name
			if len(name) == 0 {
				name = util.TrimRelativePathPrefix(cfgPkg.Path)
			}
			packages = append(packages, &Package{
				dir:      pkgDir,
				name:     name,
				commands: commands,
			})
		}
	}

	var repositories []*git.Repository
	if c.Workspace != nil {
		for i, r := range c.Workspace.Repositories {
			if repo, err := r.mapToExternalType(dir); err != nil {
				return nil, fmt.Errorf("$.workspace.repositories[%d] %s", i, err.Error())
			} else {
				repositories = append(repositories, repo)
			}
		}
	}

	return &Config{
		Dir:          dir,
		FileExists:   true,
		Filename:     filename,
		Packages:     packages,
		Repositories: repositories,
	}, nil
}
