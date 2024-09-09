# IDP Connect

IDP Connect is an implementation of the Service Programming Interface Gloo Gateway Portal uses in order to manage client credentials for accessing services in your Kubernetes Cluster. In Gloo Gateway Portal, we use the concept of "Applications" to refer to the external applications accessing the API Products exposed via your Gloo Portal.
Each Application may have multiple OAuth credentials, and it is the responsibility of the SPI to create and manage the credential(s) associated with applications. For more information and a review of key terms related to Gloo Gateway Portal, refer to our documentation: [Gloo Portal Documentation](https://docs.solo.io/gloo-portal/latest/).

## Supported Identity Providers

Here is a list of Identity Providers that we currently support:

* Amazon Cognito
* Keycloak

## Configuration Instructions

### Keycloak

A Keycloak client must be created for the Keycloak IDP Connect service to use. Provide the ID and secret of this client in the `--client-id` and `--client-secret` IDP Connect arguments respectively. This client must meet some requirements:

* The client must have the `manage-client` permission needed for IDP Connect to be able to manipulate self-service clients.
* **Authorization** must be enabled on this client, as this client will also act as an OAuth2 [resource server](https://www.keycloak.org/docs/latest/authorization_services/index.html#_resource_server_overview).
* **Service accounts roles** (or OAuth2 _client credentials_) must be enabled, to allow IDP Connect to use this client directly to manage other clients and resources.

#### Related documentation

* Keycloak's support for client registration: <https://www.keycloak.org/docs/latest/securing_apps/#_client_registration>
* Resource authorization in Keycloak: <https://www.keycloak.org/docs/latest/authorization_services/>
* IDP Connect will manipulate resources using Keycloak's Authorization Services, which is based on [User-Managed Access (UMA)](https://docs.kantarainitiative.org/uma/rec-uma-core.html)

## Production

IDP Connect provides a straightforward and easy-to-setup way of configuring credentials for the applications in your system; however,
 we expect that the needs of your system are and will evolve beyond the scope of this simple implementation. The SPI we provide provides a hook on top of which you can build a customizable system to service any number of more advanced use cases.

TODO: Add information for devs

* Install tools
* (Potential) Allow for AWS IAM Roles for service accounts as cognito auth method.
