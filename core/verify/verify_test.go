package verify_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/tychonis/cyanotype/core/bomtree"
	"github.com/tychonis/cyanotype/core/verify"
)

func sampleTree() *bomtree.Node {
	root := &bomtree.Node{Name: "robot", Qty: 1}
	arm := &bomtree.Node{Name: "arm", Qty: 2, Parent: root}
	bracket := &bomtree.Node{Name: "bracket", Qty: 0, Parent: arm}
	arm.Children = []*bomtree.Node{bracket}
	root.Children = []*bomtree.Node{arm}
	return root
}

type stubChecker struct {
	name    string
	results []verify.Result
	err     error
}

func (s *stubChecker) Name() string { return s.name }

func (s *stubChecker) Check(ctx context.Context, in *verify.Input) ([]verify.Result, error) {
	return s.results, s.err
}

func TestNodePath(t *testing.T) {
	bracket := sampleTree().Children[0].Children[0]
	if got := verify.NodePath(bracket); got != "robot/arm/bracket" {
		t.Fatalf("want robot/arm/bracket, got %s", got)
	}
}

func TestRegistry_Register(t *testing.T) {
	r := verify.NewRegistry()
	if err := r.Register(&stubChecker{name: "b"}); err != nil {
		t.Fatalf("register: %v", err)
	}
	if err := r.Register(&stubChecker{name: "a"}); err != nil {
		t.Fatalf("register: %v", err)
	}
	if err := r.Register(&stubChecker{name: "a"}); err == nil {
		t.Fatal("want error on duplicate name")
	}
	if err := r.Register(&stubChecker{}); err == nil {
		t.Fatal("want error on empty name")
	}
	if got := r.Names(); !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Fatalf("want [a b], got %v", got)
	}
}

func TestRegistry_Run(t *testing.T) {
	r := verify.NewRegistry()
	_ = r.Register(&stubChecker{name: "ok", results: []verify.Result{{Subject: "robot", Status: verify.StatusPass}}})
	_ = r.Register(&stubChecker{name: "broken", err: errors.New("solver not found")})
	in := &verify.Input{Root: sampleTree()}

	report, err := r.Run(context.Background(), in, nil)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(report.Results) != 2 {
		t.Fatalf("want 2 results, got %d", len(report.Results))
	}
	// sorted by name: broken first
	if res := report.Results[0]; res.Checker != "broken" || res.Status != verify.StatusError {
		t.Fatalf("want broken error result, got %+v", res)
	}
	if res := report.Results[1]; res.Checker != "ok" || res.Status != verify.StatusPass {
		t.Fatalf("want ok pass result, got %+v", res)
	}
	if !report.Failed() {
		t.Fatal("want report to fail")
	}

	report, err = r.Run(context.Background(), in, []string{"ok"})
	if err != nil || report.Failed() {
		t.Fatalf("want only ok to run and pass, got %+v, %v", report, err)
	}

	if _, err = r.Run(context.Background(), in, []string{"missing"}); err == nil {
		t.Fatal("want error on unknown checker")
	}
}

func TestQuantityChecker(t *testing.T) {
	results, err := verify.QuantityChecker{}.Check(context.Background(), &verify.Input{Root: sampleTree()})
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	got := make(map[string]verify.Status)
	for _, res := range results {
		got[res.Subject] = res.Status
	}
	want := map[string]verify.Status{
		"robot":             verify.StatusPass,
		"robot/arm":         verify.StatusPass,
		"robot/arm/bracket": verify.StatusFail,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

// TestHelperProcess is not a real test. ExecChecker tests run the test
// binary itself as the external checker.
func TestHelperProcess(t *testing.T) {
	mode := os.Getenv("CYANOTYPE_VERIFY_HELPER")
	if mode == "" {
		return
	}
	defer os.Exit(0)

	if mode == "crash" {
		fmt.Fprint(os.Stderr, "solver crashed")
		os.Exit(3)
	}

	var req verify.ExecRequest
	if err := json.NewDecoder(os.Stdin).Decode(&req); err != nil {
		os.Exit(2)
	}
	resp := verify.ExecResponse{Results: make([]verify.Result, 0)}
	var visit func(n *verify.ExecNode)
	visit = func(n *verify.ExecNode) {
		resp.Results = append(resp.Results, verify.Result{Subject: n.Path, Status: verify.StatusPass,
			Message: fmt.Sprintf("v%d", req.Version)})
		for _, c := range n.Children {
			visit(c)
		}
	}
	visit(req.Root)
	_ = json.NewEncoder(os.Stdout).Encode(resp)
}

func helperChecker(t *testing.T, mode string) *verify.ExecChecker {
	t.Setenv("CYANOTYPE_VERIFY_HELPER", mode)
	return &verify.ExecChecker{
		CheckerName: "helper",
		Command:     []string{os.Args[0], "-test.run=^TestHelperProcess$"},
	}
}

func TestExecChecker(t *testing.T) {
	c := helperChecker(t, "echo")
	results, err := c.Check(context.Background(), &verify.Input{Root: sampleTree()})
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("want 3 results, got %d", len(results))
	}
	if res := results[2]; res.Subject != "robot/arm/bracket" || res.Message != "v1" {
		t.Fatalf("unexpected result %+v", res)
	}
}

func TestExecChecker_Crash(t *testing.T) {
	c := helperChecker(t, "crash")
	_, err := c.Check(context.Background(), &verify.Input{Root: sampleTree()})
	if err == nil {
		t.Fatal("want error when the checker exits non-zero")
	}
}
