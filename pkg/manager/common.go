package manager

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/mattn/go-zglob"
	"github.com/rotisserie/eris"
	"github.com/solo-io/protodep/protodep"
	"github.com/spf13/afero"
)

/*
	This interface is used to abstract away the methods which require ENV vars or other
	system things. This is mostly for unit testing purposes.
*/
type FileCopier interface {
	Copy(src, dst string) (int64, error)
	GetMatches(copyPat []string, dir string) ([]string, error)
	PkgModPath(importPath, version string) string
}

var matchListFilter = fmt.Sprintf("%s/", protodep.DefaultDepDir)

type copier struct {
	fs afero.Fs
}

func (c *copier) GetMatches(copyPat []string, dir string) ([]string, error) {
	var vendorList []string

	for _, pat := range copyPat {
		matches, err := zglob.Glob(filepath.Join(dir, pat))
		if err != nil {
			return nil, eris.Wrapf(err, "Error! glob match failure")
		}
		// Filter out all matches which contain a vendor folder, those are leftovers from a previous run.
		// Might be worth clearing the vendor folder before every run.
		for _, match := range matches {
			vendorFolders := strings.Count(match, matchListFilter)
			if vendorFolders > 0 {
				continue
			}
			vendorList = append(vendorList, match)
		}
	}

	return vendorList, nil
}

func (c *copier) PkgModPath(importPath, version string) string {
	goPath := os.Getenv("GOPATH")
	if goPath == "" {
		// the default GOPATH for go v1.11
		goPath = filepath.Join(os.Getenv("HOME"), "go")
	}

	var normPath string

	for _, char := range importPath {
		if unicode.IsUpper(char) {
			normPath += "!" + string(unicode.ToLower(char))
		} else {
			normPath += string(char)
		}
	}

	return filepath.Join(goPath, "pkg", "mod", fmt.Sprintf("%s@%s", normPath, version))
}

func NewCopier(fs afero.Fs) *copier {
	return &copier{
		fs: fs,
	}
}

var (
	IrregularFileError = func(file string) error {
		return eris.Errorf("%s is not a regular file", file)
	}
)

func NewDefaultCopier() *copier {
	return &copier{fs: afero.NewOsFs()}
}

func (c *copier) Copy(src, dst string) (int64, error) {
	log.Printf("copying %v -> %v", src, dst)

	if err := c.fs.MkdirAll(filepath.Dir(dst), os.ModePerm); err != nil {
		return 0, err
	}

	srcStat, err := c.fs.Stat(src)
	if err != nil {
		return 0, err
	}

	if !srcStat.Mode().IsRegular() {
		return 0, IrregularFileError(src)
	}

	srcFile, err := c.fs.Open(src)
	if err != nil {
		return 0, err
	}
	defer srcFile.Close()

	dstFile, err := c.fs.Create(dst)
	if err != nil {
		return 0, err
	}
	defer dstFile.Close()

	return io.Copy(dstFile, srcFile)
}
