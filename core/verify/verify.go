// Package verify runs checkers over an instantiated bom tree.
//
// A checker gets the tree and returns results (pass, fail, skip, error) for
// the nodes it looked at. Checkers can be written in Go and registered, or
// run as an external program (see ExecChecker), which is how outside solvers
// (geometry, FEA, test records, quotes) plug in.
//
// The package only reads bomtree.Node and model.Item. It does not depend on
// how contracts, items or coitems are parsed, so it can follow those changes
// instead of shaping them.
package verify

import (
	"context"
	"strings"

	"github.com/tychonis/cyanotype/core/bomtree"
)

type Status string

const (
	StatusPass  Status = "pass"
	StatusFail  Status = "fail"
	StatusSkip  Status = "skip"
	StatusError Status = "error"
)

// Result is one finding of one checker about one subject.
// Subject is a node path such as "robot/arm/bracket".
type Result struct {
	Checker string `json:"checker"`
	Subject string `json:"subject"`
	Status  Status `json:"status"`
	Message string `json:"message,omitempty"`
}

// Input is what a checker gets. It is a struct so it can grow
// (catalog, contracts, options) without changing the Checker interface.
type Input struct {
	Root *bomtree.Node
}

type Checker interface {
	Name() string
	Check(ctx context.Context, in *Input) ([]Result, error)
}

type Report struct {
	Results []Result `json:"results"`
}

// Failed reports whether any result is a fail or an error.
func (r *Report) Failed() bool {
	for _, res := range r.Results {
		if res.Status == StatusFail || res.Status == StatusError {
			return true
		}
	}
	return false
}

// NodePath joins the node names from the root down to node with "/".
func NodePath(node *bomtree.Node) string {
	names := make([]string, 0)
	for n := node; n != nil; n = n.Parent {
		names = append(names, n.Name)
	}
	for i, j := 0, len(names)-1; i < j; i, j = i+1, j-1 {
		names[i], names[j] = names[j], names[i]
	}
	return strings.Join(names, "/")
}

// Walk calls fn for node and every node below it, depth first.
func Walk(node *bomtree.Node, fn func(*bomtree.Node) error) error {
	if node == nil {
		return nil
	}
	err := fn(node)
	if err != nil {
		return err
	}
	for _, child := range node.Children {
		err = Walk(child, fn)
		if err != nil {
			return err
		}
	}
	return nil
}
