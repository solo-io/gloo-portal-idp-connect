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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/solo-io/gloo-portal-idp-connect/cognito/internal/server"
	"github.com/solo-io/gloo-portal-idp-connect/cognito/internal/server/mock"
	portalv1 "github.com/solo-io/gloo-portal-idp-connect/pkg/api/v1"
)

const userPoolID = "us-west-2_abc123"

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
		}, mockCognitoClient)
	})

	Context("Client", func() {
		When("no client exists", func() {
			BeforeEach(func() {
				genClientId := "2r7vpfuuhbimiqq9bmfde1e3t3"
				genClientSecret := "6au6kel0b"

				// Return client not found error on describe.
				mockCognitoClient.EXPECT().DescribeUserPoolClient(ctx, gomock.Any(), gomock.Any()).AnyTimes().Return(
					nil,
					&smithyhttp.ResponseError{
						Response: &smithyhttp.Response{
							Response: &http.Response{
								StatusCode: 404,
								Status:     "Resource Not Found",
							},
						},
						Err: errors.New("client not found"),
					})

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
				Expect(resp).To(BeAssignableToTypeOf(portalv1.CreateClient200JSONResponse{}))
				resp200 := resp.(portalv1.CreateClient200JSONResponse)
				Expect(*resp200.ClientName).To(Equal(client))
				Expect(resp200.ClientId).NotTo(BeNil())
				Expect(resp200.ClientSecret).NotTo(BeNil())
			})

			It("returns error code on nil body", func() {
				resp, err := s.CreateClient(ctx, portalv1.CreateClientRequestObject{})
				Expect(err).NotTo(HaveOccurred())
				Expect(resp).To(BeAssignableToTypeOf(portalv1.CreateClient500JSONResponse{}))
			})

			It("returns not found code on get", func() {
				resp, err := s.GetClient(ctx, portalv1.GetClientRequestObject{
					Params: portalv1.GetClientParams{
						Id: "test-client",
					},
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(resp).To(BeAssignableToTypeOf(portalv1.GetClient404JSONResponse{}))
				resp404 := resp.(portalv1.GetClient404JSONResponse)
				Expect(resp404.Code).To(Equal(404))
			})

			It("returns not found code on deletion", func() {
				resp, err := s.DeleteClient(ctx, portalv1.DeleteClientRequestObject{
					Params: portalv1.DeleteClientParams{
						Id: "test-client",
					},
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
				// Return client on describe.
				mockCognitoClient.EXPECT().DescribeUserPoolClient(ctx, gomock.Any(), gomock.Any()).AnyTimes().Return(
					&cognito.DescribeUserPoolClientOutput{
						UserPoolClient: &types.UserPoolClientType{
							ClientId:   &clientId,
							ClientName: &clientName,
						},
					},
					nil,
				)

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

			It("can get the client", func() {
				resp, err := s.GetClient(ctx, portalv1.GetClientRequestObject{
					Params: portalv1.GetClientParams{
						Id: clientId,
					},
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(resp).To(BeAssignableToTypeOf(portalv1.GetClient200JSONResponse{}))
				resp200 := resp.(portalv1.GetClient200JSONResponse)
				Expect(*resp200.ClientName).To(Equal(clientName))
				Expect(*resp200.ClientId).To(Equal(clientId))
			})

			It("can delete the client", func() {
				resp, err := s.DeleteClient(ctx, portalv1.DeleteClientRequestObject{
					Params: portalv1.DeleteClientParams{
						Id: clientId,
					},
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(resp).To(BeAssignableToTypeOf(portalv1.DeleteClient204Response{}))
			})

		})

		Context("Scopes", func() {
			When("resource server does not exist", func() {
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

					// Expect that we create the resource server EXACTLY once.
					access := server.AccessResourceServerName
					mockCognitoClient.EXPECT().CreateResourceServer(ctx, gomock.Any(), gomock.Any()).Times(1).Return(
						&cognito.CreateResourceServerOutput{
							ResourceServer: &types.ResourceServerType{
								Identifier: &access,
								Name:       &access,
							},
						},
						nil,
					)

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
				It("returns null when getting scopes", func() {
					resp, err := s.GetScopes(ctx, portalv1.GetScopesRequestObject{})
					Expect(err).NotTo(HaveOccurred())
					Expect(resp).To(BeAssignableToTypeOf(portalv1.GetScopes200JSONResponse{}))
					resp200 := resp.(portalv1.GetScopes200JSONResponse)
					Expect(resp200.Scopes).To(BeNil())
				})

				It("can create a scope", func() {
					scope := "test-scope"
					resp, err := s.CreateScope(ctx, portalv1.CreateScopeRequestObject{
						Body: &portalv1.CreateScopeJSONRequestBody{
							Scope: portalv1.Scope{
								Value:       scope,
								Description: "test description",
							},
						},
					})

					Expect(err).NotTo(HaveOccurred())
					Expect(resp).To(BeAssignableToTypeOf(portalv1.CreateScope204Response{}))
				})
			})
			When("resource server exists", func() {
				var expScope = "test-scope"
				BeforeEach(func() {
					// Mock resource server with a single scope.
					access := server.AccessResourceServerName
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
				It("can list scopes", func() {
					resp, err := s.GetScopes(ctx, portalv1.GetScopesRequestObject{})
					Expect(err).NotTo(HaveOccurred())
					Expect(resp).To(BeAssignableToTypeOf(portalv1.GetScopes200JSONResponse{}))
					resp200 := resp.(portalv1.GetScopes200JSONResponse)
					Expect(resp200.Scopes).To(HaveLen(1))
					Expect(resp200.Scopes[0].Value).To(Equal(expScope))
				})
				It("can create scope", func() {
					scope := "new-scope"
					resp, err := s.CreateScope(ctx, portalv1.CreateScopeRequestObject{
						Body: &portalv1.CreateScopeJSONRequestBody{
							Scope: portalv1.Scope{
								Value:       scope,
								Description: "test description",
							},
						},
					})
					Expect(err).NotTo(HaveOccurred())
					Expect(resp).To(BeAssignableToTypeOf(portalv1.CreateScope204Response{}))
				})
				It("returns not found if deleting scope not present", func() {
					resp, err := s.DeleteScope(ctx, portalv1.DeleteScopeRequestObject{
						Params: portalv1.DeleteScopeParams{
							Scope: "non-existant-scope",
						},
					})
					Expect(err).NotTo(HaveOccurred())
					Expect(resp).To(BeAssignableToTypeOf(portalv1.DeleteScope404JSONResponse{}))
				})

				It("can delete the scope", func() {
					resp, err := s.DeleteScope(ctx, portalv1.DeleteScopeRequestObject{
						Params: portalv1.DeleteScopeParams{
							Scope: expScope,
						},
					})
					Expect(err).NotTo(HaveOccurred())
					Expect(resp).To(BeAssignableToTypeOf(portalv1.DeleteScope204Response{}))
				})
				It("returns that there is a resource conflict", func() {
					resp, err := s.CreateScope(ctx, portalv1.CreateScopeRequestObject{
						Body: &portalv1.CreateScopeJSONRequestBody{
							Scope: portalv1.Scope{
								Value:       expScope,
								Description: "test description",
							},
						},
					})
					Expect(err).NotTo(HaveOccurred())
					Expect(resp).To(BeAssignableToTypeOf(portalv1.CreateScope409JSONResponse{}))
				})
			})
		})
		Context("Client Scopes", func() {
			var (
				expClient = "test-client"

				expScopes = []string{"tracks"}
			)

			BeforeEach(func() {
				// Mock for having a single client, expClient, with expScopes.
				mockCognitoClient.EXPECT().DescribeUserPoolClient(ctx, gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(
					func(
						ctx context.Context,
						input *cognito.DescribeUserPoolClientInput,
						optFns ...func(*cognito.Options),
					) (*cognito.DescribeUserPoolClientOutput, error) {
						// If request includes our expClient, return valid.
						if *input.ClientId == expClient {
							return &cognito.DescribeUserPoolClientOutput{
								UserPoolClient: &types.UserPoolClientType{
									ClientId:           &expClient,
									ClientName:         &expClient,
									AllowedOAuthScopes: expScopes,
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
							Err: errors.New("client not found"),
						}
					})

				// Mock for updating client scopes.
				mockCognitoClient.EXPECT().UpdateUserPoolClient(ctx, gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(
					func(
						ctx context.Context,
						input *cognito.UpdateUserPoolClientInput,
						optFns ...func(*cognito.Options),
					) (*cognito.UpdateUserPoolClientOutput, error) {
						return &cognito.UpdateUserPoolClientOutput{
							UserPoolClient: &types.UserPoolClientType{
								ClientId:           input.ClientId,
								ClientName:         input.ClientName,
								AllowedOAuthScopes: input.AllowedOAuthScopes,
							},
						}, nil
					})
			})
			When("client does not exist", func() {
				var newClient = "new-client"
				It("returns not found on get", func() {
					resp, err := s.GetClientScopes(ctx, portalv1.GetClientScopesRequestObject{
						Params: portalv1.GetClientScopesParams{
							Id: newClient,
						},
					})
					Expect(err).NotTo(HaveOccurred())
					Expect(resp).To(BeAssignableToTypeOf(portalv1.GetClientScopes404JSONResponse{}))
				})
				It("returns not found on delete", func() {
					resp, err := s.DeleteClientScope(ctx, portalv1.DeleteClientScopeRequestObject{
						Params: portalv1.DeleteClientScopeParams{
							Id:    newClient,
							Scope: "test-scope",
						},
					})
					Expect(err).NotTo(HaveOccurred())
					Expect(resp).To(BeAssignableToTypeOf(portalv1.DeleteClientScope404JSONResponse{}))
				})
				It("returns not found on add", func() {
					resp, err := s.AddClientScope(ctx, portalv1.AddClientScopeRequestObject{
						Body: &portalv1.AddClientScopeJSONRequestBody{
							Id:    newClient,
							Scope: "test-scope",
						},
					})
					Expect(err).NotTo(HaveOccurred())
					Expect(resp).To(BeAssignableToTypeOf(portalv1.AddClientScope404JSONResponse{}))
				})
			})
			When("referencing client that does exist", func() {
				It("can get client scopes", func() {
					resp, err := s.GetClientScopes(ctx, portalv1.GetClientScopesRequestObject{
						Params: portalv1.GetClientScopesParams{
							Id: expClient,
						},
					})
					Expect(err).NotTo(HaveOccurred())
					Expect(resp).To(BeAssignableToTypeOf(portalv1.GetClientScopes200JSONResponse{}))
					resp200 := resp.(portalv1.GetClientScopes200JSONResponse)
					Expect(resp200.Scopes).To(Equal(expScopes))
				})
				It("can delete client scope", func() {
					resp, err := s.DeleteClientScope(ctx, portalv1.DeleteClientScopeRequestObject{
						Params: portalv1.DeleteClientScopeParams{
							Id:    expClient,
							Scope: expScopes[0],
						},
					})
					Expect(err).NotTo(HaveOccurred())
					Expect(resp).To(BeAssignableToTypeOf(portalv1.DeleteClientScope204Response{}))
				})
				It("returns not found if deleting non-existant scope", func() {
					resp, err := s.DeleteClientScope(ctx, portalv1.DeleteClientScopeRequestObject{
						Params: portalv1.DeleteClientScopeParams{
							Id:    expClient,
							Scope: "non-existant-scope",
						},
					})
					Expect(err).NotTo(HaveOccurred())
					Expect(resp).To(BeAssignableToTypeOf(portalv1.DeleteClientScope404JSONResponse{}))
				})
				It("returns conflict if adding existing scope", func() {
					resp, err := s.AddClientScope(ctx, portalv1.AddClientScopeRequestObject{
						Body: &portalv1.AddClientScopeJSONRequestBody{
							Id:    expClient,
							Scope: "tracks",
						},
					})
					Expect(err).NotTo(HaveOccurred())
					Expect(resp).To(BeAssignableToTypeOf(portalv1.AddClientScope409JSONResponse{}))
				})
				It("can add client scope", func() {
					resp, err := s.AddClientScope(ctx, portalv1.AddClientScopeRequestObject{
						Body: &portalv1.AddClientScopeJSONRequestBody{
							Id:    expClient,
							Scope: "new-scope",
						},
					})
					Expect(err).NotTo(HaveOccurred())
					Expect(resp).To(BeAssignableToTypeOf(portalv1.AddClientScope204Response{}))
				})
			})
		})
	})
})
