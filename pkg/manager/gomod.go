package manager

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rotisserie/eris"
	"github.com/solo-io/protodep/pkg/modutils"
	"github.com/solo-io/protodep/protodep"
	"github.com/spf13/afero"
)

const (
	ProtoMatchPattern = "**/*.proto"
)

var (
	// offer sane defaults for proto vendoring
	DefaultMatchPatterns = []string{ProtoMatchPattern}
)

type goModOptions struct {
	MatchOptions  []*protodep.GoModImport
	LocalMatchers []string
}

// struct which represents a go module package in the module package list
type Module struct {
	ImportPath     string
	SourcePath     string
	Version        string
	SourceVersion  string
	Dir            string   // full path, $GOPATH/pkg/mod/
	VendorList     []string // files to vendor
	currentPackage bool
}

func NewGoModFactory(cwd string) (*goModFactory, error) {
	if !filepath.IsAbs(cwd) {
		absoluteDir, err := filepath.Abs(cwd)
		if err != nil {
			return nil, err
		}
		cwd = absoluteDir
	}
	fs := afero.NewOsFs()
	return &goModFactory{
		WorkingDirectory: cwd,
		fs:               fs,
		fileCopier:       NewCopier(fs),
	}, nil
}

type goModFactory struct {
	WorkingDirectory string
	packageName      bool
	fs               afero.Fs
	fileCopier       FileCopier
}

func (m *goModFactory) Ensure(ctx context.Context, opts *protodep.Config) error {
	var packages []*protodep.GoModImport
	for _, cfg := range opts.Imports {
		if cfg.GetGoMod() != nil {
			packages = append(packages, cfg.GetGoMod())
		}
	}
	mods, err := m.gather(goModOptions{
		MatchOptions:  packages,
		LocalMatchers: opts.GetLocal().GetPatterns(),
	})
	if err != nil {
		return err
	}

	err = m.copy(mods)
	if err != nil {
		return err
	}
	return nil
}

// gather up all packages for a given go module
// currently this function uses the cmd `go list -m all` to figure out the list of dep
// all of the logic surrounding go.mod and the go cli calls are in the modutils package
func (m *goModFactory) gather(opts goModOptions) ([]*Module, error) {
	matchOptions := opts.MatchOptions
	// Ensure go.mod file exists and we're running from the project root,
	modPackageFile, err := modutils.GetCurrentModPackageFile()
	if err != nil {
		return nil, err
	}

	packageName, err := modutils.GetCurrentModPackageName(modPackageFile)
	if err != nil {
		return nil, err
	}

	modPackageReader, err := modutils.GetCurrentPackageList()
	if err != nil {
		return nil, err
	}

	// split list of packages from cmd by line
	scanner := bufio.NewScanner(modPackageReader)
	scanner.Split(bufio.ScanLines)

	// Clear first line as it is current package name
	scanner.Scan()

	modules := []*Module{}

	for scanner.Scan() {
		line := scanner.Text()
		s := strings.Split(line, " ")

		if s[1] == "=>" {
			// issue https://github.com/golang/go/issues/33848 added these,
			// see comments. I think we can get away with ignoring them.
			return nil, nil
		}

		mod, err := m.handleSingleModule(s, matchOptions)
		if err != nil {
			return nil, err
		}
		if len(mod.VendorList) > 0 {
			modules = append(modules, mod)
		}

	}

	localModule := &Module{
		Dir:            m.WorkingDirectory,
		ImportPath:     packageName,
		currentPackage: true,
	}
	localModule.VendorList, err = m.fileCopier.GetMatches(opts.LocalMatchers, localModule.Dir)
	if err != nil {
		return nil, err
	}
	modules = append(modules, localModule)

	return modules, nil
}

func (m *goModFactory) handleSingleModule(s []string, matchOptions []*protodep.GoModImport) (*Module, error) {
	/*
		the packages come in 3 varities
		1. helm.sh/helm/v3 v3.0.0
		2. k8s.io/api v0.0.0-20191121015604-11707872ac1c => k8s.io/api v0.0.0-20191004120104-195af9ec3521
		3. k8s.io/api v0.0.0-20191121015604-11707872ac1c => /path/to/local

		All three variants share the same first 2 members
	*/
	module := &Module{
		ImportPath: s[0],
		Version:    s[1],
	}
	// Handle "replace" in module file if any
	if len(s) > 2 && s[2] == "=>" {
		module.SourcePath = s[3]
		// non-local module with version
		if len(s) >= 5 {
			// see case 2 above
			module.SourceVersion = s[4]
			module.Dir = m.fileCopier.PkgModPath(module.SourcePath, module.SourceVersion)
		} else {
			// see case 3 above
			moduleAbsolutePath, err := filepath.Abs(module.SourcePath)
			if err != nil {
				return nil, err
			}
			module.Dir = moduleAbsolutePath
		}
	} else {
		module.Dir = m.fileCopier.PkgModPath(module.ImportPath, module.Version)
	}

	// make sure module exists
	if _, err := m.fs.Stat(module.Dir); os.IsNotExist(err) {
		return nil, eris.Wrapf(err, "Error! %q module path does not exist, check $GOPATH/pkg/mod. "+
			"Try running go mod download\n", module.Dir)
	}

	// If no match options have been supplied, match on all packages using default match patterns
	if matchOptions == nil {
		// Build list of files to module path source to project vendor folder
		vendorList, err := m.fileCopier.GetMatches(DefaultMatchPatterns, module.Dir)
		if err != nil {
			return nil, err
		}
		module.VendorList = vendorList
		return module, nil
	}

	for _, matchOpt := range matchOptions {
		// only check module if is in imports list, or imports list in empty
		if len(matchOpt.Package) != 0 &&
			!strings.Contains(module.ImportPath, matchOpt.Package) {
			continue
		}
		// Build list of files to module path source to project vendor folder
		vendorList, err := m.fileCopier.GetMatches(matchOpt.Patterns, module.Dir)
		if err != nil {
			return nil, err
		}
		module.VendorList = vendorList
	}
	return module, nil
}

func (m *goModFactory) copy(modules []*Module) error {
	// Copy mod vendor list files to ./vendor/
	for _, mod := range modules {
		if mod.currentPackage == true {
			for _, vendorFile := range mod.VendorList {
				localPath := strings.TrimPrefix(vendorFile, m.WorkingDirectory+"/")
				localFile := filepath.Join(m.WorkingDirectory, protodep.DefaultDepDir, mod.ImportPath, localPath)
				if _, err := m.fileCopier.Copy(vendorFile, localFile); err != nil {
					return eris.Wrapf(err, fmt.Sprintf("Error! %s - unable to copy file %s\n",
						err.Error(), vendorFile))
				}
			}
		} else {
			for _, vendorFile := range mod.VendorList {
				localPath := filepath.Join(mod.ImportPath, vendorFile[len(mod.Dir):])
				localFile := filepath.Join(m.WorkingDirectory, protodep.DefaultDepDir, localPath)
				if _, err := m.fileCopier.Copy(vendorFile, localFile); err != nil {
					return eris.Wrapf(err, fmt.Sprintf("Error! %s - unable to copy file %s\n",
						err.Error(), vendorFile))
				}
			}
		}
	}
	return nil
}
