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

	Context("Client", func() {
		When("no client exists", func() {
			BeforeEach(func() {
				genClientId := "2r7vpfuuhbimiqq9bmfde1e3t3"
				genClientSecret := "6au6kel0b"

				// Return generated client ID and Secret on user-provided client name.
				mockCognitoClient.EXPECT().CreateUserPoolClient(ctx, gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(
					func(
						ctx context.Context,
						input *cognito.CreateUserPoolClientInput,
						optFns ...interface{},
					) (*cognito.CreateUserPoolClientOutput, error) {

						return &cognito.CreateUserPoolClientOutput{
							UserPoolClient: &types.UserPoolClientType{
								ClientId:     &genClientId,
								ClientSecret: &genClientSecret,
								ClientName:   input.ClientName,
							},
						}, nil
					})

				// Return client not found error on describe.
				mockCognitoClient.EXPECT().DeleteUserPoolClient(ctx, gomock.Any(), gomock.Any()).AnyTimes().Return(
					nil,
					&smithyhttp.ResponseError{
						Response: &smithyhttp.Response{
							Response: &http.Response{
								StatusCode: 404,
								Status:     "Resource Not Found",
							},
						},
						Err: errors.New("client does not exist"),
					},
				)
			})
			It("can create a client", func() {
				client := "test-client"
				resp, err := s.CreateClient(ctx, portalv1.CreateClientRequestObject{
					Body: &portalv1.CreateClientJSONRequestBody{
						ClientName: client,
					},
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(resp).To(BeAssignableToTypeOf(portalv1.CreateClient201JSONResponse{}))
				resp200 := resp.(portalv1.CreateClient201JSONResponse)
				Expect(*resp200.ClientName).To(Equal(client))
				Expect(resp200.ClientId).NotTo(BeNil())
				Expect(resp200.ClientSecret).NotTo(BeNil())
			})

			It("returns error code on nil body", func() {
				resp, err := s.CreateClient(ctx, portalv1.CreateClientRequestObject{})
				Expect(err).NotTo(HaveOccurred())
				Expect(resp).To(BeAssignableToTypeOf(portalv1.CreateClient400JSONResponse{}))
			})

			It("returns error code on empty client name", func() {
				resp, err := s.CreateClient(ctx, portalv1.CreateClientRequestObject{
					Body: &portalv1.CreateClientJSONRequestBody{
						ClientName: "",
					},
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(resp).To(BeAssignableToTypeOf(portalv1.CreateClient400JSONResponse{}))
			})

			It("returns not found code on deletion", func() {
				resp, err := s.DeleteClient(ctx, portalv1.DeleteClientRequestObject{
					Id: "test-client",
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(resp).To(BeAssignableToTypeOf(portalv1.DeleteClient404JSONResponse{}))
				resp404 := resp.(portalv1.DeleteClient404JSONResponse)
				Expect(resp404.Code).To(Equal(404))
			})
		})

		When("client exists", func() {
			var (
				clientId   = "2r7vpfuuhbimiqq9bmfde1e3t3"
				clientName = "test-client"
			)
			BeforeEach(func() {
				genClientId := "4q5270uvfj8v86vc8oqfk3f4m9"

				// Delete client on delete
				mockCognitoClient.EXPECT().DeleteUserPoolClient(ctx, gomock.Any(), gomock.Any()).AnyTimes().Return(
					&cognito.DeleteUserPoolClientOutput{},
					nil,
				)

				// Create client with new id when same name is given.
				mockCognitoClient.EXPECT().CreateUserPoolClient(ctx, gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(
					func(
						ctx context.Context,
						input *cognito.CreateUserPoolClientInput,
						optFns ...interface{},
					) (*cognito.CreateUserPoolClientOutput, error) {
						return &cognito.CreateUserPoolClientOutput{
							UserPoolClient: &types.UserPoolClientType{
								ClientId:   &genClientId,
								ClientName: &clientName,
							},
						}, nil
					})
			})

			It("can delete the client", func() {
				resp, err := s.DeleteClient(ctx, portalv1.DeleteClientRequestObject{
					Id: clientId,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(resp).To(BeAssignableToTypeOf(portalv1.DeleteClient204Response{}))
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
						ApiProduct: "non-existant-APIProduct",
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
						ApiProduct: "non-existant-APIProduct",
					})
					Expect(err).NotTo(HaveOccurred())
					Expect(resp).To(BeAssignableToTypeOf(portalv1.DeleteAPIProduct404JSONResponse{}))
				})

				It("can delete the APIProduct", func() {
					resp, err := s.DeleteAPIProduct(ctx, portalv1.DeleteAPIProductRequestObject{
						ApiProduct: expScope,
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
					resp, err := s.UpdateClientAPIProducts(ctx, portalv1.UpdateClientAPIProductsRequestObject{
						Id: "non-existant-client",
						Body: &portalv1.UpdateClientAPIProductsJSONRequestBody{
							ApiProducts: expAPIProducts,
						},
					})
					Expect(err).NotTo(HaveOccurred())
					Expect(resp).To(BeAssignableToTypeOf(portalv1.UpdateClientAPIProducts404JSONResponse{}))
				})
			})
			When("referencing client that does exist", func() {
				It("can update client APIProducts", func() {
					resp, err := s.UpdateClientAPIProducts(ctx, portalv1.UpdateClientAPIProductsRequestObject{
						Id: expClient,
						Body: &portalv1.UpdateClientAPIProductsJSONRequestBody{
							ApiProducts: expAPIProducts,
						},
					})
					Expect(err).NotTo(HaveOccurred())
					Expect(resp).To(BeAssignableToTypeOf(portalv1.UpdateClientAPIProducts204Response{}))
				})
			})
		})
	})
})
