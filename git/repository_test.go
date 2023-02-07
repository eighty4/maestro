package git

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewRepository(t *testing.T) {
	repo := NewRepository("name", "dir", "url")
	assert.Equal(t, "name", repo.Name)
	assert.Equal(t, "dir", repo.Dir)
	assert.Equal(t, "url", repo.Git.Url)
}

func TestRepoNameFromUrl_Https(t *testing.T) {
	assert.Equal(t, "asdf", RepoNameFromUrl("asdf"))
	assert.Equal(t, "todai", RepoNameFromUrl("https://github.com/eighty4/todai.git"))
	assert.Equal(t, "todai", RepoNameFromUrl("https://github.com/eighty4/todai"))
	assert.Equal(t, "todai", RepoNameFromUrl("git@github.com:eighty4/todai.git"))
	assert.Equal(t, "todai", RepoNameFromUrl("git@github.com:eighty4/todai"))
}
