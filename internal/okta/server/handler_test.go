package server_test

import (
	"context"

	resty "github.com/go-resty/resty/v2"
	"github.com/jarcoal/httpmock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/solo-io/gloo-portal-idp-connect/internal/okta/server"
	portalv1 "github.com/solo-io/gloo-portal-idp-connect/pkg/api/v1"
)

var _ = Describe("Server", func() {

	const (
		oktaDomain = "https://dev-123456.okta.com"
		apiToken   = "test-api-token"

		applicationClientId     = "test-client-id"
		applicationClientSecret = "test-client-secret"
		applicationId           = "0oa1234567890abcdef"
	)

	var (
		s   *server.StrictServerHandler
		ctx context.Context

		dummyApp = server.OktaApplication{
			Id:         applicationId,
			Name:       "oidc_client",
			Label:      applicationClientId,
			SignOnMode: "OPENID_CONNECT",
			Credentials: &server.OktaCredentials{
				OAuthClient: &server.OktaOAuthClient{
					ClientId:     applicationClientId,
					ClientSecret: applicationClientSecret,
				},
			},
		}
		testToken = "test"
	)

	BeforeEach(func() {
		ctx = context.Background()

		var restyClient = resty.New()
		httpmock.ActivateNonDefault(restyClient.GetClient())

		s = server.NewStrictServerHandler(&server.Options{
			OktaDomain: oktaDomain,
			APIToken:   apiToken,
		}, restyClient)
	})

	Context("Application", func() {

		When("no client exists", func() {

			BeforeEach(func() {
				newClientResponder, _ := httpmock.NewJsonResponder(200, dummyApp)
				httpmock.RegisterResponder("POST", oktaDomain+"/api/v1/apps", newClientResponder)

				getClientResponder, _ := httpmock.NewJsonResponder(200, []server.OktaApplication{})
				httpmock.RegisterResponder("GET", oktaDomain+"/api/v1/apps?q=non-existing-client", getClientResponder)
			})

			It("can create a client", func() {
				resp, err := s.CreateOAuthApplication(ctx, portalv1.CreateOAuthApplicationRequestObject{
					Body: &portalv1.CreateOAuthApplicationJSONRequestBody{
						Id: applicationClientId,
					},
					Params: portalv1.CreateOAuthApplicationParams{
						Token: &testToken,
					},
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(resp).To(BeAssignableToTypeOf(portalv1.CreateOAuthApplication201JSONResponse{}))
				resp200 := resp.(portalv1.CreateOAuthApplication201JSONResponse)
				Expect(*resp200.ClientName).To(Equal(applicationClientId))
				Expect(resp200.ClientId).To(Equal(applicationClientId))
				Expect(resp200.ClientSecret).To(Equal(applicationClientSecret))
			})

			It("returns error code on nil body", func() {
				resp, err := s.CreateOAuthApplication(ctx, portalv1.CreateOAuthApplicationRequestObject{})
				Expect(err).NotTo(HaveOccurred())
				Expect(resp).To(BeAssignableToTypeOf(portalv1.CreateOAuthApplication400JSONResponse{}))
			})

			It("returns error code on empty client id", func() {
				resp, err := s.CreateOAuthApplication(ctx, portalv1.CreateOAuthApplicationRequestObject{
					Body: &portalv1.CreateOAuthApplicationJSONRequestBody{
						Id: "",
					},
					Params: portalv1.CreateOAuthApplicationParams{
						Token: &testToken,
					},
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(resp).To(BeAssignableToTypeOf(portalv1.CreateOAuthApplication400JSONResponse{}))
			})

			It("returns not found code on deletion", func() {
				resp, err := s.DeleteOAuthApplication(ctx, portalv1.DeleteOAuthApplicationRequestObject{
					Id: "non-existing-client",
					Params: portalv1.DeleteOAuthApplicationParams{
						Token: &testToken,
					},
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(resp).To(BeAssignableToTypeOf(portalv1.DeleteOAuthApplication404JSONResponse{}))
				resp404 := resp.(portalv1.DeleteOAuthApplication404JSONResponse)
				Expect(resp404.Code).To(Equal(404))
			})
		})

		When("client exists", func() {
			BeforeEach(func() {
				getClientResponder, _ := httpmock.NewJsonResponder(200, []server.OktaApplication{dummyApp})
				httpmock.RegisterResponder("GET", oktaDomain+"/api/v1/apps?q="+applicationClientId, getClientResponder)

				deleteClientResponder, _ := httpmock.NewJsonResponder(204, nil)
				httpmock.RegisterResponder("DELETE", oktaDomain+"/api/v1/apps/"+applicationId, deleteClientResponder)
			})

			It("can delete the client", func() {
				resp, err := s.DeleteOAuthApplication(ctx, portalv1.DeleteOAuthApplicationRequestObject{
					Id: applicationClientId,
					Params: portalv1.DeleteOAuthApplicationParams{
						Token: &testToken,
					},
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(resp).To(BeAssignableToTypeOf(portalv1.DeleteOAuthApplication204Response{}))
			})
		})
	})
})
