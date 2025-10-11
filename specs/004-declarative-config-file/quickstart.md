# Quickstart: Declarative YAML Configuration for chainctl

## Prerequisites
- Checkout branch `004-declarative-config-file` and build: `go build ./cmd/chainctl`
- Ensure Helm/cluster credentials already configured for target commands
- Create a writable working directory to store `chainctl.yaml`

## Step 1 — Create Base Configuration
1. Create `chainctl.yaml` in your working directory with the structure below:
   ```yaml
   metadata:
     name: demo-shared-config
     description: Shared defaults for demo installs
   defaults:
     kubeconfig: ~/.kube/config
     namespace: demo
   commands:
     chainctl cluster install:
       flags:
         bundle-path: ./artifacts/cluster-bundle.yaml
         dry-run: true
   ```
2. Save the file and ensure it contains **no secrets** (tokens, passwords, kubeconfigs with embedded credentials).

## Step 2 — Run chainctl with Declarative Config
1. Execute `./chainctl cluster install --config ./chainctl.yaml`.
2. Confirm the CLI prints the resolved configuration summary showing:
   - Source file path
   - Applied defaults and command-specific flags
   - Confirmation that `dry-run` originated from YAML
3. Verify the command executes end-to-end without prompting for missing flags.

## Step 3 — Override at Runtime
1. Override a flag to validate precedence:  
   `./chainctl cluster install --config ./chainctl.yaml --namespace demo-override`
2. Inspect the summary to ensure `namespace` reflects the runtime override while other values still come from YAML.
3. Repeat with `CHAINCTL_CONFIG=~/shared/chainctl.yaml ./chainctl cluster install` to confirm environment-variable discovery.

## Step 4 — Validation Failure Exercise
1. Add an unsupported flag (e.g., `unknown-flag: true`) under `chainctl cluster install.flags`.
2. Rerun the command and observe the actionable error reporting the invalid key and suggested correction.
3. Remove the invalid flag and rerun to confirm success.

## Step 5 — Multi-Command Defaults
1. Expand the YAML to include an application command:
   ```yaml
   profiles:
     staging:
       namespace: staging
   commands:
     chainctl app install:
       profiles:
         - staging
       flags:
         chart: oci://registry.example.com/apps/demo:1.2.3
         release-name: demo-staging
   ```
2. Execute `./chainctl app install --config ./chainctl.yaml --dry-run`.
3. Confirm the summary lists `profiles: [staging]` and that `namespace` comes from the shared profile.

## Cleanup
- Remove sample configuration files when finished: `rm chainctl.yaml`
- Unset `CHAINCTL_CONFIG` if you exported it: `unset CHAINCTL_CONFIG`
