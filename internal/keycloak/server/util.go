package server

import (
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

func PermissionName(clientId, apiName string) string {
	return clientId + "/" + apiName
}

func GetApiNameFromPermission(permissionId, clientId string) string {
	// we extract the api platform id from the permission ID, by removing the ClientID + '/' from the beginning
	return permissionId[len(clientId)+1:]
}
