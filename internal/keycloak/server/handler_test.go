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
				httpmock.RegisterResponder("GET", fakeAdminEndpoint+"/clients?clientId="+applicationClientId, getClientIdResponder)

				deleteClientResponder, _ := httpmock.NewJsonResponder(204, nil)
				httpmock.RegisterResponder("DELETE", fakeAdminEndpoint+"/clients/"+applicationClientId, deleteClientResponder)
			})

			It("can delete the client", func() {
				resp, err := s.DeleteApplication(ctx, portalv1.DeleteApplicationRequestObject{
					Id: applicationClientId,
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

				getResourcesResponder, _ := httpmock.NewJsonResponder(200, []string{})
				httpmock.RegisterResponder("GET", endpoints.ResourceRegistration, getResourcesResponder)

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

			It("returns an empty list if no API Products are found", func() {
				resp, err := s.GetAPIProducts(ctx, portalv1.GetAPIProductsRequestObject{})
				Expect(err).NotTo(HaveOccurred())
				Expect(resp).To(BeAssignableToTypeOf(portalv1.GetAPIProducts200JSONResponse{}))
				resp200 := resp.(portalv1.GetAPIProducts200JSONResponse)
				Expect(resp200).To(BeEmpty())
			})
		})

		When("resource exists", func() {

			resourceId := "3ccd99e8-846b-4b0b-8d75-7ff5ee86cb24"

			BeforeEach(func() {
				newResourceResponder, _ := httpmock.NewJsonResponder(409, nil)
				httpmock.RegisterResponder("POST", endpoints.ResourceRegistration, newResourceResponder)

				getResourcesResponder, _ := httpmock.NewJsonResponder(200, [1]string{resourceId})
				httpmock.RegisterResponder("GET", endpoints.ResourceRegistration, getResourcesResponder)

				getResourceResponser, _ := httpmock.NewJsonResponder(200, map[string]any{"name": apiProduct})
				httpmock.RegisterResponder("GET", endpoints.ResourceRegistration+"/"+resourceId, getResourceResponser)

				resourceIdLookupResponder, _ := httpmock.NewJsonResponder(200, []string{resourceId})
				httpmock.RegisterResponder("GET", endpoints.ResourceRegistration+"?exactName=true&name="+apiProduct, resourceIdLookupResponder)

				deleteResourceResponder, _ := httpmock.NewJsonResponder(204, nil)
				httpmock.RegisterResponder("DELETE", endpoints.ResourceRegistration+"/"+resourceId, deleteResourceResponder)
			})

			It("can get API Products", func() {
				resp, err := s.GetAPIProducts(ctx, portalv1.GetAPIProductsRequestObject{})
				Expect(err).NotTo(HaveOccurred())
				Expect(resp).To(BeAssignableToTypeOf(portalv1.GetAPIProducts200JSONResponse{}))
				resp200 := resp.(portalv1.GetAPIProducts200JSONResponse)
				Expect(resp200).To(HaveLen(1))
				Expect(resp200).To(ContainElement(portalv1.ApiProduct{
					Name: apiProduct,
				}))
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

		When("client does not exist", func() {

			BeforeEach(func() {
				getClientResponder, _ := httpmock.NewJsonResponder(200, []string{})
				httpmock.RegisterResponder("GET", fakeAdminEndpoint+"/clients?clientId="+applicationClientId, getClientResponder)
			})

			It("returns not found on update", func() {
				resp, err := s.UpdateAppAPIProducts(ctx, portalv1.UpdateAppAPIProductsRequestObject{
					Id: applicationClientId,
					Body: &portalv1.UpdateAppAPIProductsJSONRequestBody{
						ApiProducts: apiProducts,
					},
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(resp).To(BeAssignableToTypeOf(portalv1.UpdateAppAPIProducts404JSONResponse{}))
			})
		})

		When("referencing client that does exist", func() {

			newResourceName := "new-api-product"
			newResourceId := "new-resource-id"

			existingResourceName := "existing-api-product"
			existingResourceId := "existing-resource-id"
			existingPermissionId := "existing-permission-id"

			BeforeEach(func() {
				getClientResponder, _ := httpmock.NewJsonResponder(200, [1]server.KeycloakClient{dummyClient})
				httpmock.RegisterResponder("GET", fakeAdminEndpoint+"/clients?clientId="+applicationClientId, getClientResponder)

				getPermissionResponder, _ := httpmock.NewJsonResponder(200, []server.Permission{{
					Id:      existingPermissionId,
					Name:    server.PermissionName(applicationClientId, existingResourceName),
					Clients: []string{applicationClientId},
				}})
				httpmock.RegisterResponder("GET", endpoints.Policy, getPermissionResponder)

				newResourceIdLookupResponder, _ := httpmock.NewJsonResponder(200, []string{newResourceId})
				httpmock.RegisterResponder("GET", endpoints.ResourceRegistration+"?exactName=true&name="+newResourceName, newResourceIdLookupResponder)

				existingResourceIdLookupResponder, _ := httpmock.NewJsonResponder(200, []string{existingResourceId})
				httpmock.RegisterResponder("GET", endpoints.ResourceRegistration+"?exactName=true&name="+existingResourceName, existingResourceIdLookupResponder)

				deletePermissionResponder, _ := httpmock.NewJsonResponder(204, nil)
				httpmock.RegisterResponder("DELETE", endpoints.Policy+"/"+existingPermissionId, deletePermissionResponder)

				newPermissionResponder, _ := httpmock.NewJsonResponder(200, nil)
				httpmock.RegisterResponder("POST", endpoints.Policy+"/"+newResourceId, newPermissionResponder)
			})

			It("does not re-create permissions for existing Api Products", func() {
				resp, err := s.UpdateAppAPIProducts(ctx, portalv1.UpdateAppAPIProductsRequestObject{
					Id: applicationClientId,
					Body: &portalv1.UpdateAppAPIProductsJSONRequestBody{
						ApiProducts: []string{existingResourceName},
					},
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(resp).To(BeAssignableToTypeOf(portalv1.UpdateAppAPIProducts204Response{}))

				info := httpmock.GetCallCountInfo()
				Expect(info["POST "+endpoints.Policy+"/"+existingResourceId]).To(Equal(0))
				Expect(info["DELETE "+endpoints.Policy+"/"+existingPermissionId]).To(Equal(0))
			})

			It("adds permissions for new API Products", func() {
				resp, err := s.UpdateAppAPIProducts(ctx, portalv1.UpdateAppAPIProductsRequestObject{
					Id: applicationClientId,
					Body: &portalv1.UpdateAppAPIProductsJSONRequestBody{
						// here we pass the existing resource, whose permissions should not be recreated, and a new one which should
						ApiProducts: []string{existingResourceName, newResourceName},
					},
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(resp).To(BeAssignableToTypeOf(portalv1.UpdateAppAPIProducts204Response{}))

				info := httpmock.GetCallCountInfo()
				Expect(info["POST "+endpoints.Policy+"/"+newResourceId]).To(Equal(1))
				// the existing resource + its permission should not be created nor deleted
				Expect(info["POST "+endpoints.Policy+"/"+existingResourceId]).To(Equal(0))
				Expect(info["DELETE "+endpoints.Policy+"/"+existingPermissionId]).To(Equal(0))
			})

			It("deletes permissions for removed API Products", func() {
				resp, err := s.UpdateAppAPIProducts(ctx, portalv1.UpdateAppAPIProductsRequestObject{
					Id: applicationClientId,
					Body: &portalv1.UpdateAppAPIProductsJSONRequestBody{
						ApiProducts: []string{}, // because we don't pass the existingResourceName but it is an existing permissions, it should be deleted
					},
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(resp).To(BeAssignableToTypeOf(portalv1.UpdateAppAPIProducts204Response{}))

				info := httpmock.GetCallCountInfo()
				Expect(info["DELETE "+endpoints.Policy+"/"+existingPermissionId]).To(Equal(1))
			})
		})
	})
})