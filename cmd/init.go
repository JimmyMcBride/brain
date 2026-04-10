package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"brain/internal/config"
	"brain/internal/index"
	"brain/internal/output"
	"brain/internal/vault"

	"github.com/spf13/cobra"
)

func addInitCommand(root *cobra.Command, flags *rootFlagsState) {
	var vaultPath string
	var dataPath string
	var provider string
	var model string

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize config, vault structure, and local index",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, paths, err := config.LoadOrCreate(flags.configPath)
			if err != nil {
				return err
			}
			if vaultPath != "" {
				cfg.VaultPath = vaultPath
			}
			if dataPath != "" {
				cfg.DataPath = dataPath
			}
			if provider != "" {
				cfg.EmbeddingProvider = provider
			}
			if model != "" {
				cfg.EmbeddingModel = model
			}
			cfgPaths := config.BuildPaths(cfg, paths.ConfigFile)
			if err := config.Save(cfg, cfgPaths.ConfigFile); err != nil {
				return err
			}
			for _, dir := range []string{cfgPaths.DataDir, cfgPaths.BackupDir, cfgPaths.IndexDir} {
				if err := os.MkdirAll(dir, 0o755); err != nil {
					return err
				}
			}

			vaultSvc := vault.New(cfg)
			if err := vaultSvc.Initialize(); err != nil {
				return err
			}
			store, err := index.New(cfgPaths.DBFile)
			if err != nil {
				return err
			}
			defer store.Close()

			printer := output.New(modeFromFlag(flags, cfg.OutputMode), cmd.OutOrStdout())
			payload := map[string]any{
				"config_file": cfgPaths.ConfigFile,
				"vault_path":  cfg.VaultPath,
				"data_path":   cfg.DataPath,
				"para_dirs":   vault.PARASections(),
				"db_file":     cfgPaths.DBFile,
			}
			return printer.Print(payload, func(w io.Writer) error {
				if err := output.KeyValue(w, "Config", cfgPaths.ConfigFile); err != nil {
					return err
				}
				if err := output.KeyValue(w, "Vault", cfg.VaultPath); err != nil {
					return err
				}
				if err := output.KeyValue(w, "Data", cfg.DataPath); err != nil {
					return err
				}
				if err := output.KeyValue(w, "SQLite", cfgPaths.DBFile); err != nil {
					return err
				}
				_, err := fmt.Fprintf(w, "PARA:          %s\n", strings.Join(vault.PARASections(), ", "))
				return err
			})
		},
	}

	cmd.Flags().StringVar(&vaultPath, "vault", filepath.Join(userHomeDir(), "Documents", "brain"), "vault path")
	cmd.Flags().StringVar(&dataPath, "data", "", "data path for sqlite, logs, and backups")
	cmd.Flags().StringVar(&provider, "embedding-provider", "", "embedding provider (localhash, openai, none)")
	cmd.Flags().StringVar(&model, "embedding-model", "", "embedding model name")
	root.AddCommand(cmd)
}
