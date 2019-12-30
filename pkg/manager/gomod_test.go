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

		EnvoyValidateProtoMatcher = &protodep.GoModImport{
			Package:  "github.com/envoyproxy/protoc-gen-validate",
			Patterns: []string{"validate/*.proto"},
		}
	)
	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
	})
	AfterEach(func() {
		ctrl.Finish()
	})

	Context("helper functions", func() {
		FContext("handleSingleModule", func() {
			type testCase struct {
				nonSplitImport string
				splitImport    []string
				err            error
				setupMocks     func(mockFs *mock_manager.MockFs, mockCp *mock_manager.MockFileCopier)
			}
			var (
				mockFs  *mock_manager.MockFs
				mockCp  *mock_manager.MockFileCopier
				fakeErr = eris.New("test error")

				standardImport = "helm.sh/helm/v3 v3.0.0"
				// replacedImport = "k8s.io/api v0.0.0-20191121015604-11707872ac1c => k8s.io/api v0.0.0-20191004120104-195af9ec3521"
				// localImport    = "k8s.io/api v0.0.0-20191121015604-11707872ac1c => /path/to/local"

				// testCases = []testCase{
				// 	{
				// 		nonSplitImport: standardImport,
				// 		splitImport:    strings.Split(standardImport, " "),
				// 		err:            fakeErr,
				// 		setupMocks: func(mockFs *mock_manager.MockFs, mockCp *mock_manager.MockFileCopier) {
				// 			mockCp.EXPECT().PkgModPath()
				// 			mockFs.EXPECT().Stat()
				// 		},
				// 	},
				// 	{
				// 		nonSplitImport: standardImport,
				// 		splitImport:    strings.Split(standardImport, " "),
				// 		err:            nil,
				// 	},
				// 	{
				// 		nonSplitImport: replacedImport,
				// 		splitImport:    strings.Split(replacedImport, " "),
				// 		err:            nil,
				// 	},
				// 	{
				// 		nonSplitImport: localImport,
				// 		splitImport:    strings.Split(localImport, " "),
				// 		err:            nil,
				// 	},
				// }
			)
			BeforeEach(func() {
				mockCp = mock_manager.NewMockFileCopier(ctrl)
				mockFs = mock_manager.NewMockFs(ctrl)
				mgr = &goModFactory{
					fileCopier: mockCp,
					fs:         mockFs,
				}
			})
			Context("basic import", func() {
				It("will error if dir does not exist", func() {
					fakeDir := "fake/dir"
					splitStandard := strings.Split(standardImport, " ")
					mockCp.EXPECT().PkgModPath(splitStandard[0], splitStandard[1]).Return(fakeDir)
					mockFs.EXPECT().Stat(fakeDir).Return(nil, os.ErrNotExist)
					_, err := mgr.handleSingleModule(splitStandard, nil)
					Expect(err).To(HaveOccurred())
					Expect(eris.Cause(err).Error()).To(Equal(os.ErrNotExist.Error()))
				})
				It("nil match opts, will error if get matches fails", func() {
					fakeDir := "fake/dir"
					splitStandard := strings.Split(standardImport, " ")
					mockCp.EXPECT().PkgModPath(splitStandard[0], splitStandard[1]).Return(fakeDir)
					mockFs.EXPECT().Stat(fakeDir).Return(nil, nil)
					mockCp.EXPECT().GetMatches(gomock.Any(), gomock.Any()).Return(nil, fakeErr)
					_, err := mgr.handleSingleModule(splitStandard, nil)
					Expect(err).To(HaveOccurred())
					Expect(eris.Cause(err)).To(Equal(fakeErr))
				})
				It("nil match opts", func() {
					fakeDir := "fake/dir"
					splitStandard := strings.Split(standardImport, " ")
					vendorList := []string{"vendorLIst"}
					mockCp.EXPECT().PkgModPath(splitStandard[0], splitStandard[1]).Return(fakeDir)
					mockFs.EXPECT().Stat(fakeDir).Return(nil, nil)
					mockCp.EXPECT().GetMatches(DefaultMatchPatterns, fakeDir).Return(vendorList, nil)
					mod, err := mgr.handleSingleModule(splitStandard, nil)
					Expect(err).NotTo(HaveOccurred())
					Expect(mod.VendorList).To(Equal(vendorList))
					Expect(mod.Dir).To(Equal(fakeDir))
				})
				It("real match opts, will error if get matches fails", func() {
					matchOptions := []*protodep.GoModImport{
						EnvoyValidateProtoMatcher,
					}
					fakeDir := "fake/dir"
					splitStandard := strings.Split(standardImport, " ")
					mockCp.EXPECT().PkgModPath(splitStandard[0], splitStandard[1]).Return(fakeDir)
					mockFs.EXPECT().Stat(fakeDir).Return(nil, os.ErrNotExist)
					mockCp.EXPECT().GetMatches(gomock.Any(), gomock.Any()).Return(nil, fakeErr)
					_, err := mgr.handleSingleModule(splitStandard, matchOptions)
					Expect(err).To(HaveOccurred())
					Expect(eris.Cause(err)).To(Equal(fakeErr))
				})
				It("real match opts", func() {
					matchOptions := []*protodep.GoModImport{
						EnvoyValidateProtoMatcher,
					}
					fakeDir := "fake/dir"
					splitStandard := strings.Split(standardImport, " ")
					vendorList := []string{"vendorLIst"}
					mockCp.EXPECT().PkgModPath(splitStandard[0], splitStandard[1]).Return(fakeDir)
					mockFs.EXPECT().Stat(fakeDir).Return(nil, os.ErrNotExist)
					mockCp.EXPECT().GetMatches(matchOptions, fakeDir).Return(vendorList, nil)
					mod, err := mgr.handleSingleModule(splitStandard, matchOptions)
					Expect(err).NotTo(HaveOccurred())
					Expect(mod.VendorList).To(Equal(vendorList))
					Expect(mod.Dir).To(Equal(fakeDir))
				})
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
					fileCopier: mockCp,
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
