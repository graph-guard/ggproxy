package set

// Set is a generic set implementation.
type Set[T comparable] struct {
	e       int       // Number of enabled items
	index   map[T]int // T -> index in ordered
	ordered []T       // index -> T
}

// New creaates a new instance of Set.
func New[T comparable](u ...T) *Set[T] {
	r := make(map[T]int, len(u))
	for i := range u {
		r[u[i]] = i
	}
	return &Set[T]{
		e:       0,
		ordered: u,
		index:   r,
	}
}

func (r *Set[T]) Add(t T) bool {
	if _, ok := r.index[t]; ok {
		return false
	}
	r.index[t] = len(r.ordered)
	r.ordered = append(r.ordered, t)
	return true
}

func (r *Set[T]) Disable(t T) bool {
	index, ok := r.index[t]
	if !ok || index >= r.e {
		// Not in the set or already disabled
		return false
	}
	r.e--
	swapItemVal := r.ordered[r.e]
	swapItemIndex := r.index[swapItemVal]
	// Swap items
	r.ordered[swapItemIndex], r.ordered[index] =
		r.ordered[index], r.ordered[swapItemIndex]
	// Swap indexes
	r.index[t], r.index[swapItemVal] = swapItemIndex, index
	return true
}

func (r *Set[T]) Enable(t T) bool {
	index, ok := r.index[t]
	if !ok || index <= r.e-1 { // 0 <= -1
		// Not in the set or already enabled
		return false
	}
	swapItemVal := r.ordered[r.e]
	swapItemIndex := r.index[swapItemVal]
	// Swap items
	r.ordered[swapItemIndex], r.ordered[index] =
		r.ordered[index], r.ordered[swapItemIndex]
	// Swap indexes
	r.index[t], r.index[swapItemVal] = swapItemIndex, index
	r.e++
	return true
}

func (r *Set[T]) VisitEnabled(fn func(T) (stop bool)) {
	for i := range r.ordered[:r.e] {
		if fn(r.ordered[i]) {
			break
		}
	}
}

func (r *Set[T]) Enabled() (count int) { return r.e }

func (r *Set[T]) Reset() {
	r.e = 0
}
