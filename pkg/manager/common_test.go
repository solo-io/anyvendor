package manager

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/rotisserie/eris"
	mock_manager "github.com/solo-io/anyvendor/pkg/manager/mocks"
	"github.com/spf13/afero"
)

//go:generate mockgen -package mock_manager -destination ./mocks/afero.go github.com/spf13/afero Fs,File
//go:generate mockgen -package mock_manager -destination ./mocks/fileinfo.go os FileInfo
//go:generate mockgen -package mock_manager -destination ./mocks/copier.go -source ./common.go

var _ = Describe("common", func() {
	var (
		ctrl *gomock.Controller
		cp   *copier
	)
	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
	})
	AfterEach(func() {
		ctrl.Finish()
	})
	Context("pkgModPath", func() {
		BeforeEach(func() {
			cp = &copier{}
		})
		It("can translate a pkgModPath with a !", func() {
			importPath := "github.com/Microsoft/package"
			version := "this_is_a_hash"
			result := cp.PkgModPath(importPath, version)
			resultTest := filepath.Join(os.Getenv("GOPATH"), "pkg", "mod",
				fmt.Sprintf("%s@%s", "github.com/!microsoft/package", version))
			Expect(result).To(Equal(resultTest))
		})
		It("can translate a standard pkgModPath", func() {
			importPath := "github.com/microsoft/package"
			version := "this_is_a_hash"
			result := cp.PkgModPath(importPath, version)
			resultTest := filepath.Join(os.Getenv("GOPATH"), "pkg", "mod",
				fmt.Sprintf("%s@%s", importPath, version))
			Expect(result).To(Equal(resultTest))
		})
	})
	Context("copier", func() {
		Context("mocks", func() {
			var (
				mockFs       *mock_manager.MockFs
				mockFileInfo *mock_manager.MockFileInfo
				mockFile     *mock_manager.MockFile
			)
			BeforeEach(func() {
				ctrl = gomock.NewController(GinkgoT())
				mockFs = mock_manager.NewMockFs(ctrl)
				mockFileInfo = mock_manager.NewMockFileInfo(ctrl)
				mockFile = mock_manager.NewMockFile(ctrl)
				cp = NewCopier(mockFs)
			})
			It("will return error if mkdir fails", func() {
				src, dst := "src/src.go", "dst/dstgo."
				fakeErr := eris.New("hello")
				mockFs.EXPECT().MkdirAll(filepath.Dir(dst), os.ModePerm).Return(fakeErr)
				_, err := cp.Copy(src, dst)
				Expect(err).To(HaveOccurred())
				Expect(err).To(Equal(fakeErr))
			})
			It("will return error if Stat fails", func() {
				src, dst := "src/src.go", "dst/dst.go"
				fakeErr := eris.New("hello")
				mockFs.EXPECT().MkdirAll(filepath.Dir(dst), os.ModePerm).Return(nil)
				mockFs.EXPECT().Stat(src).Return(nil, fakeErr)
				_, err := cp.Copy(src, dst)
				Expect(err).To(HaveOccurred())
				Expect(err).To(Equal(fakeErr))
			})
			It("will return error if fileinfo returns error fails", func() {
				src, dst := "src/src.go", "dst/dst.go"
				mockFs.EXPECT().MkdirAll(filepath.Dir(dst), os.ModePerm).Return(nil)
				mockFs.EXPECT().Stat(src).Return(mockFileInfo, nil)
				mockFileInfo.EXPECT().Mode().Return(os.ModeIrregular)
				_, err := cp.Copy(src, dst)
				Expect(err).To(HaveOccurred())
				isErr := eris.Is(err, IrregularFileError(src))
				Expect(isErr).To(BeTrue())
			})
			It("will return error if open fails", func() {
				src, dst := "src/src.go", "dst/dst.go"
				mockFs.EXPECT().MkdirAll(filepath.Dir(dst), os.ModePerm).Return(nil)
				mockFs.EXPECT().Stat(src).Return(mockFileInfo, nil)
				mockFileInfo.EXPECT().Mode().Return(os.ModePerm)
				fakeErr := eris.New("hello")
				mockFs.EXPECT().Open(src).Return(nil, fakeErr)
				_, err := cp.Copy(src, dst)
				Expect(err).To(HaveOccurred())
				Expect(err).To(Equal(fakeErr))
			})
			It("will return error if create fails", func() {
				src, dst := "src/src.go", "dst/dst.go"
				mockFs.EXPECT().MkdirAll(filepath.Dir(dst), os.ModePerm).Return(nil)
				mockFs.EXPECT().Stat(src).Return(mockFileInfo, nil)
				mockFileInfo.EXPECT().Mode().Return(os.ModePerm)
				fakeErr := eris.New("hello")
				mockFs.EXPECT().Open(src).Return(mockFile, nil)
				mockFile.EXPECT().Close().Return(nil)
				mockFs.EXPECT().Create(dst).Return(nil, fakeErr)
				_, err := cp.Copy(src, dst)
				Expect(err).To(HaveOccurred())
				Expect(err).To(Equal(fakeErr))
			})
		})
		Context("real copy", func() {
			It("Can copy", func() {
				fs := afero.NewOsFs()
				cp := &copier{fs: fs}
				tmpFile, err := afero.TempFile(fs, "", "")
				Expect(err).NotTo(HaveOccurred())
				defer fs.Remove(tmpFile.Name())
				tmpDir, err := afero.TempDir(fs, "", "")
				Expect(err).NotTo(HaveOccurred())
				defer fs.Remove(tmpDir)
				dstFileName := "test"
				_, err = cp.Copy(tmpFile.Name(), filepath.Join(tmpDir, dstFileName))
				Expect(err).NotTo(HaveOccurred())
				_, err = fs.Stat(filepath.Join(tmpDir, dstFileName))
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
})
