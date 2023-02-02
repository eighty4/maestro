package git

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewRepository(t *testing.T) {
	repo := NewRepository("name", "dir", "url")
	assert.Equal(t, "name", repo.Name)
	assert.Equal(t, "dir", repo.Dir)
	assert.Equal(t, "url", repo.Url)
}
