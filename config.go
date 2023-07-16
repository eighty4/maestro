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

func (c *Config) AddPackages(packages []*Package) {
	c.Packages = append(c.Packages, packages...)
}

func (c *Config) SaveConfig() error {
	cy, err := convertConfigToYamlModel(c)
	if err != nil {
		return err
	}
	bytes, err := yaml.Marshal(cy)
	if err != nil {
		return err
	}
	filename := "maestro.yaml"
	err = util.WriteFile(filepath.Join(c.Dir, filename), bytes)
	if err != nil {
		return err
	}
	c.FileExists = true
	c.Filename = filename
	return nil
}

type config struct {
	Project   *configProject   `yaml:"project,omitempty"`
	Workspace *configWorkspace `yaml:"workspace,omitempty"`
}

type configProject struct {
	Packages []*configPackage
}

type configPackage struct {
	Name     string           `yaml:"name,omitempty"`
	Path     string           `yaml:"path,omitempty"`
	Commands []*configCommand `yaml:"commands,omitempty"`
}

type configCommand struct {
	Desc string `yaml:"desc,omitempty"`
	Exec string `yaml:"exec,omitempty"`
	Id   string `yaml:"id,omitempty"`
	Name string `yaml:"name,omitempty"`
}

type configWorkspace struct {
	Repositories []*configRepository
}

type configRepository struct {
	Name string `yaml:"name,omitempty"`
	Path string `yaml:"path,omitempty"`
	Git  *git.RemoteDetails
}

func convertConfigToYamlModel(c *Config) (*config, error) {
	py, err := convertPackagesToYamlModel(c.Dir, c.Packages)
	if err != nil {
		return nil, err
	}
	wy, err := convertRepositoriesToYamlModel(c.Dir, c.Repositories)
	if err != nil {
		return nil, err
	}
	cy := &config{Project: py, Workspace: wy}
	return cy, nil
}

func convertPackagesToYamlModel(rootDir string, packages []*Package) (*configProject, error) {
	if len(packages) == 0 {
		return nil, nil
	}
	result := &configProject{}
	for _, pkg := range packages {
		path, err := filepath.Rel(rootDir, pkg.dir)
		if err != nil {
			return nil, err
		}
		var cmds []*configCommand
		for _, cmd := range pkg.commands {
			cmds = append(cmds, &configCommand{
				Desc: cmd.Desc,
				Id:   cmd.Id,
				Name: cmd.Name,
			})
		}
		name := pkg.name
		if name == path {
			name = ""
		}
		result.Packages = append(result.Packages, &configPackage{
			Commands: cmds,
			Name:     name,
			Path:     path,
		})
	}
	return result, nil
}

func convertRepositoriesToYamlModel(rootDir string, repositories []*git.Repository) (*configWorkspace, error) {
	if len(repositories) == 0 {
		return nil, nil
	}
	result := &configWorkspace{}
	for _, repo := range repositories {
		path, err := filepath.Rel(rootDir, repo.Dir)
		if err != nil {
			return nil, err
		}
		result.Repositories = append(result.Repositories, &configRepository{
			Name: repo.Name,
			Path: path,
			Git:  repo.Git,
		})
	}
	return result, nil
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
