package server

import (
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"

	portalv1 "github.com/solo-io/gloo-portal-idp-connect/pkg/api/v1"
)

func newPortalError(code int, msg, reason string) portalv1.Error {
	return portalv1.Error{
		Code:    code,
		Message: msg,
		Reason:  reason,
	}
}

func newPortal500Error(reason string) portalv1.Error {
	return newPortalError(500, "Internal Server Error", reason)
}

func apiScopesToCognitoScopeType(apiScope portalv1.Scope) types.ResourceServerScopeType {
	return types.ResourceServerScopeType{
		ScopeDescription: &apiScope.Description,
		ScopeName:        &apiScope.Value,
	}
}
