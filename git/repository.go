package git

// Repository represents a cloned git repo within a Workspace.
// Instances for local repositories within a workspace can be instantiated with ScanForRepositories or NewWorkspace.
type Repository struct {
	// Name is the display name for the Repository, and will be the path from the workspace root to the repository
	// directory when not explicitly set.
	Name string
	// Dir is the absolute path to the repository.
	Dir string
	// Git provides access To GitDetails data such as GitDetails.Url.
	Git *GitDetails
}

// GitDetails contains git program details for Repository.
type GitDetails struct {
	// Url is the repo's git remote origin.
	Url string
}

// NewRepository creates a Repository.
func NewRepository(name string, dir string, url string) *Repository {
	return &Repository{
		Name: name,
		Dir:  dir,
		Git:  &GitDetails{Url: url},
	}
}
