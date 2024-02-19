package e2e_test

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "github.com/solo-io/gloo-portal-idp-connect/pkg/api/v1"
	"github.com/solo-io/gloo-portal-idp-connect/test"
)

var _ = Describe("E2e", Ordered, func() {
	// clientId is the ID of the client we will create
	var (
		clientId string
		// testApiProduct is the name of the scope we will create.
		testApiProduct string
		// clientName is the name of the client we will create.
		clientName string
	)

	BeforeAll(func() {
		nowString := strings.Replace(time.Now().Format(time.RFC3339), ":", "-", -1)
		// Unique scope string to make sure there are no conflicts between runs of e2e tests and so that,
		// should things go wrong, we can identify when the scope was created.
		testApiProduct = fmt.Sprintf("e2e-test-api-product-%s", nowString)
		// Unique client name dated with current time so that we can get a gauge on the clients we are creating when,
		// and to avoid conflicts.
		clientName = fmt.Sprintf("e2e-test-client-%s", nowString)
	})

	It("can can create client", func() {
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

	It("can create API Products", func() {
		curlFromPod := &test.CurlFromPod{
			Url:     "idp-connect/api-products",
			Cluster: env,
			Method:  "POST",
			Data:    fmt.Sprintf(`{"apiProduct": {"name": "%s", "description": "e2e test API Product"}}`, testApiProduct),
			Verbose: true,
			App:     "curl",
			Headers: []string{"Content-Type: application/json"},
		}

		out, err := curlFromPod.Execute()
		Expect(err).NotTo(HaveOccurred())
		Expect(out).To(ContainSubstring("201 Created"))
	})

	It("can add API Products to client", func() {
		curlFromPod := &test.CurlFromPod{
			Url:     "idp-connect/clients/" + clientId + "/api-products",
			Cluster: env,
			Method:  "PUT",
			Data:    fmt.Sprintf(`{"apiProducts":["%s"]}`, testApiProduct),
			Verbose: true,
			App:     "curl",
			Headers: []string{"Content-Type: application/json"},
		}

		out, err := curlFromPod.Execute()
		Expect(err).NotTo(HaveOccurred())
		Expect(out).To(ContainSubstring("204 No Content"))
	})

	It("can remove API Products from client", func() {
		curlFromPod := &test.CurlFromPod{
			Url:     "idp-connect/clients/" + clientId + "/api-products",
			Cluster: env,
			Method:  "PUT",
			Data:    `{"apiProducts":[]}`,
			Verbose: true,
			App:     "curl",
			Headers: []string{"Content-Type: application/json"},
		}

		out, err := curlFromPod.Execute()
		Expect(err).NotTo(HaveOccurred())
		Expect(out).To(ContainSubstring("204 No Content"))
	})

	It("can delete client", func() {
		curlFromPod := &test.CurlFromPod{
			Url:     "idp-connect/clients/" + clientId,
			Cluster: env,
			Method:  "DELETE",
			Verbose: true,
			App:     "curl",
		}

		out, err := curlFromPod.Execute()
		Expect(err).NotTo(HaveOccurred())
		Expect(out).To(ContainSubstring("204 No Content"))
	})

	It("can delete API Product", func() {
		curlFromPod := &test.CurlFromPod{
			Url:     fmt.Sprintf("idp-connect/api-products/%s", url.QueryEscape(testApiProduct)),
			Cluster: env,
			Method:  "DELETE",
			Verbose: true,
			App:     "curl",
		}

		out, err := curlFromPod.Execute()
		Expect(err).NotTo(HaveOccurred())
		Expect(out).To(ContainSubstring("204 No Content"))
	})
})
