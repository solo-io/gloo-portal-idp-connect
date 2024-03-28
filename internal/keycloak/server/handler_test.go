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
	issuer                   = "https://keycloak.example.com/realms/my-org"
	fakeRegistrationEndpoint = issuer + "/clients-registrations/openid-connect"
	bearerToken              = "fake-token"
	resourceServer           = "access"
)

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
			Issuer:         issuer,
			BearerToken:    bearerToken,
			ResourceServer: resourceServer,
		},
			restyClient,
			fakeRegistrationEndpoint)
	})

	Context("Client", func() {
		When("no client exists", func() {

			clientName := "test-client"
			genClientId := "2r7vpfuuhbimiqq9bmfde1e3t3"
			genClientSecret := "6au6kel0b"

			BeforeEach(func() {
				dummyClient := &server.CreatedClient{
					Id:     genClientId,
					Name:   clientName,
					Secret: genClientSecret,
				}

				responder, _ := httpmock.NewJsonResponder(200, dummyClient)
				httpmock.RegisterResponder("POST", fakeRegistrationEndpoint, responder)
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
				Expect(*resp200.ClientId).To(Equal(genClientId))
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
		})
	})
})
