/*
Package secret provides interfaces to interact with ancillary secrets management
services that integrate with containers.

SecretsManager provides an abstraction to interact with a Vault backed by
Secrets Manager without needing to make direct calls to the API to perform
frequently-used operations.

The SecretsManagerClient interface provides a convenience wrapper around the
Secrets Manager API. If the Vault interface does not fulfill your needs, you can
make calls directly to the Secrets Manager API instead.
*/
package secret
