package utils

import (
	"fmt"
)

// Sort a list of strings in dependency order.
// If dependents["foo"] = {}string["bar", "baz"], then bar and baz both depend on foo.
func TopologicalSort(items []string, dependents map[string][]string) ([]string, error) {
	var (
		sorted    []string
		unmarked  = make(map[string]bool)
		tmpMarked = make(map[string]bool)
	)

	for _, n := range items {
		unmarked[n] = true
	}

	var visit func(item string) error
	visit = func(item string) error {
		if tmpMarked[item] {
			return fmt.Errorf("Circular dependency between %s and %#v\n", item, tmpMarked)
		}
		if unmarked[item] {
			tmpMarked[item] = true
			for _, d := range dependents[item] {
				if err := visit(d); err != nil {
					return err
				}
			}
			delete(tmpMarked, item)
			delete(unmarked, item)
			sorted = append([]string{item}, sorted...)
		}
		return nil
	}

	for len(unmarked) > 0 {
		if err := visit(pluck(unmarked)); err != nil {
			return []string{}, err
		}
	}

	return sorted, nil
}

// Get an arbitrary item from a set
func pluck(set map[string]bool) string {
	for item, _ := range set {
		return item
	}

	return ""
}
