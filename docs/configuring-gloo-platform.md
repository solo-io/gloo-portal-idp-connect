# OPA-based API Product Authorization with OAuth

## AWS Cognito

Below are the instructions for using AWS Cognito to manage client credentials and access tokens for your API Products and configure Gloo Platform to authorize them.

### Manual Bootstrap

If not using the SPI and wanting to manually bootstrap AWS Cognito, follow these steps:

1. Create a UserPool
1. Within the UserPool, go to “App Integration”
1. Create one or more “Resource Servers”, and for these Resource Servers, create scopes that map to the “apiProductId” (in this example, `tracks-rest-api` and `catstronauts-api`).
1. Create an “App client”, and configure it so it:
   1. Has a client-id and client-secret
   1. Can be used for “client credentials grant” OAuth flows. 
   1. This can be configured in the “OAuth 2 Grant Types” section in app-client’s “Edit Hosted UI” submenu.
   1. In the same “Edit Hosted UI” submenu, give the app client access to one or more of the scopes we’ve configured earlier.


### Obtaining an access-token
To obtain an OAuth access-token from Cognito, we first need to determine it’s “token endpoint”. We can fetch that information from Cognito’s “.well-known/openid-configuration” endpoint:

```
export AWS_COGNITO_REGION={your cognito region}
export AWS_COGNITO_USER_POOL_ID={your cognito user-pool id}

curl https://cognito-idp.$AWS_COGNITO_REGION.amazonaws.com/$AWS_COGNITO_USER_POOL_ID/.well-known/openid-configuration
```

From the response, find the “token_endpoint”. We can now fetch a new access-token for our service account by using the client-credentials grant flow (note that in this example, we’re asking for 2 scopes, `access/catstronauts-api` and `access/tracks-rest-api` )

```
export TOKEN_ENDPOINT={your cognito token endpoint}
export CLIENT_ID={your service account’s client-id}
export CLIENT_SECRET={your service account’s client-secret}

curl -X POST -H "Accept: application/json" -H "Content-Type: application/x-www-form-urlencoded" \
--data-urlencode "client_id=$CLIENT_ID" \
--data-urlencode "client_secret=$CLIENT_SECRET" \
--data-urlencode "grant_type=client_credentials" \
--data-urlencode "scope=access/catstronauts-api access/tracks-rest-api" \
$TOKEN_ENDPOINT
```

You will be granted an access-token (JWT), which decoded should look something like this:

```
[
    {
        "kid": "toHqwIJt3ahSc6BbWdpabY6Han4psIwSbVfrI1Jod6I=",
        "alg": "RS256"
    },
    {
        "sub": "s2ai3kk4j6vfun6po9747bi1g",
        "token_use": "access",
        "scope": "access/catstronauts-api tracks-rest-api",
        "auth_time": 1699993949,
        "iss": "https://cognito-idp.eu-north-1.amazonaws.com/eu-north-1_6GQrqVZAY",
        "exp": 1699997549,
        "iat": 1699993949,
        "version": 2,
        "jti": "44da9501-26b1-4dc7-8efb-235df5fc281d",
        "client_id": "s1ai3kk5j6vfqn6pf9835bi1r"
    }
]
```

Configuring the Gloo’s ExtAuthPolicy for your APIProduct/RouteTable

In Gloo Gateway, your RouteTable defines your API Product. Below is an example of such an API Product for the Tracks API:

