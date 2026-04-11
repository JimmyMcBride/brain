package cmd

import (
	"fmt"
	"io"

	"brain/internal/buildinfo"
	"brain/internal/config"
	"brain/internal/output"

	"github.com/spf13/cobra"
)

func addVersionCommand(root *cobra.Command, flags *rootFlagsState) {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show the current brain build version",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _, err := config.LoadOrCreate(flags.configPath)
			if err != nil {
				return err
			}
			info := buildinfo.Current()
			printer := output.New(modeFromFlag(flags, cfg.OutputMode), cmd.OutOrStdout())
			return printer.Print(info, func(w io.Writer) error {
				if _, err := fmt.Fprintf(w, "version: %s\n", info.Version); err != nil {
					return err
				}
				if _, err := fmt.Fprintf(w, "commit:  %s\n", info.Commit); err != nil {
					return err
				}
				if _, err := fmt.Fprintf(w, "date:    %s\n", info.Date); err != nil {
					return err
				}
				_, err := fmt.Fprintf(w, "path:    %s\n", info.Path)
				return err
			})
		},
	}
	root.AddCommand(cmd)
}
