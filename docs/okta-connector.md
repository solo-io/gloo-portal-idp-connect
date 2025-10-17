# Okta IDP Connector

The Okta connector enables the gloo-portal-idp-connect service to manage OAuth applications in Okta for Gloo Portal users.

## Prerequisites

1. An Okta developer account or organization
2. An Okta API token with sufficient permissions to manage applications
3. Your Okta domain URL

## Setup

### 1. Create an Okta API Token

1. Log in to your Okta Admin Console
2. Go to **Security** > **API** > **Tokens**
3. Click **Create Token**
4. Enter a name for your token (e.g., "gloo-portal-idp-connect")
5. Click **Create Token** and save the token securely

### 2. Configure Required Permissions

The API token needs the following permissions:
- `okta.apps.manage` - To create and delete OAuth applications
- `okta.apps.read` - To search for existing applications

## Usage

### Running the Okta Connector Directly

```bash
go run ./cmd/idp-connect.go okta \
  --okta-domain https://your-domain.okta.com \
  --api-token your-api-token \
  --port 8080
```

### Deploying with Helm

1. **Create a values file** (e.g., `values-okta.yaml`):
```yaml
connector: okta
okta:
  domain: "https://dev-123456.okta.com"
  apiToken: "your-okta-api-token-here"
  secretName: okta-api
```

2. **Deploy using Helm**:
```bash
helm upgrade -i -n gloo-system \
  portal-idp gloo-portal-idp-connect/gloo-portal-idp-connect \
  --version 0.3.0 \
  -f values-okta.yaml
```

3. **Verify deployment**:
```bash
kubectl -n gloo-system rollout status deploy gloo-portal-idp-connect
```

### Configuration Parameters

- `--okta-domain`: Your Okta domain URL (e.g., `https://dev-123456.okta.com`)
- `--api-token`: Okta API token for application management (optional if `OKTA_API_TOKEN` env var is set)
- `--port`: HTTP server port (default: 8080)

### Environment Variables

You can also set configuration via environment variables:
- `OKTA_API_TOKEN`: API token (used if `--api-token` flag is not provided)

### Helm Configuration

When deploying with Helm, the following values are supported under the `okta` section:

| Parameter | Description | Required | Default |
|-----------|-------------|----------|---------|
| `domain` | Okta domain URL | Yes | - |
| `apiToken` | Okta API token | Yes | - |
| `secretName` | Name of secret to store API token | No | `okta-api` |

Example:
```yaml
connector: okta
okta:
  domain: "https://dev-123456.okta.com"
  apiToken: "00abc123def456789..."
  secretName: okta-api
```

## API Operations

The Okta connector implements the standard IDP Connect API:

### Create OAuth Application

**POST** `/applications`

Creates a new OAuth 2.0 Service Application in Okta with client credentials grant type.

### Delete OAuth Application

**DELETE** `/applications/{id}`

Deletes an OAuth application by searching for applications with the matching label and removing the found application.

## Application Configuration

The connector creates OAuth applications with the following Okta settings:
- **Application Type**: Service (for client credentials flow)
- **Grant Types**: `client_credentials`
- **Response Types**: `token`
- **Consent Method**: `TRUSTED`
- **Token Endpoint Auth Method**: `client_secret_basic`

## Security Considerations

- Store API tokens securely and rotate them regularly
- Use HTTPS in production
- Limit API token permissions to the minimum required scope
- Monitor Okta audit logs for application management activities

## Troubleshooting

### Common Issues

1. **Authentication Failed**: Verify your API token is valid and has sufficient permissions
2. **Domain Not Found**: Ensure your Okta domain URL is correct and accessible
3. **Application Creation Failed**: Check that your token has `okta.apps.manage` permission

### Logs

The connector provides detailed logging for debugging. Check the console output for HTTP request/response details and error messages.
