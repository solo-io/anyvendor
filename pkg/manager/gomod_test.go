package manager

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/solo-io/protodep/pkg/modutils"
	"github.com/solo-io/protodep/protodep"
)

var _ = Describe("protodep", func() {
	var (
		modPathString string
		mgr           *goModFactory
	)
	BeforeEach(func() {
		modBytes, err := modutils.GetCurrentModPackageFile()
		modFileString := strings.TrimSpace(modBytes)
		Expect(err).NotTo(HaveOccurred())
		modPathString = filepath.Dir(modFileString)
		mgr, err = NewGoModFactory(modPathString)
		Expect(err).NotTo(HaveOccurred())
	})

	Context("helper functions", func() {
		It("can translate a pkgModPath with a !", func() {
			importPath := "github.com/Microsoft/package"
			version := "this_is_a_hash"
			result := pkgModPath(importPath, version)
			resultTest := filepath.Join(os.Getenv("GOPATH"), "pkg", "mod",
				fmt.Sprintf("%s@%s", "github.com/!microsoft/package", version))
			Expect(result).To(Equal(resultTest))
		})
		It("can translate a standard pkgModPath", func() {
			importPath := "github.com/microsoft/package"
			version := "this_is_a_hash"
			result := pkgModPath(importPath, version)
			resultTest := filepath.Join(os.Getenv("GOPATH"), "pkg", "mod",
				fmt.Sprintf("%s@%s", importPath, version))
			Expect(result).To(Equal(resultTest))
		})
	})

	Context("vendor protos", func() {
		It("can vendor protos", func() {
			modules, err := mgr.gather(goModOptions{
				MatchOptions: []*protodep.GoModImport{
					EnvoyValidateProtoMatcher,
				},
				LocalMatchers: []string{"protodep/**/*.proto"},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(modules).To(HaveLen(2))
			Expect(modules[0].ImportPath).To(Equal(EnvoyValidateProtoMatcher.Package))
			Expect(modules[1].ImportPath).To(Equal("github.com/solo-io/protodep"))
			Expect(mgr.copy(modules)).NotTo(HaveOccurred())
		})
	})
})
