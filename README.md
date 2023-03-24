# vault-handler

`vault-handler` helps initialize and unseal Vault in situations where features like KMS auto-unseal or transit engine are either unavailable or not in-use.

The original inspiration for this app was to assist with bootstrapping Kubernetes clusters as part of `kubefirst`.

## Components

This utility uses in-cluster Kubernetes configuration by default. It was built to run within a Kubernetes cluster.

There is an optional flag to allow running it locally and pointing at your own `kubeconfig` file.

At this time, it only supports running Vault using Raft storage with either 1 or 3 replicas.

## Usage

```bash
❯ vault-handler unseal -h
Unseal a vault instance

Usage:
  vault-handler unseal [flags]

Flags:
  -h, --help                        help for unseal
      --leader-only                 unseal only the raft leader - false (default) - true to only init and unseal vault-0
      --use-kubeconfig-in-cluster   kube config type - in-cluster (default), set to false to use local (default true)
```
