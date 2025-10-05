package app

import "github.com/spf13/cobra"

func bindCommonFlags(cmd *cobra.Command, upgradeOpts *UpgradeOptions) {
	cmd.Flags().StringVar(&upgradeOpts.ClusterEndpoint, "cluster-endpoint", "", "Kubernetes API endpoint of the target cluster")
	cmd.Flags().BoolVar(&upgradeOpts.Airgapped, "airgapped", false, "Use offline assets from bundle")
	cmd.Flags().StringVar(&upgradeOpts.ValuesFile, "values-file", "", "Encrypted Helm values file path")
	cmd.Flags().StringVar(&upgradeOpts.ValuesPassphrase, "values-passphrase", "", "Passphrase for encrypted values")
	cmd.Flags().StringVar(&upgradeOpts.BundlePath, "bundle-path", "", "Path to local Helm bundle when operating offline")
	cmd.Flags().StringVar(&upgradeOpts.ChartReference, "chart", "", "OCI Helm chart reference (oci://registry/repo:tag)")
	cmd.Flags().StringVar(&upgradeOpts.ReleaseName, "release-name", "", "Helm release name override")
	cmd.Flags().StringVar(&upgradeOpts.AppVersion, "app-version", "", "Application version recorded in state")
	cmd.Flags().StringVar(&upgradeOpts.Namespace, "namespace", "", "Kubernetes namespace for the Helm release")
	cmd.Flags().StringVar(&upgradeOpts.StateFilePath, "state-file", "", "Absolute path for persisted state JSON")
	cmd.Flags().StringVar(&upgradeOpts.StateFileName, "state-file-name", "", "Custom state file name within the config directory")
	cmd.Flags().StringVar(&upgradeOpts.Output, "output", "text", "Output format: text or json")
}
