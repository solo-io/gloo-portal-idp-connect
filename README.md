# IDP Connect

IDP Connect is an implementation of the Service Programming Interface Gloo Platform Portal uses in order to manage client credentials for accessing services in your Kubernetes Cluster. In Gloo Platform Portal, we use the concept of "Applications" to refer to the external applications accessing the API Products exposed via your Gloo Portal. When a user registers an application as an OAuth client,
it is the responsibility of the SPI to create the credential associated with that application. For more information, and to review key terms associated with Gloo Platform Portal, checkout out our documentation: [Gloo Portal Documentation](https://docs.solo.io/gloo-portal/latest/).

## Supported Identity Providers

Here is a list of Identity Providers that we currently support:

* Amazon Cognito
* Keycloak

## Configuration Instructions

### Keycloak

The IDP Connect implementation for Keycloak is based on the [OpenID Connect Dynamic Client Registration 1.0](https://openid.net/specs/openid-connect-registration-1_0.html) (DCR) specification. DCR is supported by several IDPs, but this implementation has only been tested with Keycloak and is not guaranteed to work with other IDPs that have implemented DCR.

The following requirements must be met for the Keycloak IDP Connect service to work with a given Keycloak instance:

* *Trusted hosts:* You'll need to make a change to the realm settings in the Keycloak admin console to permit client registration requests from certain hosts. Under the "Clients" settings for your realm, click the "Client registration" tab then the "Trusted Hosts" policy. Here, either disable "Host Sending Client Registration Request Must Match" or add the hosts that client registration requests will originate from.

For a secure deployment, you should also consider the following:

* *Client Registration Policies:* By default, Keycloak will permit anonymous client registrations. However, you might wish to limit access to the client registration feature. A bearer token can be issued for a user or service account and provided to the Keycloak IDP Connect service. Any token provided to the IDP Connect servive must have `manage-client` permissions in Keycloak.

Please see <https://www.keycloak.org/docs/latest/securing_apps/#_client_registration> for more details of Keycloak's support for client registration.

## Production

IDP Connect provides a straightforward and easy-to-setup way of configuring credentials for the applications in your system; however,
 we expect that the needs of your system are and will evolve beyond the scope of this simple implementation. The SPI we provide provides a hook on top of which you can build a customizable system to service any number of more advanced use cases.

TODO: Add information for devs
* Install tools
* (Potential) Allow for AWS IAM Roles for service accounts as cognito auth method.