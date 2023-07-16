package util

import (
	"github.com/eighty4/maestro/testutil"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
)

func TestCwd(t *testing.T) {
	cwd, err := os.Getwd()
	assert.Nil(t, err)
	assert.Equal(t, cwd, Cwd())
}

func TestSubdirectories(t *testing.T) {
	dir := testutil.MkTmpDir(t)
	testutil.MkDir(t, filepath.Join(dir, "packages"))
	testutil.MkDir(t, filepath.Join(dir, "packages", "api"))
	testutil.MkDir(t, filepath.Join(dir, "packages", "data"))
	testutil.MkDir(t, filepath.Join(dir, "packages", "data", "sql"))
	testutil.MkDir(t, filepath.Join(dir, "packages", "ui"))
	result := Subdirectories(dir, 2)
	assert.Len(t, result, 4)
	assert.NotContains(t, result, filepath.Join(dir, "packages", "data", "sql"))
}

func TestTrimRelativePathPrefix(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "all prefix",
			path: "./",
			want: "",
		},
		{
			name: "noop",
			path: "asdf",
			want: "asdf",
		},
		{
			name: "unix rel path",
			path: "./asdf",
			want: "asdf",
		},
		{
			name: "windows rel path",
			path: ".\\asdf",
			want: "asdf",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, TrimRelativePathPrefix(tt.path), "TrimRelativePathPrefix(%v)", tt.path)
		})
	}
}
