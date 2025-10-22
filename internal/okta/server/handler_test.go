package server_test

import (
	"context"

	"github.com/okta/okta-sdk-golang/v5/okta"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	"github.com/solo-io/gloo-portal-idp-connect/internal/okta/server"
	mock_server "github.com/solo-io/gloo-portal-idp-connect/internal/okta/server/mock"
	portalv1 "github.com/solo-io/gloo-portal-idp-connect/pkg/api/v1"
)

var _ = Describe("Server", func() {

	const (
		applicationClientId     = "test-client-id"
		applicationClientSecret = "test-client-secret"
		applicationId           = "0oa1234567890abcdef"
	)

	var (
		s              *server.StrictServerHandler
		mockCtrl       *gomock.Controller
		mockOktaClient *mock_server.MockOktaClient
		mockAppAPI     *mock_server.MockApplicationAPI
		ctx            context.Context
		testToken      = "test"
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockOktaClient = mock_server.NewMockOktaClient(mockCtrl)
		mockAppAPI = mock_server.NewMockApplicationAPI(mockCtrl)
		ctx = context.Background()

		mockOktaClient.EXPECT().GetApplicationAPI().Return(mockAppAPI).AnyTimes()

		s = server.NewStrictServerHandler(mockOktaClient)
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Context("Application", func() {

		When("no client exists", func() {

			It("can create a client", func() {
				// Create expected application
				credentials := okta.NewOAuthApplicationCredentials()
				oauthClient := okta.NewApplicationCredentialsOAuthClient()
				oauthClient.SetClientId(applicationClientId)
				oauthClient.SetClientSecret(applicationClientSecret)
				credentials.SetOauthClient(*oauthClient)

				app := okta.NewOpenIdConnectApplication(
					*credentials,
					"oidc_client",
					*okta.NewOpenIdConnectApplicationSettings(),
					applicationClientId,
					"OPENID_CONNECT",
				)
				app.SetId(applicationId)

				appUnion := okta.OpenIdConnectApplicationAsListApplications200ResponseInner(app)

				mockCreateReq := mock_server.NewMockApiCreateApplicationRequest(mockCtrl)
				mockCreateReq.EXPECT().Application(gomock.Any()).Return(mockCreateReq)
				mockCreateReq.EXPECT().Execute().Return(&appUnion, &okta.APIResponse{}, nil)

				mockAppAPI.EXPECT().CreateApplication(ctx).Return(mockCreateReq)

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
				mockListReq := mock_server.NewMockApiListApplicationsRequest(mockCtrl)
				mockListReq.EXPECT().Execute().Return([]okta.ListApplications200ResponseInner{}, &okta.APIResponse{}, nil)

				mockAppAPI.EXPECT().ListApplications(ctx).Return(mockListReq)

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
			var dummyApp *okta.OpenIdConnectApplication

			BeforeEach(func() {
				credentials := okta.NewOAuthApplicationCredentials()
				oauthClient := okta.NewApplicationCredentialsOAuthClient()
				oauthClient.SetClientId(applicationClientId)
				oauthClient.SetClientSecret(applicationClientSecret)
				credentials.SetOauthClient(*oauthClient)

				dummyApp = okta.NewOpenIdConnectApplication(
					*credentials,
					"oidc_client",
					*okta.NewOpenIdConnectApplicationSettings(),
					applicationClientId,
					"OPENID_CONNECT",
				)
				dummyApp.SetId(applicationId)
			})

			It("can delete the client", func() {
				appUnion := okta.OpenIdConnectApplicationAsListApplications200ResponseInner(dummyApp)

				mockListReq := mock_server.NewMockApiListApplicationsRequest(mockCtrl)
				mockListReq.EXPECT().Execute().Return([]okta.ListApplications200ResponseInner{appUnion}, &okta.APIResponse{}, nil)

				mockDeactivateReq := mock_server.NewMockApiDeactivateApplicationRequest(mockCtrl)
				mockDeactivateReq.EXPECT().Execute().Return(&okta.APIResponse{}, nil)

				mockDeleteReq := mock_server.NewMockApiDeleteApplicationRequest(mockCtrl)
				mockDeleteReq.EXPECT().Execute().Return(&okta.APIResponse{}, nil)

				mockAppAPI.EXPECT().ListApplications(ctx).Return(mockListReq)
				mockAppAPI.EXPECT().DeactivateApplication(ctx, applicationId).Return(mockDeactivateReq)
				mockAppAPI.EXPECT().DeleteApplication(ctx, applicationId).Return(mockDeleteReq)

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
