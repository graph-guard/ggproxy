package testeq

import (
	"sort"

	"golang.org/x/exp/constraints"
)

func Maps[K constraints.Ordered, V any](
	writer interface {
		Helper()
		Errorf(fmt string, v ...any)
	},
	title string,
	expected, actual map[K]V,
	check func(expected, actual V) (errMsg string),
	stringify func(V) string,
) (ok bool) {
	writer.Helper()
	ok = true

	// Order expected
	expKeysOrdered := make([]K, 0, len(expected))
	for k := range expected {
		expKeysOrdered = append(expKeysOrdered, k)
	}
	sort.Slice(expKeysOrdered, func(i, j int) bool {
		return expKeysOrdered[i] < expKeysOrdered[j]
	})

	// Order actual
	actKeysOrdered := make([]K, 0, len(actual))
	for k := range actual {
		actKeysOrdered = append(actKeysOrdered, k)
	}
	sort.Slice(actKeysOrdered, func(i, j int) bool {
		return actKeysOrdered[i] < actKeysOrdered[j]
	})

	remAct := make(map[K]V, len(actual))
	for k, v := range actual {
		remAct[k] = v
	}

	// Compare
	for _, k := range expKeysOrdered {
		delete(remAct, k)
		ev := expected[k]
		if av, found := actual[k]; found {
			if msg := check(ev, av); msg != "" {
				writer.Errorf(
					"mismatching %s %v: %s",
					title, k, msg,
				)
				ok = false
			}
		} else {
			writer.Errorf(
				"missing %s %v (%s)",
				title, k, stringify(ev),
			)
			ok = false
		}
	}

	// Check unexpected keys
	for _, k := range actKeysOrdered {
		if v, found := remAct[k]; found {
			writer.Errorf(
				"unexpected %s %v (%s)",
				title, k, stringify(v),
			)
			ok = false
		}
	}

	return ok
}

func Slices[T any](
	writer interface {
		Helper()
		Errorf(fmt string, v ...any)
	},
	title string,
	expect, actual []T,
	check func(expected, actual T) (errMsg string),
	stringify func(T) string,
) (ok bool) {
	writer.Helper()
	ok = true

	for i, a := range actual {
		if i >= len(expect) {
			break
		}
		e := expect[i]
		if errMsg := check(e, a); errMsg != "" {
			writer.Errorf(
				"mismatching %s at index %d: %s",
				title, i, errMsg,
			)
			ok = false
		}
	}
	if d := len(actual) - len(expect); d > 0 {
		for i, a := range actual[len(actual)-d:] {
			writer.Errorf(
				"unexpected %s at index %d (%s)",
				title, len(actual)-d+i, stringify(a),
			)
			ok = false
		}
	} else if d < 0 {
		for i, a := range expect[len(expect)-(-d):] {
			writer.Errorf(
				"missing %s at index %d (%s)",
				title, len(expect)-(-d)+i, stringify(a),
			)
			ok = false
		}
	}
	return ok
}
