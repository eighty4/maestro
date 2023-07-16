package main

import (
	"github.com/eighty4/maestro/git"
	"github.com/eighty4/maestro/testutil"
	"github.com/eighty4/maestro/util"
	"github.com/stretchr/testify/assert"
	"path/filepath"
	"testing"
)

func TestParseConfig_CreatesConfig_WithYamlExtension(t *testing.T) {
	dir := testutil.MkTmpDir(t)
	defer testutil.RmDir(t, dir)
	testutil.WriteContentToFile(t, dir, "maestro.yaml", "---\n")

	c, err := parseConfigFile(dir)
	assert.Nil(t, err)
	assert.NotNil(t, c)
	assert.Equal(t, dir, c.Dir)
	assert.Equal(t, true, c.FileExists)
	assert.Equal(t, "maestro.yaml", c.Filename)
}

func TestParseConfig_CreatesConfig_WithYmlExtension(t *testing.T) {
	dir := testutil.MkTmpDir(t)
	defer testutil.RmDir(t, dir)
	testutil.WriteContentToFile(t, dir, "maestro.yml", "---\n")

	c, err := parseConfigFile(dir)
	assert.Nil(t, err)
	assert.NotNil(t, c)
	assert.Equal(t, dir, c.Dir)
	assert.Equal(t, true, c.FileExists)
	assert.Equal(t, "maestro.yml", c.Filename)
}

func TestParseConfig_ReturnsNilNil_WithMissingConfigFile(t *testing.T) {
	dir := testutil.MkTmpDir(t)
	defer testutil.RmDir(t, dir)

	c, err := parseConfigFile(dir)
	assert.Nil(t, err)
	assert.NotNil(t, c)

	assert.Equal(t, dir, c.Dir)
	assert.Equal(t, false, c.FileExists)
	assert.Equal(t, "", c.Filename)
	assert.Len(t, c.Packages, 0)
	assert.Len(t, c.Repositories, 0)
}

func TestParseProjectConfigText(t *testing.T) {
	dir := testutil.MkTmpDir(t)
	testutil.MkDir(t, filepath.Join(dir, "api"))
	testutil.MkDir(t, filepath.Join(dir, "web"))
	defer testutil.RmDir(t, dir)
	configText := `
---
project:
  packages:
    - name: api
      path: api
      commands:
        - desc: blah blah
          exec: cargo run 
          name: blah
    - path: ./web
      commands:
        - id: npm.run:dev
          name: dev
`
	c, err := parseConfigBytes(dir, "maestro.yaml", []byte(configText))

	assert.Nil(t, err)
	assert.Equal(t, dir, c.Dir)
	assert.Equal(t, true, c.FileExists)
	assert.Equal(t, "maestro.yaml", c.Filename)
	assert.Len(t, c.Packages, 2)
	assert.Len(t, c.Repositories, 0)

	assert.Equal(t, filepath.Join(dir, "api"), c.Packages[0].dir)
	assert.Equal(t, "api", c.Packages[0].name)
	assert.Len(t, c.Packages[0].commands, 1)
	assert.Equal(t, "", c.Packages[0].commands[0].Archetype)
	assert.Equal(t, "blah blah", c.Packages[0].commands[0].Desc)
	assert.Equal(t, filepath.Join(dir, "api"), c.Packages[0].commands[0].Dir)
	assert.Equal(t, "cargo", c.Packages[0].commands[0].Exec.Binary)
	assert.Equal(t, []string{"run"}, c.Packages[0].commands[0].Exec.Args)
	assert.Equal(t, "", c.Packages[0].commands[0].Id)
	assert.Equal(t, "blah", c.Packages[0].commands[0].Name)

	assert.Equal(t, filepath.Join(dir, "web"), c.Packages[1].dir)
	assert.Equal(t, "web", c.Packages[1].name)
	assert.Len(t, c.Packages[1].commands, 1)
	assert.Equal(t, "npm.run", c.Packages[1].commands[0].Archetype)
	assert.Equal(t, "", c.Packages[1].commands[0].Desc)
	assert.Equal(t, filepath.Join(dir, "web"), c.Packages[1].commands[0].Dir)
	assert.Equal(t, "npm", c.Packages[1].commands[0].Exec.Binary)
	assert.Equal(t, []string{"run", "dev"}, c.Packages[1].commands[0].Exec.Args)
	assert.Equal(t, "npm.run:dev", c.Packages[1].commands[0].Id)
	assert.Equal(t, "dev", c.Packages[1].commands[0].Name)
}

func TestParseProjectConfigText_RaisesMissingPackagePathError(t *testing.T) {
	dir := testutil.MkTmpDir(t)
	testutil.MkDir(t, filepath.Join(dir, "api"))
	defer testutil.RmDir(t, dir)
	configText := `
---
project:
  packages:
    - name: api
      commands:
        - exec: cargo run 
`
	c, err := parseConfigBytes(dir, "maestro.yaml", []byte(configText))

	assert.Nil(t, c)
	assert.NotNil(t, err)
	assert.Equal(t, "$.project.packages[0] missing path", err.Error())
}

