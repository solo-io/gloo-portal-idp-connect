package e2e_test

import (
	"encoding/json"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "github.com/solo-io/gloo-portal-idp-connect/pkg/api/v1"
	"github.com/solo-io/gloo-portal-idp-connect/test"
)

var _ = Describe("E2e", Ordered, func() {
	// clientId is the ID of the client we will create
	var clientId string

	It("can can create client", func() {
		clientName := "e2e-test"
		curlFromPod := &test.CurlFromPod{
			Url:     "idp-connect/clients",
			Cluster: env,
			Method:  "POST",
			Data:    fmt.Sprintf(`{"clientName": "%s"}`, clientName),
			App:     "curl",
			Headers: []string{"Content-Type: application/json"},
		}

		out, err := curlFromPod.Execute()
		Expect(err).NotTo(HaveOccurred())

		var createObj v1.CreateClient201JSONResponse
		// If the response was made correctly, we should be able to unmarshal it
		Expect(json.Unmarshal([]byte(out), &createObj)).To(Succeed())
		Expect(createObj.ClientName).ToNot(BeNil())
		Expect(*createObj.ClientName).To(Equal(clientName))
		Expect(createObj.ClientId).ToNot(BeNil())
		clientId = *createObj.ClientId
	})

	It("can delete client", func() {
		curlFromPod := &test.CurlFromPod{
			Url:     "idp-connect/clients/" + clientId,
			Cluster: env,
			Method:  "DELETE",
			App:     "curl",
		}

		out, err := curlFromPod.Execute()
		Expect(err).NotTo(HaveOccurred())
		Expect(out).To(Equal(""))
	})
})
