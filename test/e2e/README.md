# test/e2e

End-to-end CLI smoke tests exercising the high-level `chainctl` workflows against
ephemeral clusters or mocked dependencies. The suite is intentionally light on
infrastructure assumptions so it can validate CLI wiring, telemetry output, and
state persistence logic.

## Running locally

```bash
CHAINCTL_E2E=1 \
CHAINCTL_E2E_SUDO=1 \
CHAINCTL_VALUES_FILE=test/e2e/testdata/values.enc \
CHAINCTL_VALUES_PASSPHRASE=secret \
go test ./test/e2e/...
```

### Required environment variables

- `CHAINCTL_E2E=1`: opt-in flag so the suite skips by default.
- `CHAINCTL_E2E_SUDO=1`: allows tests to re-exec commands with `sudo -E go …`
  so host preflight checks (kernel modules, sudo privileges) succeed.
- `CHAINCTL_VALUES_FILE`, `CHAINCTL_VALUES_PASSPHRASE`: point at the encrypted
  values used by cluster/app commands (defaults point to repo fixtures).

### Optional environment variables

- `CHAINCTL_E2E_CLUSTER_REUSE=1`: exercises `cluster install` in reuse mode. A
  valid `CHAINCTL_CLUSTER_ENDPOINT` and kubeconfig are required when enabled.
- `CHAINCTL_CLUSTER_ENDPOINT`: overrides the default `https://cluster.local`
  endpoint used by app upgrade and cluster tests.
- `CHAINCTL_JOIN_TOKEN`: composite `id.secret` token for `node join`. When not
  supplied, the test is skipped. Provide `CHAINCTL_E2E_FORCE_NODE_JOIN=1` to
  run against the in-memory token store without a kubeconfig (for diagnostics).
- `KUBECONFIG`: required to validate cluster reuse scenarios and to avoid
  skipping `node join` when using the Kubernetes-backed token store.

Most tests create temporary bundles and state directories to avoid requiring
external registries. The install/upgrade quickstart scenario persists state and
validates JSON output end-to-end.

## CI behaviour

GitHub Actions runs unit and integration tests in `build-and-test`, and executes
the e2e suite in a dedicated `e2e-tests` job. The e2e job loads the `overlay`
and `br_netfilter` kernel modules, sets the environment above, and leverages the
sudo-aware helpers added in this package.

Cluster and node scenarios still skip when real infrastructure is not present;
the quickstart app install/upgrade path always runs.
