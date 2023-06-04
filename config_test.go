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

	c, err := parseConfig(dir)
	assert.Nil(t, err)
	assert.NotNil(t, c)
}

func TestParseConfig_CreatesConfig_WithYmlExtension(t *testing.T) {
	dir := testutil.MkTmpDir(t)
	defer testutil.RmDir(t, dir)
	testutil.WriteContentToFile(t, dir, "maestro.yml", "---\n")

	c, err := parseConfig(dir)
	assert.Nil(t, err)
	assert.NotNil(t, c)
}

func TestParseConfig_ReturnsNilNil_WithMissingConfigFile(t *testing.T) {
	dir := testutil.MkTmpDir(t)
	defer testutil.RmDir(t, dir)

	c, err := parseConfig(dir)
	assert.Nil(t, c)
	assert.Nil(t, err)
}

func TestParseConfigText(t *testing.T) {
	configText := `
---
workspace:
  repositories:
    - name: todai
      path: apps/todai
      git:
        url: git@github.com:eighty4/todai
`
	c, err := parseConfigBytes(util.Cwd(), []byte(configText))

	assert.Nil(t, err)
	assert.Len(t, c.Repositories, 1)
	repo := c.Repositories[0]
	assert.Equal(t, "todai", repo.Name)
	assert.Equal(t, filepath.Join(util.Cwd(), "apps/todai"), repo.Dir)
	assert.Equal(t, "git@github.com:eighty4/todai", repo.Git.Url)
}

func TestParseConfigText_RaisesError(t *testing.T) {
	configText := `
---
workspace:
  repositories:
    - name: todai
      path: apps/todai
`
	c, err := parseConfigBytes(util.Cwd(), []byte(configText))

	assert.Nil(t, c)
	assert.NotNil(t, err)
	assert.Equal(t, "$.workspace.repositories[0] error missing git.url", err.Error())
}

func TestRepositoryMapToExternalType_OmitName_UsesPath(t *testing.T) {
	r := &repository{
		Name: "",
		Path: "apps/todai",
		Git:  &git.RemoteDetails{Url: "git@github.com:eighty4/todai"},
	}
	result, err := r.mapToExternalType(util.Cwd())
	assert.Nil(t, err)
	assert.Equal(t, "apps/todai", result.Name)
}

func TestRepositoryMapToExternalType_OmitName_UsesRelativePath(t *testing.T) {
	r := &repository{
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
	r := &repository{
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
	r := &repository{
		Name: "",
		Path: "",
		Git:  nil,
	}
	result, err := r.mapToExternalType(util.Cwd())
	assert.Nil(t, result)
	assert.Equal(t, "missing git.url", err.Error())
}

func TestRepositoryMapToExternalType_OmitGitUrl_ReturnsError(t *testing.T) {
	r := &repository{
		Name: "",
		Path: "",
		Git:  &git.RemoteDetails{Url: ""},
	}
	result, err := r.mapToExternalType(util.Cwd())
	assert.Nil(t, result)
	assert.Equal(t, "missing git.url", err.Error())
}
