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

To obtain an OAuth access-token from Cognito, we first need to determine its “token endpoint”. We can fetch that information from Cognito’s `.well-known/openid-configuration` endpoint:

```sh
export AWS_COGNITO_REGION={your cognito region}
export AWS_COGNITO_USER_POOL_ID={your cognito user-pool id}

curl https://cognito-idp.$AWS_COGNITO_REGION.amazonaws.com/$AWS_COGNITO_USER_POOL_ID/.well-known/openid-configuration
```

From the response, find the “token_endpoint”. We can now fetch a new access-token for our service account by using the client-credentials grant flow (note that in this example, we’re asking for 2 scopes, `access/catstronauts-api` and `access/tracks-rest-api` )

```yaml
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

```json
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

## Keycloak

### Set up the resource server, resources, clients and permissions

The base requirement for API Product Authorization with Keycloak is a realm and a resource server. In the Keycloak administation UI, perform the following steps to create these:

1. In the realm drop-down click **Create realm**. Give your realm a name and make sure it is **Enabled**
1. Within the realm, select _Clients_ and create a new client for the Gloo Portal. This will be the _resource server_ with which we associate API products as _resources_
   * Enable **Authorization** on this client, which allows it to act as a resource server

If not using the SPI and wanting to manually bootstrap Keycloak, follow these additional steps:

1. Under the client's _Authorization_ tab, select _Resources_ and then **Create resource**
   * Give the resource a **Name** that exactly matches the API Product ID it represents
   * Make sure **User-Managed access enabled** is selected
   * Create a resource for each API product being protected
1. Create a new _Client_ for each application
   * Enable **Direct access grants** so that the client can obtain tokens for users with their username and password
1. Back in the Gloo Portal client (resource server), authorise a test application (client) to one of the API products (resources)
   1. Click into the client's _Authorization_ tab, select _Policies_ and then **Create client policy**
   1. Select a policy type of "Client", give it the same name as the test application, and select the test application client from the list in the **Clients** field
   1. Save the new policy
   1. Return to the resource server's _Authorization_ tab, select _Resources_, find the API product you want to allow access to and click **Create permission** next to it
   1. Give the new permission a name, then select the policy you created in the previous step in the **Policies** field
   1. Save the new permission
1. Create a new _User_ for testing, and give it a password

### Obtaining an access token

To obtain an OAuth access token from Keycloak, we first need to determine its “token endpoint”. We can fetch that information from Keycloak's `.well-known/openid-configuration` endpoint:

```sh
KEYCLOAK_URL=<your Keycloak host and port>
REALM=<the realm you created>

TOKEN_ENDPOINT=$(curl http://${KEYCLOAK_URL}/realms/${REALM}/.well-known/openid-configuration | jq -r .token_endpoint)
```

We can now fetch a new access token for our test user and one of the client applications:

```sh
CLIENT_ID=<the name of the application client you selected>
CLIENT_SECRET=<the secret of the application client you selected>
USERNAME=<username of the test user>
PASSWORD=<password of the test user>

USER_TOKEN=$(curl ${TOKEN_ENDPOINT} \
  -d "client_id=${CLIENT_ID}" -d "client_secret=${CLIENT_SECRET}" \
  -d "username=${USERNAME}" -d "password=${PASSWORD}" \
  -d "grant_type=password" |
  jq -r .access_token)
```

You will be granted an access token (JWT), which decoded should look something like this:

```json
$ echo $USER_TOKEN | jwt decode -

Token header
------------
{
  "typ": "JWT",
  "alg": "RS256",
  "kid": "nZr3uOYfZT1tsdPqWYSfpJPykrlU6RMZNLcpGqH15DA"
}

Token claims
------------
{
  "acr": "1",
  "aud": "account",
  "azp": "${CLIENT_ID}",
  "email": "user1@solo.io",
  "email_verified": false,
  "exp": 1713443354,
  "family_name": "One",
  "given_name": "User",
  "iat": 1713443054,
  "iss": "${KEYCLOAK_URL}/realms/${REALM}",
  "jti": "e6c8494a-618a-44f5-9c40-503c503c42bd",
  "name": "User One",
  "preferred_username": "${USERNAME}",
  "realm_access": {
    "roles": [
      "offline_access",
      "uma_authorization",
      "default-roles-my-realm"
    ]
  },
  "resource_access": {
    "account": {
      "roles": [
        "manage-account",
        "manage-account-links",
        "view-profile"
      ]
    }
  },
  "scope": "profile email",
  "session_state": "2dd78f5d-172c-47a5-b131-2e562d7a6533",
  "sid": "2dd78f5d-172c-47a5-b131-2e562d7a6533",
  "sub": "521f0859-1353-4d4f-b4a2-4c2d4aa654f4",
  "typ": "Bearer"
}
```

