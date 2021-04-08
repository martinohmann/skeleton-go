package kickoff

import (
	"fmt"
	"net/url"
	"path/filepath"

	"github.com/martinohmann/kickoff/internal/homedir"
)

// RepoRef holds information about a skeleton repository's location.
type RepoRef struct {
	// Name holds the optional local name for the repository.
	Name string `json:"name,omitempty"`
	// URL to the repository, e.g. 'https://github.com/foobar/baz' or
	// `/some/local/path`.
	URL string `json:"url,omitempty"`
	// Path is the path to a local repository.
	Path string `json:"path,omitempty"`
	// Revision holds the revision of the remote git repository to checkout.
	// This can be a branch, tag or commit SHA.
	Revision string `json:"revision,omitempty"`
}

// String implements fmt.Stringer.
func (r *RepoRef) String() string {
	if r.URL == "" {
		return r.Path
	}

	if r.Revision == "" {
		return r.URL
	}

	return fmt.Sprintf("%s?revision=%s", r.URL, r.Revision)
}

// Validate implements the Validator interface.
func (r *RepoRef) Validate() error {
	if r.IsEmpty() {
		return newRepositoryRefError("URL or Path must be set")
	}

	if r.IsRemote() {
		if _, err := url.Parse(r.URL); err != nil {
			return newRepositoryRefError("invalid URL: %w", err)
		}
	}

	return nil
}

// IsEmpty return true if l is empty, that is: it neither describes a local nor
// remote repository.
func (r *RepoRef) IsEmpty() bool {
	return r.URL == "" && r.Path == ""
}

// IsRemote returns true if the repo ref describes a remote repository.
func (r *RepoRef) IsRemote() bool {
	return r.URL != ""
}

// ParseRepoRef parses a raw repository url and returns a repository ref
// describing a local or remote skeleton repository. The rawurl parameter must
// be either a local path or a remote url to a git repository. Remote url may
// optionally include a `revision` query parameter. If absent, `master` will be
// assumed. Returns an error if url does not match any of the criteria
// mentioned above.
func ParseRepoRef(rawurl string) (*RepoRef, error) {
	u, err := url.Parse(rawurl)
	if err != nil {
		return nil, fmt.Errorf("invalid repo URL %q: %w", rawurl, err)
	}

	if u.Host == "" {
		path, err := homedir.Expand(u.Path)
		if err != nil {
			return nil, err
		}

		return &RepoRef{Path: path}, nil
	}

	query, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return nil, fmt.Errorf("invalid URL query %q: %w", u.RawQuery, err)
	}

	var revision string
	if rev, ok := query["revision"]; ok && len(rev) > 0 {
		revision = rev[0]
	}

	if revision == "" {
		revision = "master"
	}

	// Query is only used to pass an optional revision and needs to be empty in
	// the final repository URL.
	u.RawQuery = ""

	return &RepoRef{
		Path:     buildLocalCacheDir(u.Host, u.Path, revision),
		URL:      u.String(),
		Revision: revision,
	}, nil
}

func buildLocalCacheDir(host, path, revision string) string {
	revision = url.PathEscape(revision)

	return filepath.Join(LocalRepositoryCacheDir, host, fmt.Sprintf("%s@%s", path, revision))
}
