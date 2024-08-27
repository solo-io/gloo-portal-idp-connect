// Package v1 provides primitives to interact with the openapi HTTP API.
//
// Code generated by github.com/deepmap/oapi-codegen/v2 version v2.1.0 DO NOT EDIT.
package v1

// ApiProduct defines model for ApiProduct.
type ApiProduct struct {
	Description *string `json:"description,omitempty"`
	Name        string  `json:"name"`
}

// Error defines model for Error.
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Reason  string `json:"reason"`
}

// CreateAPIProductJSONBody defines parameters for CreateAPIProduct.
type CreateAPIProductJSONBody struct {
	ApiProduct ApiProduct `json:"apiProduct"`
}

// CreateOAuthApplicationJSONBody defines parameters for CreateOAuthApplication.
type CreateOAuthApplicationJSONBody struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

// UpdateAppAPIProductsJSONBody defines parameters for UpdateAppAPIProducts.
type UpdateAppAPIProductsJSONBody struct {
	ApiProducts []string `json:"apiProducts"`
}

// CreateAPIProductJSONRequestBody defines body for CreateAPIProduct for application/json ContentType.
type CreateAPIProductJSONRequestBody CreateAPIProductJSONBody

// CreateOAuthApplicationJSONRequestBody defines body for CreateOAuthApplication for application/json ContentType.
type CreateOAuthApplicationJSONRequestBody CreateOAuthApplicationJSONBody

// UpdateAppAPIProductsJSONRequestBody defines body for UpdateAppAPIProducts for application/json ContentType.
type UpdateAppAPIProductsJSONRequestBody UpdateAppAPIProductsJSONBody