Note that there are no scopes or permissions for the protected API products in this token. We'll be using Keycloak's Authorization Services, based on [User-Managed Access (UMA)](https://docs.kantarainitiative.org/uma/rec-uma-core.html), to subsequently authorise requests to API products. If you wish to obtain a _requesting party token_ (RPT) for this user to send with API calls, you can get one from the same endpoint using the given access token:

```sh
RESOURCE_SERVER_ID=<the name of the resource server you created>

USER1_RPT=$(curl ${TOKEN_ENDPOINT} \
  -H "Authorization: Bearer ${USER_TOKEN}" \
  -d "grant_type=urn:ietf:params:oauth:grant-type:uma-ticket" \
  -d "audience=${RESOURCE_SERVER_ID}" |
  jq -r .access_token)
```

## Configuring Gloo’s ExtAuthPolicy for your APIProduct/RouteTable

In Gloo Gateway, your RouteTable defines your API Product. Below is an example of such an API Product for the Tracks API:

```yaml
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

In this RouteTable, notice the “apiProductId” in the Portal metadata which matches the name of one of our scopes (i.e. access/tracks-rest-api). This facilitates authorization using OPA as this will be available to the rules when they are evaluated. Note that this label is just an example and can be named anything.

> **Note**: In order for the OPA policy to work, a dev portal must be enabled and an API Doc for the configured API Product must be generated. This is what triggers Gloo Platform to add the necessary context to allow the OPA policy to validate which API Product the request is targeting.

We can now create a ConfigMap with our OPA policies to only grant access to API Products when the access token corresponds to permissions to access the ApiProductId. How to implement this policy differs depending on the IDP in use:

### Cognito

Create an OPA policy config map as follows:

```yaml
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

Finally, we can apply the ExtAuthPolicy to our ApiProduct route(s) that performs JWT validation using the Cognito’s JSON Web Key Set (JWKS), which contains the public keys used to verify the validity of the token, and applies to OPA policy to perform Authorization checks:

```yaml
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

### Keycloak

Create a new OPA policy and store it in a ConfigMap. The policy below will check whether the access token is a _requesting party token_ (RPT) and, if it is, try to match the permissions against the requested API product. If the access token is not an RPT then the policy will call Keycloak directly to check if the user and client have permission to access the requested API product:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: oauth-scope-apiproduct-opa-cm
  namespace: gloo-mesh-addons
data:
  policy.rego: |-
    package test

    import future.keywords.if
    import future.keywords.in

    resource_server_id := "${RESOURCE_SERVER_ID}"

    # Get the requested API product from the metadata
    filter_metadata := input.check_request.attributes.metadata_context.filter_metadata
    api_product_id := filter_metadata["io.solo.gloo.apimanagement"].api_product_id

    default allow := false

    allow if not api_product_id

    allow if api_product_id == ""

    allow if authorised_by_rpt

    allow if authorised_by_keycloak

    # Check if the token is an RPT and includes a permission to the requested API product
    authorised_by_rpt if {
        input.state.jwtAccessToken.aud == resource_server_id
        some permission in input.state.jwtAccessToken.authorization.permissions
        permission.rsname == api_product_id
    }

    # Check if the user and client can access the API product with the authorisation server directly
    authorised_by_keycloak if {
        discovered_config := http.send({
            "url": concat("", [input.state.jwtAccessToken.iss, "/.well-known/uma2-configuration"]),
            "method": "GET",
            "force_cache": true,
            "force_cache_duration_seconds": 86400, # Cache response for 24 hours
        }).body

        authorisation_response := http.send({
            "url": discovered_config.token_endpoint,
            "method": "POST",
            "headers": {
                "Authorization": input.http_request.headers["authorization"],
                "Content-Type": "application/x-www-form-urlencoded",
            },
            "raw_body": sprintf("grant_type=urn:ietf:params:oauth:grant-type:uma-ticket&audience=%v&response_mode=decision&permission=%v", [resource_server_id, api_product_id]),
        })

        authorisation_response.body.result
    }
```

Finally, we can apply the ExtAuthPolicy to our ApiProduct route(s) that performs JWT validation using Keycloak’s JSON Web Key Set (JWKS), which contains the public keys used to verify the validity of the token, and applies to OPA policy to perform Authorization checks. Substitute the Keycloak URL and realm as needed:

```yaml
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
                url: http://${KEYCLOAK_URL}/realms/${REALM}/protocol/openid-connect/certs
      - opaAuth:
          modules:
          - name: oauth-scope-apiproduct-opa-cm
            namespace: gloo-mesh-addons
          query: "data.test.allow == true"
```

## Validation

When we now call our service with the access-token we fetched earlier, we can see that we can access our service:

```sh
export ACCESS_TOKEN={your access-token}

curl -v -H "Authorization: Bearer $ACCESS_TOKEN" http://api.example.com/trackapi/v1/tracks
```

You can validate that the OPA authorization works as expected by fetching a new access token that does not contain the scope or permission needed to access this service, and try to access the service with that token.
