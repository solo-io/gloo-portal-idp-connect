package server_test

import (
	"context"
	"errors"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
	cognito "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	smithyhttp "github.com/aws/smithy-go/transport/http"
	"github.com/golang/mock/gomock"
	_ "github.com/golang/mock/mockgen/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/solo-io/gloo-portal-idp-connect/internal/cognito/server"
	"github.com/solo-io/gloo-portal-idp-connect/internal/cognito/server/mock"
	portalv1 "github.com/solo-io/gloo-portal-idp-connect/pkg/api/v1"
)

const (
	userPoolID     = "us-west-2_abc123"
	resourceServer = "access"
)

var _ = Describe("Server", func() {
	var (
		s                 *server.StrictServerHandler
		mockCtrl          *gomock.Controller
		mockCognitoClient *mock_server.MockCognitoClient
		ctx               context.Context
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockCognitoClient = mock_server.NewMockCognitoClient(mockCtrl)
		ctx = context.Background()

		s = server.NewStrictServerHandler(&server.Options{
			CognitoUserPool: userPoolID,
			ResourceServer:  resourceServer,
		}, mockCognitoClient)
	})

	Context("Application", func() {

		It("returns error code on nil body", func() {
			resp, err := s.CreateApplication(ctx, portalv1.CreateApplicationRequestObject{})
			Expect(err).NotTo(HaveOccurred())
			Expect(resp).To(BeAssignableToTypeOf(portalv1.CreateApplication400JSONResponse{}))
		})

		When("no application exists", func() {
			BeforeEach(func() {
				// Return generated client ID and Secret on user-provided client name.
				mockCognitoClient.EXPECT().CreateUserPoolClient(ctx, gomock.Any(), gomock.Any()).Times(0)

				// Return client not found error on describe.
				mockCognitoClient.EXPECT().DeleteUserPoolClient(ctx, gomock.Any(), gomock.Any()).Times(0)
			})

			It("returns not found code on deletion", func() {
				resp, err := s.DeleteApplication(ctx, portalv1.DeleteApplicationRequestObject{
					Name: "test-client",
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(resp).To(BeAssignableToTypeOf(portalv1.DeleteApplication404JSONResponse{}))
				resp404 := resp.(portalv1.DeleteApplication404JSONResponse)
				Expect(resp404.Code).To(Equal(404))
			})

			It("returns not found code on update", func() {
				resp, err := s.UpdateAppAPIProducts(ctx, portalv1.UpdateAppAPIProductsRequestObject{
					Name: "test-client",
					Body: &portalv1.UpdateAppAPIProductsJSONRequestBody{
						ApiProductNames: []string{"test-APIProduct"},
					},
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(resp).To(BeAssignableToTypeOf(portalv1.UpdateAppAPIProducts404JSONResponse{}))
				resp404 := resp.(portalv1.UpdateAppAPIProducts404JSONResponse)
				Expect(resp404.Code).To(Equal(404))
			})

			It("returns not found code on register", func() {
				resp, err := s.RegisterAppOauthClient(ctx, portalv1.RegisterAppOauthClientRequestObject{
					Name: "test-client",
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(resp).To(BeAssignableToTypeOf(portalv1.RegisterAppOauthClient404JSONResponse{}))
				resp404 := resp.(portalv1.RegisterAppOauthClient404JSONResponse)
				Expect(resp404.Code).To(Equal(404))
			})

			It("can create application", func() {
				client := "test-client"
				resp, err := s.CreateApplication(ctx, portalv1.CreateApplicationRequestObject{
					Body: &portalv1.CreateApplicationJSONRequestBody{
						Name: client,
					},
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(resp).To(BeAssignableToTypeOf(portalv1.CreateApplication201JSONResponse{}))
			})
		})

		When("application exists", func() {
			const clientName = "test-client"
			BeforeEach(func() {
				resp, err := s.CreateApplication(ctx, portalv1.CreateApplicationRequestObject{
					Body: &portalv1.CreateApplicationJSONRequestBody{
						Name: clientName,
					},
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(resp).To(BeAssignableToTypeOf(portalv1.CreateApplication201JSONResponse{}))
			})
			When("client does not exist", func() {
				const (
					genClientId     = "2r7vpfuuhbimiqq9bmfde1e3t3"
					genClientSecret = "6au6kel0b"
				)

				BeforeEach(func() {
					// Return generated client ID and Secret on user-provided client name.
					mockCognitoClient.EXPECT().CreateUserPoolClient(ctx, gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(
						func(
							ctx context.Context,
							input *cognito.CreateUserPoolClientInput,
							optFns ...interface{},
						) (*cognito.CreateUserPoolClientOutput, error) {

							client := genClientId
							secret := genClientSecret
							return &cognito.CreateUserPoolClientOutput{
								UserPoolClient: &types.UserPoolClientType{
									ClientId:     &client,
									ClientSecret: &secret,
									ClientName:   input.ClientName,
								},
							}, nil
						})

					// Return client not found error on delete.
					mockCognitoClient.EXPECT().DeleteUserPoolClient(ctx, gomock.Any(), gomock.Any()).Times(0)
				})

				It("can create a client", func() {
					client := "test-client"
					resp, err := s.RegisterAppOauthClient(ctx, portalv1.RegisterAppOauthClientRequestObject{
						Name: client,
					})
					Expect(err).NotTo(HaveOccurred())
					Expect(resp).To(BeAssignableToTypeOf(portalv1.RegisterAppOauthClient201JSONResponse{}))
					resp200 := resp.(portalv1.RegisterAppOauthClient201JSONResponse)
					Expect(*resp200.ClientName).To(Equal(client))
					Expect(resp200.ClientId).NotTo(BeNil())
					Expect(resp200.ClientSecret).NotTo(BeNil())
				})

				It("does not call OIDC deletion on delete", func() {
					resp, err := s.DeleteApplication(ctx, portalv1.DeleteApplicationRequestObject{
						Name: "test-client",
					})
					Expect(err).NotTo(HaveOccurred())
					Expect(resp).To(BeAssignableToTypeOf(portalv1.DeleteApplication204Response{}))
				})
			})

			When("client exists", func() {
				const genClientId = "4q5270uvfj8v86vc8oqfk3f4m9"
				BeforeEach(func() {

					// Create client with new id when same name is given.
					mockCognitoClient.EXPECT().CreateUserPoolClient(ctx, gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(
						func(
							ctx context.Context,
							input *cognito.CreateUserPoolClientInput,
							optFns ...interface{},
						) (*cognito.CreateUserPoolClientOutput, error) {
							client := clientName
							clientId := genClientId
							return &cognito.CreateUserPoolClientOutput{
								UserPoolClient: &types.UserPoolClientType{
									ClientId:   &clientId,
									ClientName: &client,
								},
							}, nil
						})

					resp, err := s.RegisterAppOauthClient(ctx, portalv1.RegisterAppOauthClientRequestObject{
						Name: clientName,
					})
					Expect(err).NotTo(HaveOccurred())
					Expect(resp).To(BeAssignableToTypeOf(portalv1.RegisterAppOauthClient201JSONResponse{}))
				})

				It("deletes the client as well as the application", func() {
					// Delete client on delete
					mockCognitoClient.EXPECT().DeleteUserPoolClient(ctx, gomock.Any(), gomock.Any()).Times(1).Return(
						&cognito.DeleteUserPoolClientOutput{},
						nil,
					)
					resp, err := s.DeleteApplication(ctx, portalv1.DeleteApplicationRequestObject{
						Name: clientName,
					})
					Expect(err).NotTo(HaveOccurred())
					Expect(resp).To(BeAssignableToTypeOf(portalv1.DeleteApplication204Response{}))
				})

				It("cannot generate multiple clients for same application", func() {
					resp, err := s.RegisterAppOauthClient(ctx, portalv1.RegisterAppOauthClientRequestObject{
						Name: clientName,
					})
					Expect(err).NotTo(HaveOccurred())
					Expect(resp).To(BeAssignableToTypeOf(portalv1.RegisterAppOauthClient201JSONResponse{}))
					resp201 := resp.(portalv1.RegisterAppOauthClient201JSONResponse)
					Expect(*resp201.ClientName).To(Equal(clientName))
					Expect(*resp201.ClientId).NotTo(Equal(genClientId))
				})
			})
		})

		Context("APIProducts", func() {
			When("resource server does not exist", func() {
				var resourceServerCreated bool
				BeforeEach(func() {
					mockCognitoClient.EXPECT().DescribeResourceServer(ctx, gomock.Any(), gomock.Any()).AnyTimes().Return(
						nil,
						&smithyhttp.ResponseError{
							Response: &smithyhttp.Response{
								Response: &http.Response{
									StatusCode: 404,
									Status:     "Resource Not Found",
								},
							},
							Err: &types.ResourceNotFoundException{
								Message: aws.String("resource not found"),
							},
						},
					)

					resourceServerCreated = false
					// Expect that we create the resource server EXACTLY once.
					access := resourceServer
					mockCognitoClient.EXPECT().CreateResourceServer(ctx, gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(
						func(
							ctx context.Context,
							input *cognito.CreateResourceServerInput,
							optFns ...func(*cognito.Options),
						) (*cognito.CreateResourceServerOutput, error) {
							if resourceServerCreated {
								return nil, &smithyhttp.ResponseError{
									Response: &smithyhttp.Response{
										Response: &http.Response{
											StatusCode: 409,
											Status:     "Conflict",
										},
									},
									Err: errors.New("resource server already exists"),
								}
							}

							return &cognito.CreateResourceServerOutput{
								ResourceServer: &types.ResourceServerType{
									Identifier: &access,
									Name:       &access,
								},
							}, nil
						})

					// Updating resource server is valid.
					mockCognitoClient.EXPECT().UpdateResourceServer(ctx, gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(
						func(
							ctx context.Context,
							input *cognito.UpdateResourceServerInput,
							optFns ...func(*cognito.Options),
						) (*cognito.UpdateResourceServerOutput, error) {
							return &cognito.UpdateResourceServerOutput{
								ResourceServer: &types.ResourceServerType{
									Identifier: input.Identifier,
									Name:       input.Name,
									Scopes:     input.Scopes,
								},
							}, nil
						})
				})

				It("can create a APIProduct", func() {
					APIProduct := "test-APIProduct"
					resp, err := s.CreateAPIProduct(ctx, portalv1.CreateAPIProductRequestObject{
						Body: &portalv1.CreateAPIProductJSONRequestBody{
							ApiProduct: portalv1.ApiProduct{
								Name:        APIProduct,
								Description: aws.String("test description"),
							},
						},
					})

					Expect(err).NotTo(HaveOccurred())
					Expect(resp).To(BeAssignableToTypeOf(portalv1.CreateAPIProduct201Response{}))
				})
				It("returns not found on delete", func() {
					resp, err := s.DeleteAPIProduct(ctx, portalv1.DeleteAPIProductRequestObject{
						Name: "non-existant-APIProduct",
					})
					Expect(err).NotTo(HaveOccurred())
					Expect(resp).To(BeAssignableToTypeOf(portalv1.DeleteAPIProduct404JSONResponse{}))
				})
			})
			When("resource server exists", func() {
				var expScope = "test-APIProduct"
				BeforeEach(func() {
					// Mock resource server with a single APIProduct.
					access := resourceServer
					mockCognitoClient.EXPECT().DescribeResourceServer(ctx, gomock.Any(), gomock.Any()).AnyTimes().Return(
						&cognito.DescribeResourceServerOutput{
							ResourceServer: &types.ResourceServerType{
								Identifier: &access,
								Name:       &access,
								Scopes: []types.ResourceServerScopeType{
									{
										ScopeName:        &expScope,
										ScopeDescription: aws.String("test description"),
									},
								},
							},
						}, nil)

					// Updating resource server is valid.
					mockCognitoClient.EXPECT().UpdateResourceServer(ctx, gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(
						func(
							ctx context.Context,
							input *cognito.UpdateResourceServerInput,
							optFns ...func(*cognito.Options),
						) (*cognito.UpdateResourceServerOutput, error) {
							return &cognito.UpdateResourceServerOutput{
								ResourceServer: &types.ResourceServerType{
									Identifier: input.Identifier,
									Name:       input.Name,
									Scopes:     input.Scopes,
								},
							}, nil
						})
				})
				It("can create APIProduct", func() {
					APIProduct := "new-APIProduct"
					resp, err := s.CreateAPIProduct(ctx, portalv1.CreateAPIProductRequestObject{
						Body: &portalv1.CreateAPIProductJSONRequestBody{
							ApiProduct: portalv1.ApiProduct{
								Name:        APIProduct,
								Description: aws.String("test description"),
							},
						},
					})
					Expect(err).NotTo(HaveOccurred())
					Expect(resp).To(BeAssignableToTypeOf(portalv1.CreateAPIProduct201Response{}))
				})
				It("returns not found if deleting APIProduct not present", func() {
					resp, err := s.DeleteAPIProduct(ctx, portalv1.DeleteAPIProductRequestObject{
						Name: "non-existant-APIProduct",
					})
					Expect(err).NotTo(HaveOccurred())
					Expect(resp).To(BeAssignableToTypeOf(portalv1.DeleteAPIProduct404JSONResponse{}))
				})

				It("can delete the APIProduct", func() {
					resp, err := s.DeleteAPIProduct(ctx, portalv1.DeleteAPIProductRequestObject{
						Name: expScope,
					})
					Expect(err).NotTo(HaveOccurred())
					Expect(resp).To(BeAssignableToTypeOf(portalv1.DeleteAPIProduct204Response{}))
				})
				It("returns that there is a resource conflict", func() {
					resp, err := s.CreateAPIProduct(ctx, portalv1.CreateAPIProductRequestObject{
						Body: &portalv1.CreateAPIProductJSONRequestBody{
							ApiProduct: portalv1.ApiProduct{
								Name:        expScope,
								Description: aws.String("test description"),
							},
						},
					})
					Expect(err).NotTo(HaveOccurred())
					Expect(resp).To(BeAssignableToTypeOf(portalv1.CreateAPIProduct409JSONResponse{}))
				})
			})
		})
		Context("Client APIProducts", func() {
			var (
				expClient = "test-client"

				expAPIProducts = []string{"tracks"}
			)

			BeforeEach(func() {
				// Mock with a single known user and single expAPIProduct.
				mockCognitoClient.EXPECT().UpdateUserPoolClient(ctx, gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(
					func(
						ctx context.Context,
						input *cognito.UpdateUserPoolClientInput,
						optFns ...func(*cognito.Options),
					) (*cognito.UpdateUserPoolClientOutput, error) {
						if *input.ClientId == expClient {
							return &cognito.UpdateUserPoolClientOutput{
								UserPoolClient: &types.UserPoolClientType{
									ClientId:           input.ClientId,
									ClientName:         input.ClientName,
									AllowedOAuthScopes: input.AllowedOAuthScopes,
								},
							}, nil
						}

						return nil, &smithyhttp.ResponseError{
							Response: &smithyhttp.Response{
								Response: &http.Response{
									StatusCode: 404,
									Status:     "Resource Not Found",
								},
							},
							Err: &types.ResourceNotFoundException{
								Message: aws.String("resource not found"),
							},
						}
					})
			})
			When("client does not exist", func() {
				It("returns not found on update", func() {
					resp, err := s.UpdateAppAPIProducts(ctx, portalv1.UpdateAppAPIProductsRequestObject{
						Name: "non-existant-client",
						Body: &portalv1.UpdateAppAPIProductsJSONRequestBody{
							ApiProductNames: expAPIProducts,
						},
					})
					Expect(err).NotTo(HaveOccurred())
					Expect(resp).To(BeAssignableToTypeOf(portalv1.UpdateAppAPIProducts404JSONResponse{}))
				})
			})
			When("referencing client that does exist", func() {
				It("can update client APIProducts", func() {
					resp, err := s.UpdateAppAPIProducts(ctx, portalv1.UpdateAppAPIProductsRequestObject{
						Name: expClient,
						Body: &portalv1.UpdateAppAPIProductsJSONRequestBody{
							ApiProductNames: expAPIProducts,
						},
					})
					Expect(err).NotTo(HaveOccurred())
					Expect(resp).To(BeAssignableToTypeOf(portalv1.UpdateAppAPIProducts204Response{}))
				})
			})
		})
	})
})
