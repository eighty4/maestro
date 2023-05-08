package git

import (
	"strings"
)

// Repository represents a cloned git repo within a Workspace.
// Instances for local repositories within a workspace can be instantiated with ScanForRepositories or NewWorkspace.
type Repository struct {
	// Name is the display name for the Repository, and will be the path from the workspace root to the repository
	// directory when not explicitly set.
	Name string
	// Dir is the absolute path to the repository.
	Dir string
	// Git specific config for the repository remote such as RemoteDetails.Url.
	Git *RemoteDetails
}

// todo validate git repo url: const urlPattern = `^(git@|https:\/\/)(github.com)(:|\/)(.+)\/(.+)\.git$`

// RemoteDetails contains git program details for a Repository.
type RemoteDetails struct {
	// Url is the repo's git remote origin.
	Url string
}

// NewRepository creates a Repository.
func NewRepository(name string, dir string, url string) *Repository {
	return &Repository{
		Name: name,
		Dir:  dir,
		Git:  &RemoteDetails{Url: url},
	}
}

func RepoNameFromUrl(url string) string {
	repoName := url[strings.LastIndex(url, "/")+1:]
	extPos := strings.Index(repoName, ".git")
	if extPos == -1 {
		return repoName
	} else {
		return repoName[:extPos]
	}
}
