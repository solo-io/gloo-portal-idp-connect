## Use with Gloo Platform

The idea behind the Service Programming Interface is to provide an interopability layer between Gloo Platform Portal and the IDP that the customer wants to use. In order to get the desired functionality, there are two flows that need to be implemented: IDP Configuration and Data Path Authorization.

### AWS Cognito Example

#### IDP Configuration
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

#### Data Path Authorization

![Obtaining Access Token](./images/retrieve-credentials.png)

Cognito's token endpoint can be used to retrieve the access token. See [Configuring Gloo Platform](./configuring-gloo-platform.md) for more information on how to use the access token to authorize requests to your API Products.

![Data Path](./images/data-path.png)

The Access token can be used to authorize requests via ext-auth. The most convenient method would be to use OPA in order to match the scope of the access token against the `apiProductId` of the API Product. See [Configuring Gloo Platform](./configuring-gloo-platform.md) for more information on how to use the access token to authorize requests to your API Products.

