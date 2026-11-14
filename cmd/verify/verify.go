package verify

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/spf13/cobra"

	"github.com/tychonis/cyanotype/core/catalog"
	"github.com/tychonis/cyanotype/core/instantiator"
	"github.com/tychonis/cyanotype/core/parser/hcl"
	"github.com/tychonis/cyanotype/core/ranker"
	"github.com/tychonis/cyanotype/core/verify"
)

var Cmd = &cobra.Command{
	Use:   "verify <root>",
	Short: "Run checkers over the bom tree of root",
	Long: `Run checkers over the bom tree of root.

Built-in checkers are registered in core/verify. External checkers are
programs that read the tree as JSON on stdin and write results as JSON on
stdout, added with --exec name=command (see core/verify/README.md).`,
	RunE: run,
	Args: cobra.MaximumNArgs(1),
}

var (
	checkers []string
	execs    []string
	asJSON   bool
	list     bool
)

func init() {
	Cmd.Flags().StringSliceVarP(&checkers, "checker", "c", nil, "run only these checkers (default: all)")
	Cmd.Flags().StringArrayVar(&execs, "exec", nil, "add an external checker, name=command")
	Cmd.Flags().BoolVar(&asJSON, "json", false, "print the report as JSON")
	Cmd.Flags().BoolVar(&list, "list", false, "list registered checkers and exit")

	_ = verify.Register(verify.QuantityChecker{})
}

func run(cmd *cobra.Command, args []string) error {
	reg := verify.Default
	for _, e := range execs {
		c, err := parseExec(e)
		if err != nil {
			return err
		}
		err = reg.Register(c)
		if err != nil {
			return err
		}
	}

	if list {
		for _, name := range reg.Names() {
			fmt.Fprintln(cmd.OutOrStdout(), name)
		}
		return nil
	}
	if len(args) != 1 {
		return errors.New("verify needs a root")
	}

	p := hcl.NewParser()
	err := p.Build(".")
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

	ins := instantiator.New(ranker.NewCatalogRanker(cat))
	root, err := ins.TreeFromQualifier(cat, args[0])
	if err != nil {
		slog.Error("Failed to build.", "error", err)
		return err
	}

	report, err := reg.Run(cmd.Context(), &verify.Input{Root: root}, checkers)
	if err != nil {
		return err
	}

	if asJSON {
		out, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(out))
	} else {
		for _, res := range report.Results {
			fmt.Fprintln(cmd.OutOrStdout(), formatResult(res))
		}
	}

	if report.Failed() {
		return errors.New("verification failed")
	}
	return nil
}

func parseExec(s string) (*verify.ExecChecker, error) {
	name, command, ok := strings.Cut(s, "=")
	fields := strings.Fields(command)
	if !ok || name == "" || len(fields) == 0 {
		return nil, fmt.Errorf("invalid --exec %q, want name=command", s)
	}
	return &verify.ExecChecker{CheckerName: name, Command: fields}, nil
}

func formatResult(res verify.Result) string {
	line := fmt.Sprintf("%-5s %-12s %s", strings.ToUpper(string(res.Status)), res.Checker, res.Subject)
	if res.Message != "" {
		line += "  " + res.Message
	}
	return line
}
