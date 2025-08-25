package e2e_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	utils_test "github.com/solo-io/gloo-portal-idp-connect/test/utils"
)

var env *utils_test.KubeContext

func TestE2e(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "IDP Connect E2e Suite")
}

var _ = BeforeSuite(func() {
	var err error
	env, err = utils_test.NewKubeContext("kind-kind")
	Expect(err).NotTo(HaveOccurred())

	env.CheckPodsInCluster(context.Background())
})
