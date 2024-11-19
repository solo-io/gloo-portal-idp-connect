package server_test

import (
	"context"
	resty "github.com/go-resty/resty/v2"
	_ "github.com/golang/mock/mockgen/model"
	"github.com/jarcoal/httpmock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/solo-io/gloo-portal-idp-connect/internal/keycloak/server"
	portalv1 "github.com/solo-io/gloo-portal-idp-connect/pkg/api/v1"
)

var _ = Describe("Server", func() {

	const (
		issuer           = "https://keycloak.example.com/realms/my-org"
		mgmtClientId     = "client-id"
		mgmtClientSecret = "client-secret"

		fakeAdminEndpoint = "https://keycloak.example.com/admin/realms/my-org"

		applicationClientId     = "client-internal-id"
		applicationClientSecret = "client-secret"
	)

	var (
		s   *server.StrictServerHandler
		ctx context.Context

		endpoints = server.DiscoveredEndpoints{
			Tokens:               issuer + "/protocol/openid-connect/token",
			ResourceRegistration: issuer + "/authz/protection/resource_set",
		}

		dummyClient = server.KeycloakClient{
			Id:     applicationClientId,
			Name:   applicationClientId,
			Secret: applicationClientSecret,
		}
		testToken = "test"
	)

	BeforeEach(func() {
		ctx = context.Background()

		var restyClient = resty.New()
		httpmock.ActivateNonDefault(restyClient.GetClient())

		s = server.NewStrictServerHandler(&server.Options{
			Issuer:           issuer,
			MgmtClientId:     mgmtClientId,
			MgmtClientSecret: mgmtClientSecret,
		},
			restyClient,
			endpoints)

		dummyToken := &server.KeycloakToken{
			AccessToken: "access-token",
		}

		newTokenResponder, _ := httpmock.NewJsonResponder(200, dummyToken)
		httpmock.RegisterResponder("POST", endpoints.Tokens, newTokenResponder)
	})

	Context("Application", func() {

		When("no client exists", func() {

			BeforeEach(func() {
				newClientResponder, _ := httpmock.NewJsonResponder(200, dummyClient)
				httpmock.RegisterResponder("POST", issuer+"/clients-registrations/default", newClientResponder)

				getClientResponder, _ := httpmock.NewJsonResponder(200, []string{})
				httpmock.RegisterResponder("GET", fakeAdminEndpoint+"/clients?clientId=non-existing-client", getClientResponder)
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
				Expect(resp404.Code).To(Equal(400))
			})
		})

		When("client exists", func() {
			BeforeEach(func() {
				getClientIdResponder, _ := httpmock.NewJsonResponder(200, [1]server.KeycloakClient{dummyClient})
				httpmock.RegisterResponder("GET", fakeAdminEndpoint+"/clients?clientId="+applicationClientId, getClientIdResponder)

				deleteClientResponder, _ := httpmock.NewJsonResponder(204, nil)
				httpmock.RegisterResponder("DELETE", fakeAdminEndpoint+"/clients/"+applicationClientId, deleteClientResponder)
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
