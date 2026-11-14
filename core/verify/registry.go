package verify

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
)

type Registry struct {
	mu       sync.RWMutex
	checkers map[string]Checker
}

func NewRegistry() *Registry {
	return &Registry{
		checkers: make(map[string]Checker),
	}
}

// Default is the registry the verify command uses.
var Default = NewRegistry()

// Register adds c to the default registry.
func Register(c Checker) error {
	return Default.Register(c)
}

func (r *Registry) Register(c Checker) error {
	if c == nil || c.Name() == "" {
		return errors.New("checker needs a name")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.checkers[c.Name()]
	if ok {
		return fmt.Errorf("checker %q already registered", c.Name())
	}
	r.checkers[c.Name()] = c
	return nil
}

func (r *Registry) Get(name string) (Checker, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.checkers[name]
	return c, ok
}

// Names returns the registered checker names, sorted.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.checkers))
	for name := range r.checkers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Run runs the named checkers (all of them when names is empty) in order.
// A checker that returns an error does not stop the others; it shows up
// as a StatusError result.
func (r *Registry) Run(ctx context.Context, in *Input, names []string) (*Report, error) {
	if len(names) == 0 {
		names = r.Names()
	}
	checkers := make([]Checker, 0, len(names))
	for _, name := range names {
		c, ok := r.Get(name)
		if !ok {
			return nil, fmt.Errorf("unknown checker %q", name)
		}
		checkers = append(checkers, c)
	}

	report := &Report{Results: make([]Result, 0)}
	for _, c := range checkers {
		results, err := c.Check(ctx, in)
		if err != nil {
			report.Results = append(report.Results, Result{
				Checker: c.Name(),
				Subject: NodePath(in.Root),
				Status:  StatusError,
				Message: err.Error(),
			})
			continue
		}
		for _, res := range results {
			res.Checker = c.Name()
			report.Results = append(report.Results, res)
		}
	}
	return report, nil
}
