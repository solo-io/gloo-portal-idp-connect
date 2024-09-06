# Use with Gloo Gateway

The idea behind the Service Programming Interface is to provide an interopability layer between Gloo Gateway Portal and the IDP that the customer wants to use. In order to get the desired functionality, there are two flows that need to be implemented: IDP Configuration and Data Path Authorization.

## AWS Cognito Example

### IDP Configuration

![Create Client Flow](./images/create-client-flow.png)

When an application is created in Gloo Portal, the SPI will be called to create the client representation in the SPI. The Client ID and Secret can then be used to create the access_token via Cognito's token endpoint.

![Cognito Client](./images/cognito-client.png)

A client is created in AWS Cognito. The expectation is that the OAuth flow type "client_credentials" is selected so that an access token can be created from the client id and secret. This, however, is a decision that the customer can make to decide how credentials are managed and tokens are distributed.

![Create API Product Flow](./images/create-api-product-flow.png)

When an API Product is created in Gloo Portal, the SPI will be called to create the representation in the IDP. For Cognito, this would most likely be represented as a resource server:

![Cognito Resource Server](./images/cognito-resource-server.png)

For Cognito clients which have been given a custom scope via this resource server, the scope will take the form `<resource-server>/<custom-scope>`. In this diagram, it would be `access/tracks-rest-api`. Below, we will discuss how this can be used on the data path to authorize a request to your API Product.

![Add API Product to Application Flow](./images/app-authorize-api-product-flow.png)

Once an Application and API Product is in the system, you can begin to authorize API Products for particular applications.

![Adding Scopes to Applications](./images/tracks-rest-api-custom-scope.png)

Once the API Product has been given access to the application, you can see that the custom scope is included in the client's "Hosted UI" section.

### Data Path Authorization

![Obtaining Access Token](./images/retrieve-credentials.png)

Cognito's token endpoint can be used to retrieve the access token. See [Configuring Gloo Gateway](./configuring-gloo-gateway.md) for more information on how to use the access token to authorize requests to your API Products.

![Data Path](./images/data-path.png)

The Access token can be used to authorize requests via ext-auth. The most convenient method would be to use OPA in order to match the scope of the access token against the `apiProductId` of the API Product. See [Configuring Gloo Gateway](./configuring-gloo-gateway.md) for more information on how to use the access token to authorize requests to your API Products.

## Keycloak Example

### IDP Configuration

When an application is created in Gloo Portal, the SPI will be called to create the client representation in the SPI. This will result in a new _client_ created in the Keycloak realm, with an auto-generated client ID and secret returned to the caller.

> **Note:** Clients created by the SPI use Keycloak's default settings, and no assumptions are made about how the client will manage and distribute tokens. It is left to the customer to decide how the created clients should be configured.

When an API Product is created in Gloo Portal, the SPI will be called to create the representation in the IDP. For Keycloak, this would most likely be represented as a _resource_ managed by the _resource server_ client. For convenience, the default implementation assumes that the client being used by the SPI to manage applications is also the resource server that will manage API products.

With at least one Application (client) and one API Product (resource) registered in Keycloak, you can begin to authorize API Products for particular applications. This can be represented in Keycloak as a _permission_ granted on the API product resource to the client application.

Once the application has been given access to the API Product, the application can obtain a Requesting Party Token (RPT) from Keycloak's token endpoint, or the policy enforcement point (e.g. Gloo ext-auth) can interrogate Keycloak directly to validate that the user and application are authorised to the API Product.

### Data Path Authorization

Policy enforcement points (such as Gloo ext-auth) have at least two options for checking authorised access to protected API Products with the representations described above:

* Validating a Requesting Party Token (RPT) obtained by the client, possibly via a [UMA Grant Flow](https://www.keycloak.org/docs/latest/authorization_services/#_service_uma_authorization_process)
* Directly checking permissions in Keycloak

In either case, requests can be authorised via Gloo ext-auth. The most convenient method is to use OPA in order to match permissions against the `apiProductId` of the API Product. See [Configuring Gloo Gateway](./configuring-gloo-gateway.md) for an example of how to use the access token to authorize requests to your API Products.

#### Requesting Party Token (RPT)

Whether following the UMA Grant Flow or otherwise, the client will need to use Keycloak's token endpoint to obtain a RPT on the user's behalf.

> **Note:** The resource server client ID (i.e. the client ID used by the SPI itself) must be specified as the `audience` in the token request for the correct permissions to be evaluated. This will be automatic if an RPT is obtained using a permission ticket in the UMA Grant Flow.

The access token returned will contain a new `authorization` claim with permissions for the permitted API products:

```json
"authorization": {
  "permissions": [
    {
      "rsid": "aa6edf59-a4b7-4532-b6b1-a5b423da7809",
      "rsname": "tracks-rest-api"
    }
  ]
}
```

Requests to the API Product can then be (re-)tried using this new access token.

See <https://www.keycloak.org/docs/latest/authorization_services/index.html#_service_obtaining_permissions> for more details on obtaining an RPT.

## Direct permission check

Rather than having the client obtain an RPT with the authorised permissions, the policy enforcement point can request a decision from Keycloak based on the access token presented by the client. This approach is useful when clients do not support the UMA Grant Flow but has the downside of introducing an HTTP call to Keycloak as part of the authorisation process.