```
apiVersion: networking.gloo.solo.io/v2
kind: RouteTable
metadata:
  name: v1-tracks-rt
  namespace: istio-gateway-ns
  annotations:
    cluster.solo.io/cluster: cluster-1
  labels:
    portal: tracks-portal
    api: tracks
spec:
  hosts:
    - "*"
  virtualGateways:
    - name: cluster-1-north-south-gw-443
      namespace: istio-gateway-ns
  portalMetadata:
    apiProductDisplayName: tracks REST API
    apiProductId: tracks-rest-api
    apiVersion: v1
    title: tracks v1 REST API
    description: V1 REST API for tracks to retrieve data for tracks, authors and modules.
    contact: example@solo.io
    license: MIT
    termsOfService: sample terms of service
    lifecycle: development
    customMetadata:
      compatibility: backwards
  http:
    - name: tracks-api-v1
      labels:
        apiProduct: tracks-v1
      matchers:
        - uri:
            prefix: /trackapi/v1/
      forwardTo:
        pathRewrite: /
        destinations:
          - ref:
              name: tracks-rest-api
              namespace: tracks
            port:
              number: 5000
```


In this RouteTable, notice the “apiProductId” in the Portal metadata which matches the name of one of our scopes (i.e. access/tracks-rest-api). This enables the authorization based on scopes using OPA. Note that this label is just an example and can be named anything.

We can now create a ConfigMap with our OPA policies to only grant access to API Products when the access-token has a scope that matches the ApiProductId:

```
apiVersion: v1
kind: ConfigMap
metadata:
  name: oauth-scope-apiproduct-opa-cm
  namespace: gloo-mesh-addons
  annotations:
    cluster.solo.io/cluster: cluster-1
data:
  policy.rego: |-
    package test

    default allow = false

    allow {
        # Get the accessed ApiProductId from the metadata
        filter_metadata := input.check_request.attributes.metadata_context.filter_metadata
        apimanagement_metadata := filter_metadata["io.solo.gloo.apimanagement"]
        api_product_id := apimanagement_metadata.api_product_id

        # Get the scopes from the access-token
        scopes := split(input.state.jwtAccessToken.scope, " ")
        # Split scopes to remove resource server prefix
        scopeComponents := split(scopes[_], "/")

        scope := scopeComponents[1]

        # Ensure apiproduct and scopes are not empty
        api_product_id != ""
        scope != ""

        # Validate that we have a scope for this API Product
        scope == api_product_id
    }
```

**Note**: In order for the OPA policy to work, a dev portal must be enabled and an API Doc for the configured API Product must be generated. This is what triggers Gloo Platform to add the necessary context to allow the OPA policy to validate which API Product the request is targeting.

Finally, we can apply the ExtAuthPolicy to our ApiProduct route(s) that performs JWT validation using the Cognito’s JSON Web Key Set (JWKS), which contains the public keys used  to verify the validity of the token, and applies to OPA policy to perform Authorization checks:

```
apiVersion: security.policy.gloo.solo.io/v2
kind: ExtAuthPolicy
metadata:
  annotations:
    cluster.solo.io/cluster: cluster-1
  name: tracks-v2-oauth
  namespace: gloo-mesh-addons
spec:
  applyToRoutes:
  - route:
      labels:
        apiProduct: tracks-v1
  config:
    server:
      name: ext-auth-server
      namespace: gloo-mesh-addons
      cluster: cluster-1
    glooAuth:
      configs:
      - oauth2:
          accessTokenValidation:
            jwt:
              remote_jwks:
                url: https://cognito-idp.us-west-2.amazonaws.com/us-west-2_CngONp9kI/.well-known/jwks.json
      - opaAuth:
          modules:
          - name: oauth-scope-apiproduct-opa-cm
            namespace: gloo-mesh-addons
          query: "data.test.allow == true"
```

Note that the url for remoteJwks url might be different for your Cognito instances. The location can also be found from Cognito’s `.well-known/openid-configuration` endpoint, from which we also fetched the token-endpoint earlier.

When we now call our service with the access-token we fetched earlier from Cognito, we can see that we can access our service:

```
export ACCESS_TOKEN={your cognito access-token}

curl -v -H "Authorization: Bearer $ACCESS_TOKEN" http://api.example.com/trackapi/v1/tracks
```

You can validate that the OPA authorization works as expected by fetching a new access-token from Cognito that does not contain the scope needed to access this service and try to access the service with that token.
