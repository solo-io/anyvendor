package git_test

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/solo-io/anyvendor/pkg/git"
)

var _ = Describe("VendorGit", func() {
	AfterEach(func() {
		_ = os.RemoveAll("./test_vendor")
	})
	It("vendors from git", func() {
		err := git.VendorOptions{GitRepositories: []git.GitRepository{{
			URL:           "https://github.com/kelseyhightower/nocode",
			SHA:           "6c073b08f7987018cbb2cb9a5747c84913b3608e",
			MatchPatterns: []string{"README.md"},
		}}}.Vendor(git.DefaultCache(), "./test_vendor")
		Expect(err).NotTo(HaveOccurred())

		_, err = os.Stat("./test_vendor/github.com/kelseyhightower/nocode/README.md")
		Expect(err).NotTo(HaveOccurred())

	})
})
