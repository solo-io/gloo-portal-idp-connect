package e2e_test

import (
	"context"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/solo-io/gloo-portal-idp-connect/test"
)

var env *KubeContext

func TestE2e(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "IDP Connect E2e Suite")
}

var _ = BeforeSuite(func() {
	var err error
	env, err = NewKubeContext("kind-kind")
	Expect(err).NotTo(HaveOccurred())

	ctx, _ := context.WithTimeout(context.Background(), 90*time.Second)
	env.CheckPodsInCluster(ctx)
})
