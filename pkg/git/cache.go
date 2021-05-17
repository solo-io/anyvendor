package git

import (
	"log"
	"os"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

// set to override cache dir
var CacheDir = os.Getenv("HOME") + "/.anyvendor/git"

// set to override progress logging
var ProgressOut = os.Stdout

// GitVendorCache maintains a local cache of vendored git repos
type GitVendorCache struct {
	Dir string
}

func DefaultCache() *GitVendorCache {
	return &GitVendorCache{Dir: CacheDir}
}

func (c *GitVendorCache) Init() error {
	return os.MkdirAll(c.Dir, 0777)
}

func (c *GitVendorCache) EnsureCheckedOut(
	url, sha, tag, authUser, authToken string,
) error {
	repoDir, _ := c.GetRepoDir(url)
	repoExists, err := fileExists(repoDir)
	if err != nil {
		return err
	}
	var authMethod transport.AuthMethod
	if authToken != "" {
		authMethod = &http.BasicAuth{Username: authUser, Password: authToken}
	}
	var repo *git.Repository
	if repoExists {
		log.Printf("using repo %v in local cache %v", url, repoDir)
		repo, err = git.PlainOpen(repoDir)
	} else {
		log.Printf("cloning repo %v to local cache %v", url, repoDir)
		repo, err = git.PlainClone(repoDir, false, &git.CloneOptions{
			URL:      url,
			Progress: ProgressOut,
			Auth:     authMethod,
		})
	}
	if err != nil {
		return err
	}

	log.Printf("fetching repo")
	if err := repo.Fetch(&git.FetchOptions{
		RemoteName:      "origin",
		RefSpecs:        nil,
		Depth:           0,
		Auth:            authMethod,
		Progress:        ProgressOut,
		Tags:            0,
		Force:           false,
		InsecureSkipTLS: false,
		CABundle:        nil,
	}); err != nil && err != git.NoErrAlreadyUpToDate {
		return err
	}

	wt, err := repo.Worktree()
	if err != nil {
		return err
	}

	// allow disabling hard reset
	if os.Getenv("DISABLE_HARD_RESET") != "1" {
		log.Printf("performing hard reset")
		if err := wt.Reset(&git.ResetOptions{
			Mode: git.HardReset,
		}); err != nil {
			return err
		}
	}

	checkout := &git.CheckoutOptions{
		Create: false,
		Force:  false,
		Keep:   false,
	}
	var ref plumbing.ReferenceName
	switch {
	case sha != "":
		log.Printf("checking out with sha %v", sha)
		checkout.Hash = plumbing.NewHash(sha)
		ref = plumbing.NewBranchReferenceName(sha)
		if err := wt.Checkout(checkout); err != nil {
			return err
		}

	case tag != "":
		log.Printf("checking out with tag %v", tag)
		checkout.Branch = plumbing.NewTagReferenceName(tag)
		ref = plumbing.NewTagReferenceName(tag)
		if err := wt.Checkout(checkout); err != nil {
			return err
		}

		log.Printf("pulling ref %v", ref)
		if err := wt.Pull(&git.PullOptions{
			RemoteName:    "origin",
			ReferenceName: ref,
			SingleBranch:  true,
			Depth:         1,
			Progress:      ProgressOut,
			Force:         false,
			Auth:          authMethod,
		}); err != nil && err != git.NoErrAlreadyUpToDate {
			return err
		}

	}

	return nil
}

func (c *GitVendorCache) GetRepoDir(url string) (string, string) {
	repoDir := strings.TrimPrefix(url, "git://")
	repoDir = strings.TrimPrefix(repoDir, "https://")
	repoDir = strings.TrimPrefix(repoDir, "http://")
	repoDir = strings.TrimSuffix(repoDir, ".git")
	return c.Dir + "/" + repoDir, repoDir
}

func fileExists(path string) (bool, error) {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		} else {
			return false, err
		}
	}
	return true, nil
}
