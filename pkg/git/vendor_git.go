package git

import (
	"fmt"
	"path/filepath"

	"github.com/rotisserie/eris"
	"github.com/solo-io/anyvendor/pkg/manager"
	"github.com/spf13/afero"
)

// vendor files from git repositories
type VendorOptions struct {
	GitRepositories []GitRepository
}

func (r VendorOptions) Vendor(cache *GitVendorCache, vendorDir string) error {
	for _, repository := range r.GitRepositories {
		if err := repository.Vendor(cache, vendorDir); err != nil {
			return err
		}
	}
	return nil
}

// vendor files from a git repository
type GitRepository struct {
	// The repo URL
	URL string
	// provide one of SHA or Tag
	SHA string
	Tag string

	// HTTP Auth User (for private repository)
	AuthUser string
	// HTTP Auth Token (for private repository)
	AuthToken string

	// match files with these patterns in the repo
	MatchPatterns []string
	// skip these dirs when vendoring files
	SkipDirs []string
}

func (r *GitRepository) Vendor(cache *GitVendorCache, vendorDir string) error {
	if err := cache.EnsureCheckedOut(
		r.URL,
		r.SHA,
		r.Tag,
		r.AuthUser,
		r.AuthToken,
	); err != nil {
		return err
	}
	cachedRepoDir, repoRelativePath := cache.GetRepoDir(r.URL)

	fileCopier := manager.NewCopier(afero.NewOsFs(), r.SkipDirs)
	filesToCopy, err := fileCopier.GetMatches(r.MatchPatterns, cachedRepoDir)
	if err != nil {
		return err
	}
	for _, cachedFile := range filesToCopy {
		copiedFileSuffix := filepath.Join(repoRelativePath, cachedFile[len(cachedRepoDir):])
		copiedFile := filepath.Join(vendorDir, copiedFileSuffix)
		if _, err := fileCopier.Copy(cachedFile, copiedFile); err != nil {
			return eris.Wrap(err, fmt.Sprintf("Error! %s - unable to copy file %s\n",
				err.Error(), cachedFile))
		}
	}
	return nil
}
