package modutils_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestModutils(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Modutils Suite")
}
