# vault-handler

`vault-handler` helps initialize and unseal Vault in situations where features like KMS auto-unseal or transit engine are either unavailable or not in-use.

The original inspiration for this app was to assist with bootstrapping Kubernetes clusters as part of `kubefirst`.
