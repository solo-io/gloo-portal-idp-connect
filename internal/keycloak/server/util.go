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
