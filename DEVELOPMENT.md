## Development

This is a development guide for users wanting to help contribute to the project.

## Environment

Before you begin, make sure you have all of the required tools installed:

```
./env/validate-env.sh
```

## TODO:

* Create middleware to handle login requests and responses and exposing those metrics via Prometheus metrics
* Cognito
  * Develop auth mechanism when Cognito is running in EKS, taking advantage of AWS IAM Role for Service Accounts