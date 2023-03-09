package manager

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rotisserie/eris"
	"github.com/solo-io/anyvendor/anyvendor"
	mock_manager "github.com/solo-io/anyvendor/pkg/manager/mocks"
	"github.com/solo-io/anyvendor/pkg/modutils"
)

var _ = Describe("anyvendor", func() {
	var (
		modPathString string
		mgr           *goModFactory
		ctrl          *gomock.Controller

		EnvoyValidateProtoMatcher = &anyvendor.GoModImport{
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
		Context("handleSingleModule", func() {
			type testCase struct {
				nonSplitImport string
				splitImport    []string
				err            error
				setupMocks     func(mockFs *mock_manager.MockFs, mockCp *mock_manager.MockFileCopier)
			}
			var (
				mockFs         *mock_manager.MockFs
				mockCp         *mock_manager.MockFileCopier
				fakeErr        = eris.New("test error")
				fakeDir        = "fake/dir"
				standardModule = &modutils.Module{
					Path:     "github.com/envoyproxy/protoc-gen-validate",
					Version:  "v0.1.0",
					Indirect: false,
					Dir:      fakeDir,
				}
			)
			BeforeEach(func() {
				mockCp = mock_manager.NewMockFileCopier(ctrl)
				mockFs = mock_manager.NewMockFs(ctrl)
				mgr = &goModFactory{
					fileCopier: mockCp,
					fs:         mockFs,
				}
			})
			Context("errors", func() {
				It("will error if dir does not exist", func() {
					mockFs.EXPECT().Stat(fakeDir).Return(nil, os.ErrNotExist)
					_, err := mgr.handleSingleModule(standardModule, nil)
					Expect(err).To(HaveOccurred())
					Expect(eris.Cause(err).Error()).To(Equal(os.ErrNotExist.Error()))
				})
				It("nil match opts, will error if get matches fails", func() {
					mockFs.EXPECT().Stat(fakeDir).Return(nil, nil)
					mockCp.EXPECT().GetMatches(gomock.Any(), gomock.Any()).Return(nil, fakeErr)
					_, err := mgr.handleSingleModule(standardModule, nil)
					Expect(err).To(HaveOccurred())
					Expect(eris.Cause(err)).To(Equal(fakeErr))
				})
				It("real match opts, will error if get matches fails", func() {
					matchOptions := []*anyvendor.GoModImport{
						EnvoyValidateProtoMatcher,
					}
					mockFs.EXPECT().Stat(fakeDir).Return(nil, nil)
					mockCp.EXPECT().GetMatches(gomock.Any(), gomock.Any()).Return(nil, fakeErr)
					_, err := mgr.handleSingleModule(standardModule, matchOptions)
					Expect(err).To(HaveOccurred())
					Expect(eris.Cause(err)).To(Equal(fakeErr))
				})
			})
			Context("basic imports", func() {
				It("nil match opts", func() {
					vendorList := []string{"vendorLIst"}
					mockFs.EXPECT().Stat(fakeDir).Return(nil, nil)
					mockCp.EXPECT().GetMatches(DefaultMatchPatterns, fakeDir).Return(vendorList, nil)
					mod, err := mgr.handleSingleModule(standardModule, nil)
					Expect(err).NotTo(HaveOccurred())
					Expect(mod.vendorList).To(Equal(vendorList))
					Expect(mod.module.Dir).To(Equal(fakeDir))
				})
				It("real match opts", func() {
					matchOptions := []*anyvendor.GoModImport{
						EnvoyValidateProtoMatcher,
					}
					vendorList := []string{"vendorLIst"}
					mockFs.EXPECT().Stat(fakeDir).Return(nil, nil)
					mockCp.EXPECT().GetMatches(matchOptions[0].Patterns, fakeDir).Return(vendorList, nil)
					mod, err := mgr.handleSingleModule(standardModule, matchOptions)
					Expect(err).NotTo(HaveOccurred())
					Expect(mod.vendorList).To(Equal(vendorList))
					Expect(mod.module.Dir).To(Equal(fakeDir))
				})
			})
		})

		Context("copy", func() {
			type testCase struct {
				mod        *moduleWithImports
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
						mod: &moduleWithImports{
							vendorList: []string{"test"},
							module:     &modutils.Module{},
						},
						err: fakeErr,
					},
					{
						mod: &moduleWithImports{
							vendorList: []string{"test"},
							module: &modutils.Module{
								Main: true,
							},
						},
						err: fakeErr,
					},
				}
				// input isn't important here, just checking for error state
				mockCp.EXPECT().Copy(gomock.Any(), gomock.Any()).Times(2).Return(int64(0), fakeErr)
				for _, v := range testCases {
					err := mgr.copy([]*moduleWithImports{v.mod})
					Expect(eris.Cause(err)).To(Equal(fakeErr))
				}
			})
			It("can handle a single local module", func() {
				testDir := "/fake/test/dir"
				importPath := "/import/path"
				tc := &testCase{
					mod: &moduleWithImports{
						module: &modutils.Module{
							Path: importPath,
							Dir:  testDir,
							Main: true,
						},
						vendorList: []string{filepath.Join(testDir, "package", "1", "hello.proto")},
					},
					vendorFile: "/fake/test/dir/package/1/hello.proto",
					localFile:  "/fake/test/dir/vendor_any/import/path/package/1/hello.proto",
				}
				mgr.WorkingDirectory = testDir
				mockCp.EXPECT().Copy(tc.vendorFile, tc.localFile).Return(int64(0), nil)
				Expect(mgr.copy([]*moduleWithImports{tc.mod})).NotTo(HaveOccurred())

			})
			Context("multiple standard", func() {
				var (
					testDir    = "/fake/test/dir"
					importPath = "/import/path"
					testCases  = []testCase{
						{
							mod: &moduleWithImports{
								module: &modutils.Module{
									Path: importPath,
									Dir:  testDir,
								},
								vendorList: []string{filepath.Join(testDir, "package", "1", "hello.proto")},
							},
							vendorFile: "/fake/test/dir/package/1/hello.proto",
							localFile:  "vendor_any/import/path/package/1/hello.proto",
						},
						{
							mod: &moduleWithImports{
								module: &modutils.Module{
									Path: importPath,
									Dir:  testDir,
								},
								vendorList: []string{filepath.Join(testDir, "package", "2", "hello.proto")},
							},
							vendorFile: "/fake/test/dir/package/2/hello.proto",
							localFile:  "vendor_any/import/path/package/2/hello.proto",
						},
					}
				)

				for i, v := range testCases {
					It(fmt.Sprintf("testcase %d", i), func() {
						mockCp.EXPECT().Copy(v.vendorFile, v.localFile).Return(int64(0), nil)
						Expect(mgr.copy([]*moduleWithImports{v.mod})).NotTo(HaveOccurred())
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
			mgr, err = NewGoModFactory(&anyvendor.FactorySettings{
				Cwd: modPathString,
			})
			Expect(err).NotTo(HaveOccurred())
		})
		It("can vendor protos", func() {
			modules, err := mgr.gather(goModOptions{
				MatchOptions: []*anyvendor.GoModImport{
					EnvoyValidateProtoMatcher,
				},
				LocalMatchers: []string{"anyvendor/**/*.proto"},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(modules).To(HaveLen(2))
			Expect(modules[0].module.Path).To(Equal("github.com/solo-io/anyvendor"))
			Expect(modules[1].module.Path).To(Equal(EnvoyValidateProtoMatcher.Package))
			Expect(mgr.copy(modules)).NotTo(HaveOccurred())
		})
	})
})
