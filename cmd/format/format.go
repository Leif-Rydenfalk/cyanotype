package format

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/spf13/cobra"
)

var Cmd = &cobra.Command{
	Use:   "fmt [path ...]",
	Short: "Format BPO source files",
	Args:  cobra.ArbitraryArgs,
	RunE:  run,
}

func run(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		args = []string{"."}
	}

	for _, path := range args {
		if err := filepath.WalkDir(path, func(name string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				if entry.Name() == ".git" || entry.Name() == ".bpc" {
					return filepath.SkipDir
				}
				return nil
			}
			if filepath.Ext(name) != ".bpo" && name != path {
				return nil
			}
			return formatFile(cmd, name)
		}); err != nil {
			return err
		}
	}
	return nil
}

func formatFile(cmd *cobra.Command, path string) error {
	src, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	formatted := hclwrite.Format(src)
	if bytes.Equal(src, formatted) {
		return nil
	}

	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, formatted, info.Mode().Perm()); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}

	cmd.Println(path)
	return nil
}
