package verify

import (
	"context"
	"fmt"

	"github.com/tychonis/cyanotype/core/bomtree"
)

// QuantityChecker is a minimal example checker: every node needs a
// quantity greater than zero.
type QuantityChecker struct{}

func (QuantityChecker) Name() string {
	return "quantity"
}

func (QuantityChecker) Check(ctx context.Context, in *Input) ([]Result, error) {
	results := make([]Result, 0)
	err := Walk(in.Root, func(node *bomtree.Node) error {
		res := Result{Subject: NodePath(node), Status: StatusPass}
		if node.Qty <= 0 {
			res.Status = StatusFail
			res.Message = fmt.Sprintf("quantity %g, want > 0", node.Qty)
		}
		results = append(results, res)
		return ctx.Err()
	})
	return results, err
}