func TestParseProjectConfigText_RaisesNonExistingPackagePathError(t *testing.T) {
	configText := `
---
project:
  packages:
    - path: api
      commands:
        - exec: cargo run 
`
	c, err := parseConfigBytes(util.Cwd(), "maestro.yaml", []byte(configText))

	assert.Nil(t, c)
	assert.NotNil(t, err)
	assert.Equal(t, "$.project.packages[0] uses non-existing path api", err.Error())
}

func TestParseProjectConfigText_RaisesNewCommandError(t *testing.T) {
	dir := testutil.MkTmpDir(t)
	testutil.MkDir(t, filepath.Join(dir, "api"))
	defer testutil.RmDir(t, dir)
	configText := `
---
project:
  packages:
    - path: api
      commands:
        - name: foo
`
	c, err := parseConfigBytes(dir, "maestro.yaml", []byte(configText))

	assert.Nil(t, c)
	assert.NotNil(t, err)
	assert.Equal(t, "$.project.packages[0].commands[0] does not specify an exec string or an id", err.Error())
}

func TestParseProjectConfigText_RaisesEmptyCommandsError(t *testing.T) {
	dir := testutil.MkTmpDir(t)
	testutil.MkDir(t, filepath.Join(dir, "api"))
	defer testutil.RmDir(t, dir)
	configText := `
---
project:
  packages:
    - path: api
      commands:
`
	c, err := parseConfigBytes(dir, "maestro.yaml", []byte(configText))

	assert.Nil(t, c)
	assert.NotNil(t, err)
	assert.Equal(t, "$.project.packages[0] missing configured commands", err.Error())
}

func TestParseWorkspaceConfigText(t *testing.T) {
	configText := `
---
workspace:
  repositories:
    - name: todai
      path: apps/todai
      git:
        url: git@github.com:eighty4/todai
`
	c, err := parseConfigBytes(util.Cwd(), "maestro.yaml", []byte(configText))

	assert.Nil(t, err)
	assert.Equal(t, util.Cwd(), c.Dir)
	assert.Equal(t, true, c.FileExists)
	assert.Equal(t, "maestro.yaml", c.Filename)
	assert.Len(t, c.Packages, 0)
	assert.Len(t, c.Repositories, 1)
	repo := c.Repositories[0]
	assert.Equal(t, "todai", repo.Name)
	assert.Equal(t, filepath.Join(util.Cwd(), "apps/todai"), repo.Dir)
	assert.Equal(t, "git@github.com:eighty4/todai", repo.Git.Url)
}

func TestParseWorkspaceConfigText_RaisesMissingGitUrlError(t *testing.T) {
	configText := `
---
workspace:
  repositories:
    - name: todai
      path: apps/todai
`
	c, err := parseConfigBytes(util.Cwd(), "maestro.yaml", []byte(configText))

	assert.Nil(t, c)
	assert.NotNil(t, err)
	assert.Equal(t, "$.workspace.repositories[0] missing git.url", err.Error())
}

func TestRepositoryMapToExternalType_OmitName_UsesPath(t *testing.T) {
	r := &configRepository{
		Name: "",
		Path: "apps/todai",
		Git:  &git.RemoteDetails{Url: "git@github.com:eighty4/todai"},
	}
	result, err := r.mapToExternalType(util.Cwd())
	assert.Nil(t, err)
	assert.Equal(t, "apps/todai", result.Name)
}

func TestRepositoryMapToExternalType_OmitName_UsesRelativePath(t *testing.T) {
	r := &configRepository{
		Name: "",
		Path: "./apps/todai",
		Git:  &git.RemoteDetails{Url: "git@github.com:eighty4/todai"},
	}
	result, err := r.mapToExternalType(util.Cwd())
	assert.Nil(t, err)
	assert.Equal(t, "apps/todai", result.Name)
	assert.Equal(t, filepath.Join(util.Cwd(), "apps/todai"), result.Dir)
}

func TestRepositoryMapToExternalType_OmitNameAndPath_UsesGitUrl(t *testing.T) {
	r := &configRepository{
		Name: "",
		Path: "",
		Git:  &git.RemoteDetails{Url: "git@github.com:eighty4/todai"},
	}
	result, err := r.mapToExternalType(util.Cwd())
	assert.Nil(t, err)
	assert.Equal(t, "todai", result.Name)
	assert.Equal(t, filepath.Join(util.Cwd(), "todai"), result.Dir)
}

func TestRepositoryMapToExternalType_OmitGit_ReturnsError(t *testing.T) {
	r := &configRepository{
		Name: "",
		Path: "",
		Git:  nil,
	}
	result, err := r.mapToExternalType(util.Cwd())
	assert.Nil(t, result)
	assert.Equal(t, "missing git.url", err.Error())
}

func TestRepositoryMapToExternalType_OmitGitUrl_ReturnsError(t *testing.T) {
	r := &configRepository{
		Name: "",
		Path: "",
		Git:  &git.RemoteDetails{Url: ""},
	}
	result, err := r.mapToExternalType(util.Cwd())
	assert.Nil(t, result)
	assert.Equal(t, "missing git.url", err.Error())
}
