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

func newPortal400Error(reason string) portalv1.Error {
	return newPortalError(400, "Bad Request", reason)
}

func newPortal500Error(reason string) portalv1.Error {
	return newPortalError(500, "Internal Server Error", reason)
}

func apiProductToCognitoScopeType(apiProduct portalv1.ApiProduct) types.ResourceServerScopeType {
	return types.ResourceServerScopeType{
		ScopeDescription: apiProduct.Description,
		ScopeName:        &apiProduct.Name,
	}
}
