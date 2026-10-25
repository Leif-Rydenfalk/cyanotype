package tree

import (
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"github.com/tychonis/cyanotype/core/catalog"
	"github.com/tychonis/cyanotype/core/instantiator"
	"github.com/tychonis/cyanotype/core/parser/hcl"
	"github.com/tychonis/cyanotype/core/ranker"
)

var Cmd = &cobra.Command{
	Use:   "tree <root>",
	Short: "Build bom tree from bpo",
	RunE:  run,
	Args:  cobra.ExactArgs(1),
}

func init() {
	// TODO: distinguish from output format
	Cmd.Flags().StringP("output", "o", "", "set output path")
}

func run(cmd *cobra.Command, args []string) error {
	bpoPath := "."
	root := args[0]

	bpcPath := cmd.Flag("output").Value.String()
	if bpcPath == "" {
		bpcPath = root + ".bpc"
	}
	p := hcl.NewParser()
	err := p.Build(bpoPath)
	if err != nil {
		slog.Warn("Failed to parse bpo.", "error", err)
		return err
	}

	cat := catalog.New("memory")
	err = p.Commit(cat)
	if err != nil {
		slog.Error("Failed to commit to catalog.", "error", err)
		return err
	}

	r := ranker.NewCatalogRanker(cat)
	ins := instantiator.New(r)
	rootNode, err := ins.TreeFromQualifier(cat, root)
	if err != nil {
		slog.Error("Failed to build.", "error", err)
		return err
	}

	output, err := rootNode.Export()
	if err != nil {
		slog.Error("Failed to export.", "error", err)
		return err
	}
	return os.WriteFile(bpcPath, append(output, '\n'), 0o644)
}
