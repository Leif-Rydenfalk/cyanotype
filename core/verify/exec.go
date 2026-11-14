package verify

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/tychonis/cyanotype/core/bomtree"
	"github.com/tychonis/cyanotype/model"
)

// ExecProtocolVersion is the version of the stdin/stdout format below.
// It is a first draft and expected to change.
const ExecProtocolVersion = 1

// ExecRequest is written as JSON to the external checker's stdin.
type ExecRequest struct {
	Version int       `json:"version"`
	Root    *ExecNode `json:"root"`
}

// ExecNode is a plain copy of a bom tree node for external programs.
type ExecNode struct {
	Name          string      `json:"name"`
	Path          string      `json:"path"`
	Qty           float64     `json:"qty"`
	ItemQualifier string      `json:"item_qualifier,omitempty"`
	ItemDigest    string      `json:"item_digest,omitempty"`
	Item          *model.Item `json:"item,omitempty"`
	Children      []*ExecNode `json:"children"`
}

// ExecResponse is read as JSON from the external checker's stdout.
// The checker field of each result is filled in by cyanotype.
type ExecResponse struct {
	Results []Result `json:"results"`
}

// ExecChecker runs an external program as a checker. The program gets an
// ExecRequest on stdin and answers with an ExecResponse on stdout. A non-zero
// exit status means the checker itself failed (not the bom); that becomes a
// StatusError result.
type ExecChecker struct {
	CheckerName string
	Command     []string
}

func (c *ExecChecker) Name() string {
	return c.CheckerName
}

func (c *ExecChecker) Check(ctx context.Context, in *Input) ([]Result, error) {
	if len(c.Command) == 0 {
		return nil, errors.New("no command")
	}
	req := &ExecRequest{
		Version: ExecProtocolVersion,
		Root:    NewExecNode(in.Root),
	}
	payload, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, c.Command[0], c.Command[1:]...)
	cmd.Stdin = bytes.NewReader(payload)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg != "" {
			return nil, fmt.Errorf("%w: %s", err, msg)
		}
		return nil, err
	}

	var resp ExecResponse
	err = json.Unmarshal(stdout.Bytes(), &resp)
	if err != nil {
		return nil, fmt.Errorf("invalid checker output: %w", err)
	}
	return resp.Results, nil
}

// NewExecNode copies node and everything below it.
func NewExecNode(node *bomtree.Node) *ExecNode {
	if node == nil {
		return nil
	}
	en := &ExecNode{
		Name:     node.Name,
		Path:     NodePath(node),
		Qty:      node.Qty,
		Item:     node.Item,
		Children: make([]*ExecNode, 0, len(node.Children)),
	}
	if node.Item != nil {
		en.ItemQualifier = node.Item.Qualifier
		en.ItemDigest = node.Item.Digest
	}
	for _, child := range node.Children {
		en.Children = append(en.Children, NewExecNode(child))
	}
	return en
}
