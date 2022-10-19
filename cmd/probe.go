package cmd

import (
	"errors"

	"github.com/spf13/cobra"
	"github.com/topolvm/csi-driver-availability-monitor/probe"
)

var probeCmd = &cobra.Command{
	Use: "probe",
	RunE: func(cmd *cobra.Command, args []string) error {
		if probeConfig.nodeName == "" {
			return errors.New("no node name specified")
		}
		if probeConfig.storageClass == "" {
			return errors.New("no Storage Class specified")
		}

		return probe.SubMain(
			probeConfig.nodeName,
			probeConfig.fioFilename,
			probeConfig.storageClass,
			probeConfig.controllerAddr)
	},
}

var probeConfig struct {
	controllerAddr string
	storageClass   string
	fioFilename    string
	nodeName       string
}

func init() {
	fs := probeCmd.Flags()
	fs.StringVar(&probeConfig.controllerAddr, "destination-address", "http://localhost:8080", "metrics aggregator's address")
	fs.StringVar(&probeConfig.storageClass, "storage-class", "", "target StorageClass name")
	fs.StringVar(&probeConfig.fioFilename, "path", "/test", "target I/O test directory path")
	fs.StringVar(&probeConfig.nodeName, "node-name", "", "node name")

	rootCmd.AddCommand(probeCmd)
}
