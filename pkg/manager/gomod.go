package manager

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rotisserie/eris"
	"github.com/solo-io/anyvendor/anyvendor"
	"github.com/solo-io/anyvendor/pkg/modutils"
	"github.com/spf13/afero"
)

var (
	// offer sane defaults for proto vendoring
	DefaultMatchPatterns = []string{anyvendor.ProtoMatchPattern}
)

type goModOptions struct {
	MatchOptions  []*anyvendor.GoModImport
	LocalMatchers []string
}

// struct which represents a go module package in the module package list
type moduleWithImports struct {
	module     *modutils.Module
	vendorList []string // files to vendor
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

func (m *goModFactory) Ensure(ctx context.Context, opts *anyvendor.Config) error {
	var packages []*anyvendor.GoModImport
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
func (m *goModFactory) gather(opts goModOptions) ([]*moduleWithImports, error) {
	// Ensure go.mod file exists and we're running from the project root,
	modPackageFile, err := modutils.GetCurrentModPackageFile()
	if err != nil {
		return nil, err
	}

	packageName, err := modutils.GetCurrentModPackageName(modPackageFile)
	if err != nil {
		return nil, err
	}

	var moduleNames []string
	for _, v := range opts.MatchOptions {
		moduleNames = append(moduleNames, v.Package)
	}
	moduleNames = append(moduleNames, packageName)
	modPackages, err := modutils.GetCurrentPackageListJson(moduleNames)
	if err != nil {
		return nil, err
	}

	// handle local pacakges, should never be length 0
	localImports := []*anyvendor.GoModImport{
		{
			Patterns: opts.LocalMatchers,
			Package:  packageName,
		},
	}

	var modules []*moduleWithImports
	// handle all packages
	for _, modPackage := range modPackages {
		imports := opts.MatchOptions
		if modPackage.Path == packageName {
			imports = localImports
		}
		mod, err := m.handleSingleModule(modPackage, imports)
		if err != nil {
			return nil, err
		}
		if len(mod.vendorList) > 0 {
			modules = append(modules, mod)
		}
	}

	return modules, nil
}

func (m *goModFactory) handleSingleModule(module *modutils.Module, matchOptions []*anyvendor.GoModImport) (*moduleWithImports, error) {
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
		return &moduleWithImports{
			module:     module,
			vendorList: vendorList,
		}, nil
	}

	var result []string
	for _, matchOpt := range matchOptions {
		// only check module if is in imports list, or imports list in empty
		if len(matchOpt.Package) != 0 &&
			!strings.Contains(module.Path, matchOpt.Package) {
			continue
		}
		// Build list of files to module path source to project vendor folder
		vendorList, err := m.fileCopier.GetMatches(matchOpt.Patterns, module.Dir)
		if err != nil {
			return nil, err
		}
		result = vendorList
	}
	return &moduleWithImports{
		module:     module,
		vendorList: result,
	}, nil
}

func (m *goModFactory) copy(modules []*moduleWithImports) error {
	// Copy mod vendor list files to ./vendor/
	for _, mod := range modules {
		if mod.module.Main == true {
			for _, vendorFile := range mod.vendorList {
				localPath := strings.TrimPrefix(vendorFile, m.WorkingDirectory+"/")
				localFile := filepath.Join(m.WorkingDirectory, anyvendor.DefaultDepDir, mod.module.Path, localPath)
				if _, err := m.fileCopier.Copy(vendorFile, localFile); err != nil {
					return eris.Wrapf(err, fmt.Sprintf("Error! %s - unable to copy file %s\n",
						err.Error(), vendorFile))
				}
			}
		} else {
			for _, vendorFile := range mod.vendorList {
				localPath := filepath.Join(mod.module.Path, vendorFile[len(mod.module.Dir):])
				localFile := filepath.Join(m.WorkingDirectory, anyvendor.DefaultDepDir, localPath)
				if _, err := m.fileCopier.Copy(vendorFile, localFile); err != nil {
					return eris.Wrapf(err, fmt.Sprintf("Error! %s - unable to copy file %s\n",
						err.Error(), vendorFile))
				}
			}
		}
	}
	return nil
}
