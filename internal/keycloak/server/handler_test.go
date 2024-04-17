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

const (
	issuer            = "https://keycloak.example.com/realms/my-org"
	mgmtClientId      = "client-id"
	mgmtClientSecret  = "client-secret"
	fakeAdminEndpoint = "https://keycloak.example.com/admin/realms/my-org"
	resourceServer    = "access"
)

var endpoints = server.DiscoveredEndpoints{
	Tokens:               issuer + "/protocol/openid-connect/token",
	ResourceRegistration: issuer + "/authz/protection/resource_set",
}

var _ = Describe("Server", func() {
	var (
		s   *server.StrictServerHandler
		ctx context.Context
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
	})

	const (
		clientName      = "test-client"
		genClientId     = "created-client-id"
		genClientSecret = "created-client-secret"
	)

	var dummyClient = server.KeycloakClient{
		Id:     genClientId,
		Name:   clientName,
		Secret: genClientSecret,
	}

	Context("Client", func() {
		When("no client exists", func() {

			BeforeEach(func() {
				dummyToken := &server.KeycloakToken{
					AccessToken: "access-token",
				}

				newTokenResponder, _ := httpmock.NewJsonResponder(200, dummyToken)
				httpmock.RegisterResponder("POST", endpoints.Tokens, newTokenResponder)

				newClientResponder, _ := httpmock.NewJsonResponder(200, dummyClient)
				httpmock.RegisterResponder("POST", issuer+"/clients-registrations/default", newClientResponder)

				getClientResponder, _ := httpmock.NewJsonResponder(200, []string{})
				httpmock.RegisterResponder("GET", fakeAdminEndpoint+"/clients?clientId=non-existing-client", getClientResponder)
			})

			It("can create a client", func() {
				resp, err := s.CreateOAuthApplication(ctx, portalv1.CreateOAuthApplicationRequestObject{
					Body: &portalv1.CreateOAuthApplicationJSONRequestBody{
						Name: clientName,
					},
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(resp).To(BeAssignableToTypeOf(portalv1.CreateOAuthApplication201JSONResponse{}))
				resp200 := resp.(portalv1.CreateOAuthApplication201JSONResponse)
				Expect(*resp200.ClientName).To(Equal(clientName))
				Expect(*resp200.ClientId).To(Equal(clientName))
				Expect(*resp200.ClientSecret).To(Equal(genClientSecret))
			})

			It("returns error code on nil body", func() {
				resp, err := s.CreateOAuthApplication(ctx, portalv1.CreateOAuthApplicationRequestObject{})
				Expect(err).NotTo(HaveOccurred())
				Expect(resp).To(BeAssignableToTypeOf(portalv1.CreateOAuthApplication400JSONResponse{}))
			})

			It("returns error code on empty client name", func() {
				resp, err := s.CreateOAuthApplication(ctx, portalv1.CreateOAuthApplicationRequestObject{
					Body: &portalv1.CreateOAuthApplicationJSONRequestBody{
						Name: "",
					},
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(resp).To(BeAssignableToTypeOf(portalv1.CreateOAuthApplication400JSONResponse{}))
			})

			It("returns not found code on deletion", func() {
				resp, err := s.DeleteApplication(ctx, portalv1.DeleteApplicationRequestObject{
					Id: "non-existing-client",
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(resp).To(BeAssignableToTypeOf(portalv1.DeleteApplication404JSONResponse{}))
				resp404 := resp.(portalv1.DeleteApplication404JSONResponse)
				Expect(resp404.Code).To(Equal(400))
			})
		})

		When("client exists", func() {
			BeforeEach(func() {
				dummyToken := &server.KeycloakToken{
					AccessToken: "access-token",
				}

				newTokenResponder, _ := httpmock.NewJsonResponder(200, dummyToken)
				httpmock.RegisterResponder("POST", endpoints.Tokens, newTokenResponder)

				getClientIdResponder, _ := httpmock.NewJsonResponder(200, [1]server.KeycloakClient{dummyClient})
				httpmock.RegisterResponder("GET", fakeAdminEndpoint+"/clients?clientId="+clientName, getClientIdResponder)

				deleteClientResponder, _ := httpmock.NewJsonResponder(204, nil)
				httpmock.RegisterResponder("DELETE", fakeAdminEndpoint+"/clients/"+genClientId, deleteClientResponder)
			})

			It("can delete the client", func() {
				resp, err := s.DeleteApplication(ctx, portalv1.DeleteApplicationRequestObject{
					Id: clientName,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(resp).To(BeAssignableToTypeOf(portalv1.DeleteApplication204Response{}))
			})

		})
	})
})
