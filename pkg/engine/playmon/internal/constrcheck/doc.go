// package constrcheck provides function Make which
// creates constraint checker functions for each input in a GQT operation.
// Example:
//
//	query {
//	  f(
//	    a: *
//	    b=$v: > 10
//	    c: [...["a", != "b"]]
//	    d: len < (4+2) * $v
//	  )
//	}
//
// The above template will produce 3 constraint checker functions because
// there is no need to check argument "a" since it accepts any value.
// Argument "b" will accept Int values greater 10;
// argument "c" will accept arrays where every item is an array
// that contains exactly 2 items, first of which must be equal String("a")
// and second must not be equal String("b");
// argument "d" will accept String values (or arrays depending on schema type)
// the length of which is less than (4+2) multiplied by
// the value of argument "b".
// Constraint checker functions will be returned as a map of path -> function.
// and require argument inputs to be passed as map[string]any, where
// int32, float64, string, Enum, bool values as well as (nested) slices of any
// of the mentioned types are matched.
package constrcheck
