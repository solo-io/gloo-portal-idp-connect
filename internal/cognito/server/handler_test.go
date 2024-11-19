package server_test

import (
	"context"
	"errors"
	"net/http"

	cognito "github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"
	smithyhttp "github.com/aws/smithy-go/transport/http"
	_ "github.com/golang/mock/mockgen/model"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

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
		s                   *server.StrictServerHandler
		mockCtrl            *gomock.Controller
		mockCognitoClient   *mock_server.MockCognitoClient
		ctx                 context.Context
		applicationClientId = "client-internal-id"
		testToken           = "test"
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
				Expect(resp200.ClientId).NotTo(BeNil())
				Expect(resp200.ClientSecret).NotTo(BeNil())
			})

			It("returns error code on nil body", func() {
				resp, err := s.CreateOAuthApplication(ctx, portalv1.CreateOAuthApplicationRequestObject{})
				Expect(err).NotTo(HaveOccurred())
				Expect(resp).To(BeAssignableToTypeOf(portalv1.CreateOAuthApplication400JSONResponse{}))
			})

			It("returns an error code on empty client id", func() {
				resp, err := s.CreateOAuthApplication(ctx, portalv1.CreateOAuthApplicationRequestObject{
					Params: portalv1.CreateOAuthApplicationParams{
						Token: &testToken,
					},
					Body: &portalv1.CreateOAuthApplicationJSONRequestBody{
						Id: "",
					},
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(resp).To(BeAssignableToTypeOf(portalv1.CreateOAuthApplication400JSONResponse{}))
			})

			It("returns not found code on deletion", func() {
				resp, err := s.DeleteOAuthApplication(ctx, portalv1.DeleteOAuthApplicationRequestObject{
					Id: "test-client",
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
				resp, err := s.DeleteOAuthApplication(ctx, portalv1.DeleteOAuthApplicationRequestObject{
					Id: clientId,
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
