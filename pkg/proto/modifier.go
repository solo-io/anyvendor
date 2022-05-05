package proto

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/mattn/go-zglob"
	"github.com/rotisserie/eris"
)

// ProtoFilePatcher updates vendored proto files with custom patches.
type ProtoFilePatcher struct {
	// specify a function that returns the go_package option the patched proto file will use
	PatchGoPackage func(path string) string

	// specify a function that can manipulate the proto imports
	PatchImports func(protoImport string) string

	// the root of the vendor dir where the proto files will be patched recursively
	RootDir string
	// patch files matching these patterns
	MatchPatterns []string
}

func (p ProtoFilePatcher) PatchProtoFiles() error {
	var filesToPatch []string

	for _, pat := range p.MatchPatterns {
		matches, err := zglob.Glob(filepath.Join(p.RootDir, pat))
		if err != nil {
			return eris.Wrapf(err, "glob match failure")
		}
		for _, match := range matches {
			filesToPatch = append(filesToPatch, match)
		}
	}

	for _, fileToPatch := range filesToPatch {
		var goPackageForFile string
		if p.PatchGoPackage != nil {
			goPackageForFile = p.PatchGoPackage(strings.TrimPrefix(fileToPatch, p.RootDir))
		}
		if err := PatchProtoFile(fileToPatch, goPackageForFile, p.PatchImports); err != nil {
			return err
		}
	}

	return nil
}

func PatchProtoFile(path, goPackage string, patchImports func(string) string) error {
	if goPackage == "" && patchImports == nil {
		return nil
	}
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(b), "\n")

	goPackageLine := `option go_package = "` + goPackage + `";`
	var packageDeclarationLine int
	var replaced bool
	for i, line := range lines {
		switch {
		case strings.HasPrefix(line, "import"):
			lines[i] = replaceImport(line, patchImports)
		case strings.HasPrefix(line, "package"):
			packageDeclarationLine = i
		case strings.HasPrefix(line, "option go_package") && goPackage != "":
			// replace existing go_package
			lines[i] = goPackageLine
			replaced = true
		}
	}
	if !replaced && goPackage != "" {
		// insert after syntax line
		lines = append(lines[:packageDeclarationLine+1], append([]string{"\n" + goPackageLine}, lines[packageDeclarationLine+1:]...)...)
	}
	fileInfo, err := os.Stat(path)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(path, []byte(strings.Join(lines, "\n")), fileInfo.Mode())
}

func replaceImport(line string, importFunc func(string) string) string {
	if importFunc == nil {
		return line
	}

	parts := strings.Split(line, `"`)
	if len(parts) != 3 {
		// Won't happen unless there's a syntax error
		return line
	}

	return fmt.Sprintf(`import "%s";`, importFunc(parts[1]))
}
