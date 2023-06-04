package main

import (
	"errors"
	"fmt"
	"github.com/eighty4/maestro/git"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
)

type Config struct {
	Filename     string
	Repositories []*git.Repository
}

type config struct {
	Workspace *workspace
}

type workspace struct {
	Repositories []*repository
}

type repository struct {
	Name string
	Path string
	Git  *git.RemoteDetails
}

func (r *repository) mapToExternalType(parentDir string) (*git.Repository, error) {
	if r.Git == nil || r.Git.Url == "" {
		return nil, errors.New("missing git.url")
	}
	repoName := r.Name
	repoPath := r.Path
	if repoPath == "" {
		repoPath = git.RepoNameFromUrl(r.Git.Url)
	}
	if repoName == "" {
		repoName = repoPath
		for {
			if repoName[0] == '.' || repoName[0] == os.PathSeparator {
				repoName = repoName[1:]
			} else {
				break
			}
		}
	}
	repoDir := filepath.Join(parentDir, repoPath)
	et := &git.Repository{
		Name: repoName,
		Dir:  repoDir,
		Git:  r.Git,
	}
	return et, nil
}

func parseConfig(dir string) (*Config, error) {
	for _, filename := range []string{"maestro.yaml", "maestro.yml"} {
		if content, err := os.ReadFile(filepath.Join(dir, filename)); err != nil {
			if os.IsNotExist(err) {
				continue
			} else {
				return nil, err
			}
		} else {
			return parseConfigFile(dir, filename, content)
		}
	}
	return nil, nil
}

func parseConfigFile(dir string, filename string, content []byte) (*Config, error) {
	c, err := parseConfigBytes(dir, content)
	if c != nil {
		c.Filename = filename
	}
	return c, err
}

func parseConfigBytes(dir string, bytes []byte) (*Config, error) {
	var c config
	if err := yaml.Unmarshal(bytes, &c); err != nil {
		return nil, err
	}

	var repositories []*git.Repository
	if c.Workspace != nil {
		for i, repo := range c.Workspace.Repositories {
			repo, err := repo.mapToExternalType(dir)
			if err != nil {
				return nil, fmt.Errorf("$.workspace.repositories[%d] error %s", i, err.Error())
			}
			repositories = append(repositories, repo)
		}
	}
	return &Config{Repositories: repositories}, nil
}
