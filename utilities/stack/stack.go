package stack

// Stack is an implementation of stack container.
type Stack[T any] struct{ s []T }

// New creates a new instance of Stack.
func New[T any](capacity int) *Stack[T] {
	return &Stack[T]{s: make([]T, 0, capacity)}
}

// Reset resets the stack.
func (s *Stack[T]) Reset() { s.s = s.s[:0] }

// Push adds an element to the stack.
func (s *Stack[T]) Push(f T) { s.s = append(s.s, f) }

// Pop returns and deletes the last stack element.
func (s *Stack[T]) Pop() (top T) {
	if l := len(s.s) - 1; l >= 0 {
		top = s.s[l]
		s.s = s.s[:l]
	}
	return
}

// PopPush executes Pop and Push operations in sequence.
func (s *Stack[T]) PopPush(f T) (popped T) {
	if l := len(s.s) - 1; l >= 0 {
		popped = s.s[l]
		s.s = s.s[:l]
	}
	s.s = append(s.s, f)
	return
}

// Top returns the last stack element.
func (s *Stack[T]) Top() (top T) {
	if l := len(s.s) - 1; l >= 0 {
		return s.s[l]
	}
	return
}

// TopOffsetFn calls fn with the last stack element at offset.
func (s *Stack[T]) TopOffsetFn(offset int, fn func(*T)) {
	if l := len(s.s) - 1; l >= 0 {
		fn(&s.s[l-offset])
	}
}

// Get returns the element at index from bottom.
func (s *Stack[T]) Get(index int) T {
	return s.s[index]
}

// Len returns the stack length.
func (s *Stack[T]) Len() int {
	return len(s.s)
}
