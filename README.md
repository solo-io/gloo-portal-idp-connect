# IDP Connect

IDP Connect is an implementation of the Service Programming Interface Gloo Platform Portal uses in order to manage client credentials for accessing services in your Kubernetes Cluster. In Gloo Platform Portal, we use the concept of "Applications" to refer to the external applications accessing the API Products exposed via your Gloo Portal. When a user registers an application as an OAuth client,
it is the responsibility of the SPI to create the credential associated with that application. For more information, and to review key terms associated with Gloo Platform Portal, checkout out our documentation: [Gloo Portal Documentation](https://docs.solo.io/gloo-portal/latest/).

## Supported Identity Providers

Here is a list of Identity Providers that we currently support:

* Amazon Cognito

## Production

IDP Connect provides a straightforward and easy-to-setup way of configuring credentials for the applications in your system; however,
 we expect that the needs of your system are and will evolve beyond the scope of this simple implementation. The SPI we provide provides a hook on top of which you can build a customizable system to service any number of more advanced use cases.