package manager_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestAnyVendor(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "anyvendor Suite")
}
