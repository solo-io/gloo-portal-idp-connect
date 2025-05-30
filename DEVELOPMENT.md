# Development

This is a development guide for users wanting to help contribute to the project.

## Environment

Before you begin, make sure you have all of the required tools installed:

```sh
./env/validate-env.sh
```

Add any new connector implementations to `cmd/idp-connect.go` so that they can become valid server options to start.

## Keycloak

You can test the manipulation of self-service clients using a dedicated realm in a Keycloak instance. Create a new realm using curl and the admin credentials using the examples below.

First, create a token to access the Keycloak REST API. This is a short-lived token, so you may need to repeat this step later on:

```sh
KEYCLOAK_URL=http://$(kubectl --context mgmt -n keycloak get service keycloak -o jsonpath='{.status.loadBalancer.ingress[0].*}'):8080

KEYCLOAK_TOKEN=$(curl -Ssm 10 --fail-with-body \
  -d "client_id=admin-cli" \
  -d "username=admin" \
  -d "password=admin" \
  -d "grant_type=password" \
  "$KEYCLOAK_URL/realms/master/protocol/openid-connect/token" |
  jq -r .access_token)
```

Create the new realm:

```sh
REALM=my-realm

curl -Ssm 10 --fail-with-body -H "Authorization: Bearer ${KEYCLOAK_TOKEN}" -H "Content-Type: application/json" \
  -d '{ "realm": "'${REALM}'", "enabled": true }' \
  $KEYCLOAK_URL/admin/realms
```

You'll need to provision a client in this realm that permits service accounts and has permissions to manipulate self-service clients. For convenience, we'll also treat this client as a _resource server_ in which we will store API products as _resources_. You can create such a client like this:

```sh
KEYCLOAK_CLIENT=gloo-portal

# Create initial token to register the client
INITIAL_TOKEN=$(curl -Ssm 10 --fail-with-body -H "Authorization: Bearer ${KEYCLOAK_TOKEN}" -H "Content-Type: application/json" \
  -d '{ "expiration": 0, "count": 1 }' \
  $KEYCLOAK_URL/admin/realms/${REALM}/clients-initial-access |
  jq -r .token)

# Register the client
read -r KEYCLOAK_CLIENT_INTERNAL_ID KEYCLOAK_SECRET <<<$(curl -Ssm 10 --fail-with-body -H "Authorization: bearer ${INITIAL_TOKEN}" -H "Content-Type: application/json" \
  -d '{ "clientId": "'${KEYCLOAK_CLIENT}'", "name": "Solo.io Gloo Portal Resource Server" }' \
  ${KEYCLOAK_URL}/realms/${REALM}/clients-registrations/default |
  jq -r '[.id, .secret] | @tsv')

echo "Management client ID: ${KEYCLOAK_CLIENT}"
echo "Management client secret: ${KEYCLOAK_SECRET}"

# Set up the client as we need
curl -Ssm 10 --fail-with-body -H "Authorization: Bearer ${KEYCLOAK_TOKEN}" -H "Content-Type: application/json" \
  -X PUT -d '{ "serviceAccountsEnabled": true, "authorizationServicesEnabled": true }' \
  ${KEYCLOAK_URL}/admin/realms/${REALM}/clients/${KEYCLOAK_CLIENT_INTERNAL_ID}

# Get the internal ID of the client's service account user
SA_USER_ID=$(curl -Ssm 10 --fail-with-body -H "Authorization: Bearer ${KEYCLOAK_TOKEN}" -H "Content-Type: application/json" \
  ${KEYCLOAK_URL}/admin/realms/${REALM}/clients/${KEYCLOAK_CLIENT_INTERNAL_ID}/service-account-user |
  jq -r .id)

# Get the ID of the 'realm-management' client
REALM_MGMT_CLIENT_ID=$(curl -Ssm 10 --fail-with-body -H "Authorization: Bearer ${KEYCLOAK_TOKEN}" \
  "${KEYCLOAK_URL}/admin/realms/${REALM}/clients?clientId=realm-management" |
  jq -r '.[].id')

# Get the ID of the 'manage-clients' role
ROLE_ID=$(curl -Ssm 10 --fail-with-body -H "Authorization: Bearer ${KEYCLOAK_TOKEN}" -H "Content-Type: application/json" \
  ${KEYCLOAK_URL}/admin/realms/${REALM}/users/${SA_USER_ID}/role-mappings/clients/${REALM_MGMT_CLIENT_ID}/available | jq -r '.[] | select(.name=="manage-clients") | .id')

# Add the 'manage-clients' role to the service account user
curl -Ssm 10 --fail-with-body -H "Authorization: Bearer ${KEYCLOAK_TOKEN}" -H "Content-Type: application/json" \
  -d '[ { "id": "'${ROLE_ID}'", "name": "manage-clients", "composite": false, "clientRole": true, "containerId": "'${REALM_MGMT_CLIENT_ID}'" } ]' \
  ${KEYCLOAK_URL}/admin/realms/${REALM}/users/${SA_USER_ID}/role-mappings/clients/${REALM_MGMT_CLIENT_ID}
```

The values of `KEYCLOAK_CLIENT` and `KEYCLOAK_SECRET` should be supplied to the Keycloak flavour of `idp-connect` at runtime (via `--client-id` and `--client-secret`) so that the service can obtain tokens and manipulate self-service clients on behalf of this management client. In the example used so far, you can start the service like this:

```sh
./idp-connect keycloak --issuer ${KEYCLOAK_URL}/realms/${REALM} --client-id ${KEYCLOAK_CLIENT} --client-secret ${KEYCLOAK_SECRET}
 ```

IDP Connect will use the token endpoint to obtain a token for the management client. You can replicate this for testing purposes like this:

```sh
MGMT_TOKEN=$(curl -Ssm 10 --fail-with-body \
  -u ${KEYCLOAK_CLIENT}:${KEYCLOAK_SECRET} \
  -d "grant_type=urn:ietf:params:oauth:grant-type:uma-ticket" \
  -d "audience=${KEYCLOAK_CLIENT}" \
  ${KEYCLOAK_URL}/realms/${REALM}/protocol/openid-connect/token |
  jq -r .access_token)

# Test the token by listing the clients in the realm
curl -Ssm 10 --fail-with-body -H "Authorization: Bearer ${MGMT_TOKEN}" ${KEYCLOAK_URL}/admin/realms/${REALM}/clients | jq .
```

## TODO

* Create middleware to handle login requests and responses and exposing those metrics via Prometheus metrics
* Cognito
  * Develop auth mechanism when Cognito is running in EKS, taking advantage of AWS IAM Role for Service Accounts
