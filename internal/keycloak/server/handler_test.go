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

		applicationName         = "test-client"
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
			Name:   applicationName,
			Secret: applicationClientSecret,
		}
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
						Name: applicationName,
						Id:   applicationClientId,
					},
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(resp).To(BeAssignableToTypeOf(portalv1.CreateOAuthApplication201JSONResponse{}))
				resp200 := resp.(portalv1.CreateOAuthApplication201JSONResponse)
				Expect(*resp200.ClientName).To(Equal(applicationName))
				Expect(resp200.ClientId).To(Equal(applicationClientId))
				Expect(resp200.ClientSecret).To(Equal(applicationClientSecret))
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
						Id:   applicationClientId,
					},
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(resp).To(BeAssignableToTypeOf(portalv1.CreateOAuthApplication400JSONResponse{}))
			})

			It("returns error code on empty client id", func() {
				resp, err := s.CreateOAuthApplication(ctx, portalv1.CreateOAuthApplicationRequestObject{
					Body: &portalv1.CreateOAuthApplicationJSONRequestBody{
						Name: applicationName,
						Id:   "",
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
				getClientIdResponder, _ := httpmock.NewJsonResponder(200, [1]server.KeycloakClient{dummyClient})
				httpmock.RegisterResponder("GET", fakeAdminEndpoint+"/clients?clientId="+applicationName, getClientIdResponder)

				deleteClientResponder, _ := httpmock.NewJsonResponder(204, nil)
				httpmock.RegisterResponder("DELETE", fakeAdminEndpoint+"/clients/"+applicationClientId, deleteClientResponder)
			})

			It("can delete the client", func() {
				resp, err := s.DeleteApplication(ctx, portalv1.DeleteApplicationRequestObject{
					Id: applicationName,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(resp).To(BeAssignableToTypeOf(portalv1.DeleteApplication204Response{}))
			})

		})
	})

	Context("API Products", func() {

		apiProduct := "new-API-Product"
		apiProductDescription := "test description"

		When("resource doesn't exist", func() {

			nonExistingApiProduct := "non-existent-API-Product"

			BeforeEach(func() {
				newResourceResponder, _ := httpmock.NewJsonResponder(201, nil)
				httpmock.RegisterResponder("POST", endpoints.ResourceRegistration, newResourceResponder)

				resourceIdLookupResponder, _ := httpmock.NewJsonResponder(200, []string{})
				httpmock.RegisterResponder("GET", endpoints.ResourceRegistration+"?exactName=true&name="+nonExistingApiProduct, resourceIdLookupResponder)
			})

			It("can create an API Product", func() {
				resp, err := s.CreateAPIProduct(ctx, portalv1.CreateAPIProductRequestObject{
					Body: &portalv1.CreateAPIProductJSONRequestBody{
						ApiProduct: portalv1.ApiProduct{
							Name:        apiProduct,
							Description: &apiProductDescription,
						},
					},
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(resp).To(BeAssignableToTypeOf(portalv1.CreateAPIProduct201Response{}))
			})

			It("returns not found if deleting API Product not present", func() {
				resp, err := s.DeleteAPIProduct(ctx, portalv1.DeleteAPIProductRequestObject{
					Name: nonExistingApiProduct,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(resp).To(BeAssignableToTypeOf(portalv1.DeleteAPIProduct404JSONResponse{}))
			})
		})

		When("resource exists", func() {

			resourceId := "3ccd99e8-846b-4b0b-8d75-7ff5ee86cb24"

			BeforeEach(func() {
				newResourceResponder, _ := httpmock.NewJsonResponder(409, nil)
				httpmock.RegisterResponder("POST", endpoints.ResourceRegistration, newResourceResponder)

				resourceIdLookupResponder, _ := httpmock.NewJsonResponder(200, []string{resourceId})
				httpmock.RegisterResponder("GET", endpoints.ResourceRegistration+"?exactName=true&name="+apiProduct, resourceIdLookupResponder)

				deleteResourceResponder, _ := httpmock.NewJsonResponder(204, nil)
				httpmock.RegisterResponder("DELETE", endpoints.ResourceRegistration+"/"+resourceId, deleteResourceResponder)
			})

			It("can delete the APIProduct", func() {
				resp, err := s.DeleteAPIProduct(ctx, portalv1.DeleteAPIProductRequestObject{
					Name: apiProduct,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(resp).To(BeAssignableToTypeOf(portalv1.DeleteAPIProduct204Response{}))
			})

			It("returns that there is a resource conflict", func() {
				resp, err := s.CreateAPIProduct(ctx, portalv1.CreateAPIProductRequestObject{
					Body: &portalv1.CreateAPIProductJSONRequestBody{
						ApiProduct: portalv1.ApiProduct{
							Name:        apiProduct,
							Description: &apiProductDescription,
						},
					},
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(resp).To(BeAssignableToTypeOf(portalv1.CreateAPIProduct409JSONResponse{}))
			})
		})
	})

	Context("Client <-> API Products authorisation", func() {

		apiProducts := []string{"api-product-1", "api-product-2"}
		resource1Id := "resource-1-internal-id"
		resource2Id := "resource-2-internal-id"

		When("client does not exist", func() {

			BeforeEach(func() {
				getClientResponder, _ := httpmock.NewJsonResponder(200, []string{})
				httpmock.RegisterResponder("GET", fakeAdminEndpoint+"/clients?clientId="+applicationName, getClientResponder)
			})

			It("returns not found on update", func() {
				resp, err := s.UpdateAppAPIProducts(ctx, portalv1.UpdateAppAPIProductsRequestObject{
					Id: applicationName,
					Body: &portalv1.UpdateAppAPIProductsJSONRequestBody{
						ApiProducts: apiProducts,
					},
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(resp).To(BeAssignableToTypeOf(portalv1.UpdateAppAPIProducts404JSONResponse{}))
			})
		})

		When("referencing client that does exist", func() {

			existingPermissionId := "existing-permission-id"

			BeforeEach(func() {
				getClientResponder, _ := httpmock.NewJsonResponder(200, [1]server.KeycloakClient{dummyClient})
				httpmock.RegisterResponder("GET", fakeAdminEndpoint+"/clients?clientId="+applicationName, getClientResponder)

				resource1IdLookupResponder, _ := httpmock.NewJsonResponder(200, []string{resource1Id})
				httpmock.RegisterResponder("GET", endpoints.ResourceRegistration+"?exactName=true&name=api-product-1", resource1IdLookupResponder)

				resource2IdLookupResponder, _ := httpmock.NewJsonResponder(200, []string{resource2Id})
				httpmock.RegisterResponder("GET", endpoints.ResourceRegistration+"?exactName=true&name=api-product-2", resource2IdLookupResponder)

				getPermissionResponder, _ := httpmock.NewJsonResponder(200, []server.Permission{{
					Id:      existingPermissionId,
					Clients: []string{applicationName},
				}})
				httpmock.RegisterResponder("GET", endpoints.Policy, getPermissionResponder)

				deletePermissionResponder, _ := httpmock.NewJsonResponder(204, nil)
				httpmock.RegisterResponder("DELETE", endpoints.Policy+"/"+existingPermissionId, deletePermissionResponder)

				newPermission1Responder, _ := httpmock.NewJsonResponder(200, nil)
				httpmock.RegisterResponder("POST", endpoints.Policy+"/"+resource1Id, newPermission1Responder)

				newPermission2Responder, _ := httpmock.NewJsonResponder(200, nil)
				httpmock.RegisterResponder("POST", endpoints.Policy+"/"+resource2Id, newPermission2Responder)
			})

			It("can update client API Products", func() {
				resp, err := s.UpdateAppAPIProducts(ctx, portalv1.UpdateAppAPIProductsRequestObject{
					Id: applicationName,
					Body: &portalv1.UpdateAppAPIProductsJSONRequestBody{
						ApiProducts: apiProducts,
					},
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(resp).To(BeAssignableToTypeOf(portalv1.UpdateAppAPIProducts204Response{}))

				info := httpmock.GetCallCountInfo()
				Expect(info["DELETE "+endpoints.Policy+"/"+existingPermissionId]).To(Equal(1))
				Expect(info["POST "+endpoints.Policy+"/"+resource1Id]).To(Equal(1))
				Expect(info["POST "+endpoints.Policy+"/"+resource2Id]).To(Equal(1))
			})
		})
	})
})
