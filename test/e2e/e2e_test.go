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

// Note: This test uses Cognito as the identity provider. Cognito's ClientId is generated by cognito, so we store it in the clientId variable and use that in subsequent tests.
// If these were in Keycloak, because the ClientID is provided by the user, we would not need to store it in a variable and could simply use the 'internalClientId' variable instead.
var _ = Describe("E2e", Ordered, func() {
	var (
		// clientId is the ID of the client we create
		clientId string
		// testApiProduct is the name of the scope we will create.
		testApiProduct string
		// internalClientId is the user-defined unique Id. In Portal's case, it will be the ID of the oauth credential table entry.
		internalClientId string
	)

	BeforeAll(func() {
		nowString := strings.Replace(time.Now().Format(time.RFC3339), ":", "-", -1)
		// Unique scope string to make sure there are no conflicts between runs of e2e tests and so that,
		// should things go wrong, we can identify when the scope was created.
		testApiProduct = fmt.Sprintf("e2e-test-api-product-%s", nowString)
		// Unique ID dated with current time so that we can get a gauge on the client we are creating when,
		// and to avoid conflicts.
		internalClientId = fmt.Sprintf("id-%s", nowString)
	})

	It("can can create client", func() {
		curlFromPod := &test.CurlFromPod{
			Url:     "idp-connect/applications/oauth2",
			Cluster: env,
			Method:  "POST",
			Data:    fmt.Sprintf(`{"id": "%s"}`, internalClientId),
			App:     "curl",
			Headers: []string{"Content-Type: application/json"},
		}

		out, err := curlFromPod.Execute()
		Expect(err).NotTo(HaveOccurred())

		var createObj v1.CreateOAuthApplication201JSONResponse
		// If the response was made correctly, we should be able to unmarshal it
		Expect(json.Unmarshal([]byte(out), &createObj)).To(Succeed())
		Expect(createObj.ClientName).ToNot(BeNil())
		Expect(*createObj.ClientName).To(Equal(internalClientId))
		Expect(createObj.ClientId).ToNot(BeNil())
		clientId = createObj.ClientId
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
			Url:     "idp-connect/applications/" + clientId + "/api-products",
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
			Url:     "idp-connect/applications/" + clientId + "/api-products",
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
			Url:     "idp-connect/applications/" + clientId,
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
