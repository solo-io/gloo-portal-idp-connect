package server

import (
	"crypto/md5"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider/types"

	portalv1 "github.com/solo-io/gloo-portal-idp-connect/pkg/api/v1"
)

// shortenName returns a shortened version of the input string.
// It is based on the `kubeutils.SanitizeNameV2` function, but it
// just does the shortening part.
func shortenName(name string) string {
	if len(name) > 63 {
		hash := md5.Sum([]byte(name))
		name = fmt.Sprintf("%s-%x", name[:31], hash)
		name = name[:63]
	}
	return name
}

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

// cognitoScopeToAPIScope converts a Cognito scope to an API scope type
func cognitoScopesToAPIScopesType(cognitoScopes ...types.ResourceServerScopeType) []portalv1.Scope {
	var apiScopes []portalv1.Scope
	for _, scope := range cognitoScopes {
		apiScopes = append(apiScopes, portalv1.Scope{
			Description: scope.ScopeDescription,
			Value:       scope.ScopeName,
		})
	}

	return apiScopes
}

func apiScopesToCognitoScopeType(apiScope portalv1.Scope) types.ResourceServerScopeType {
	return types.ResourceServerScopeType{
		ScopeDescription: apiScope.Description,
		ScopeName:        apiScope.Value,
	}
}
