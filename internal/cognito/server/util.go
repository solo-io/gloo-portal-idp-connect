package server

import (
	"log"

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

// cognitoScopeToAPIScope converts a Cognito scope to an API scope type
func cognitoScopesToAPIScopesType(cognitoScopes ...types.ResourceServerScopeType) []portalv1.Scope {
	var apiScopes []portalv1.Scope
	for _, scope := range cognitoScopes {
		if scope.ScopeDescription == nil || scope.ScopeName == nil {
			log.Printf("Skipping scope with nil description or name: %v", scope)
			continue
		}

		apiScopes = append(apiScopes, portalv1.Scope{
			Description: *scope.ScopeDescription,
			Value:       *scope.ScopeName,
		})
	}

	return apiScopes
}

func apiScopesToCognitoScopeType(apiScope portalv1.Scope) types.ResourceServerScopeType {
	return types.ResourceServerScopeType{
		ScopeDescription: &apiScope.Description,
		ScopeName:        &apiScope.Value,
	}
}
