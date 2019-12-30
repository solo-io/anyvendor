package manager

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/rotisserie/eris"
	mock_manager "github.com/solo-io/protodep/pkg/manager/mocks"
	"github.com/solo-io/protodep/pkg/modutils"
	"github.com/solo-io/protodep/protodep"
)

var _ = Describe("protodep", func() {
	var (
		modPathString string
		mgr           *goModFactory
		ctrl          *gomock.Controller
	)
	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
	})
	AfterEach(func() {
		ctrl.Finish()
	})

	Context("helper functions", func() {
		Context("pkgModPath", func() {
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
		Context("copy", func() {
			type testCase struct {
				mod        *Module
				vendorFile string
				localFile  string
				err        error
			}
			var (
				mockCp *mock_manager.MockFileCopier
			)
			BeforeEach(func() {
				mockCp = mock_manager.NewMockFileCopier(ctrl)
				mgr = &goModFactory{
					cp: mockCp,
				}
			})

			It("can handle errors from the copier (normal)", func() {
				fakeErr := eris.New("test error")
				testCases := []testCase{
					{
						mod: &Module{
							VendorList: []string{"test"},
						},
						err: fakeErr,
					},
					{
						mod: &Module{
							VendorList:     []string{"test"},
							currentPackage: true,
						},
						err: fakeErr,
					},
				}
				// input isn't important here, just checking for error state
				mockCp.EXPECT().Copy(gomock.Any(), gomock.Any()).Times(2).Return(int64(0), fakeErr)
				for _, v := range testCases {
					err := mgr.copy([]*Module{v.mod})
					Expect(eris.Cause(err)).To(Equal(fakeErr))
				}
			})
			It("can handle a single local module", func() {
				testDir := "/fake/test/dir"
				importPath := "/import/path"
				tc := &testCase{
					mod: &Module{
						ImportPath:     importPath,
						Dir:            testDir,
						VendorList:     []string{filepath.Join(testDir, "package", "1", "hello.proto")},
						currentPackage: true,
					},
					vendorFile: "/fake/test/dir/package/1/hello.proto",
					localFile:  "/fake/test/dir/vendor_proto/import/path/package/1/hello.proto",
				}
				mgr.WorkingDirectory = testDir
				mockCp.EXPECT().Copy(tc.vendorFile, tc.localFile).Return(int64(0), nil)
				Expect(mgr.copy([]*Module{tc.mod})).NotTo(HaveOccurred())

			})
			Context("multiple standard", func() {
				var (
					testDir    = "/fake/test/dir"
					importPath = "/import/path"
					testCases  = []testCase{
						{
							mod: &Module{
								ImportPath: importPath,
								Dir:        testDir,
								VendorList: []string{filepath.Join(testDir, "package", "1", "hello.proto")},
							},
							vendorFile: "/fake/test/dir/package/1/hello.proto",
							localFile:  "vendor_proto/import/path/package/1/hello.proto",
						},
						{
							mod: &Module{
								ImportPath: importPath,
								Dir:        testDir,
								VendorList: []string{filepath.Join(testDir, "package", "2", "hello.proto")},
							},
							vendorFile: "/fake/test/dir/package/2/hello.proto",
							localFile:  "vendor_proto/import/path/package/2/hello.proto",
						},
					}
				)

				for i, v := range testCases {
					It(fmt.Sprintf("testcase %d", i), func() {
						mockCp.EXPECT().Copy(v.vendorFile, v.localFile).Return(int64(0), nil)
						Expect(mgr.copy([]*Module{v.mod})).NotTo(HaveOccurred())
					})
				}
			})

		})
	})

	Context("vendor protos", func() {
		BeforeEach(func() {
			modBytes, err := modutils.GetCurrentModPackageFile()
			modFileString := strings.TrimSpace(modBytes)
			Expect(err).NotTo(HaveOccurred())
			modPathString = filepath.Dir(modFileString)
			mgr, err = NewGoModFactory(modPathString)
			Expect(err).NotTo(HaveOccurred())
		})
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
