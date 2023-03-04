package union

import (
	"github.com/graph-guard/ggproxy/pkg/atoi"
	"github.com/graph-guard/ggproxy/pkg/gqlparse"
	"github.com/graph-guard/ggproxy/pkg/unsafe"
	"github.com/graph-guard/gqlscan"
)

type Type int8

const (
	_ Type = iota
	TypeNull
	TypeTokens
	TypeString
	TypeEnum
	TypeFloat
	TypeInt
	TypeBoolean
	TypeArray
	TypeArrayEnd
)

func Array() Union                    { return Union{unionType: TypeArray} }
func ArrayEnd() Union                 { return Union{unionType: TypeArrayEnd} }
func Null() Union                     { return Union{unionType: TypeNull} }
func Int(v int32) Union               { return Union{unionType: TypeInt, i: v} }
func Float(v float64) Union           { return Union{unionType: TypeFloat, f: v} }
func String(v string) Union           { return Union{unionType: TypeString, s: v} }
func Enum(v string) Union             { return Union{unionType: TypeEnum, s: v} }
func Tokens(v []gqlparse.Token) Union { return Union{unionType: TypeTokens, t: v} }
func True() Union                     { return Union{unionType: TypeBoolean, i: 1} }
func False() Union                    { return Union{unionType: TypeBoolean, i: 0} }

func (t Type) String() string {
	switch t {
	case TypeNull:
		return "Null"
	case TypeTokens:
		return "Tokens"
	case TypeString:
		return "String"
	case TypeEnum:
		return "Enum"
	case TypeFloat:
		return "Float"
	case TypeInt:
		return "Int"
	case TypeBoolean:
		return "Boolean"
	case TypeArray:
		return "array"
	case TypeArrayEnd:
		return "array_end"
	}
	return ""
}

// Union represents any of Float, Int, String, Enum, Boolean
type Union struct {
	t         []gqlparse.Token
	s         string
	f         float64
	i         int32
	unionType Type
}

// Value defines whether a value could be extracted (>ValueNone)
// and how it was extracted.
type Value int8

const (
	// ValueNone that there is no value.
	ValueNone Value = iota - 1
	// ValueConv indicates that the value matched exactly.
	ValueExact
	// ValueConv indicates that the value was converted.
	ValueConv
	// ValueInf indicates that the value was inferred.
	ValueInf
	// ValueInfConv indicates that the value was inferred and converted.
	ValueInfConv
)

// Type returns the type of the value the union is holding,
// or 0 if the union has zero value.
func (u *Union) Type() Type { return u.unionType }

// Float returns a float64 value or ValueNone if
// the union is storing a value of a different type.
func (u *Union) Float() (float64, Value) {
	switch u.unionType {
	case TypeFloat:
		return u.f, ValueExact
	case TypeInt:
		return float64(u.i), ValueConv
	case TypeTokens:
		switch u.t[0].ID {
		case gqlscan.TokenInt:
			return float64(atoi.MustI32(u.t[0].Value)), ValueInfConv
		case gqlscan.TokenFloat:
			return atoi.MustF64(u.t[0].Value), ValueInf
		}
	}
	return 0, ValueNone
}

// Int returns an int32 value or ValueNone if
// the union is storing a value of a different type.
func (u *Union) Int() (value int32, ok Value) {
	switch u.unionType {
	case TypeInt:
		return u.i, ValueExact
	case TypeTokens:
		if u.t[0].ID == gqlscan.TokenInt {
			return atoi.MustI32(u.t[0].Value), ValueInf
		}
	}
	return 0, ValueNone
}

// Enum returns an enum value or ValueNone if
// the union is storing a value of a different type.
func (u *Union) Enum() (value string, ok Value) {
	switch u.unionType {
	case TypeEnum:
		return u.s, ValueExact
	case TypeTokens:
		if u.t[0].ID == gqlscan.TokenEnumVal {
			return unsafe.B2S(u.t[0].Value), ValueInf
		}
	}
	return "", ValueNone
}

// Bool returns a boolean value or ValueNone if
// the union is storing a value of a different type.
func (u *Union) Bool() (value bool, ok Value) {
	if u.unionType == TypeBoolean {
		return u.i > 0, ValueExact
	} else if u.unionType == TypeTokens {
		switch u.t[0].ID {
		case gqlscan.TokenTrue:
			return true, ValueInf
		case gqlscan.TokenFalse:
			return false, ValueInf
		}
	}
	return false, ValueNone
}

// String returns a string value or ValueNone if
// the union is storing a value of a different type.
func (u *Union) String() (value string, ok Value) {
	if u.unionType == TypeString {
		return u.s, ValueExact
	} else if u.unionType == TypeTokens {
		switch u.t[0].ID {
		case gqlscan.TokenStr:
			return unsafe.B2S(u.t[0].Value), ValueInf
		case gqlscan.TokenStrBlock:
			panic("todo")
			// return unsafe.B2S(u.t[0].Value), true
		}
	}
	return "", ValueNone
}

// Tokens returns the tokens slice or nil if
// the union is storing a value of a different type.
func (u *Union) Tokens() (value []gqlparse.Token) {
	if u.unionType == TypeTokens {
		return u.t
	}
	return nil
}

// IsNull returns true if the union represents a null value.
func (u *Union) IsNull() bool {
	return u.unionType == TypeNull ||
		u.unionType == TypeTokens && u.t[0].ID == gqlscan.TokenNull
}

func Equal(l, r Union) bool {
	if l.unionType == TypeArray && r.unionType == TypeArray ||
		l.unionType == TypeArrayEnd && r.unionType == TypeArrayEnd {
		return true
	}
	if l.IsNull() && r.IsNull() {
		return true
	}
	if l, v := l.Bool(); v != ValueNone {
		r, v := r.Bool()
		return v != ValueNone && l == r
	}
	if l, v := l.Int(); v != ValueNone {
		r, v := r.Int()
		return v != ValueNone && l == r
	}
	if l, v := l.Float(); v != ValueNone {
		r, v := r.Float()
		return v != ValueNone && l == r
	}
	if l, v := l.String(); v != ValueNone {
		r, v := r.String()
		return v != ValueNone && l == r
	}
	if l, v := l.Enum(); v != ValueNone {
		r, v := r.Enum()
		return v != ValueNone && l == r
	}

	if r.unionType != TypeTokens || len(l.t) != len(r.t) {
		return false
	}
	for i := range r.t {
		if l.t[i].ID != r.t[i].ID ||
			string(l.t[i].Value) != string(r.t[i].Value) {
			return false
		}
	}
	return l.unionType != TypeArray && l.unionType != TypeArrayEnd

	/* OLD IMPLEMENTATION */
	// switch l.unionType {
	// case TypeInt:
	// 	switch r.unionType {
	// 	case TypeInt:
	// 		return l.i == r.i
	// 	case TypeFloat:
	// 		return l.Float() == r.f
	// 	case TypeTokens:
	// 		switch r.t[0].Type {
	// 		case gqlscan.TokenInt:
	// 			return l.i == atoi.MustI32(r.t[0].Value)
	// 		case gqlscan.TokenFloat:
	// 			return l.Float() == atoi.MustF64(r.t[0].Value)
	// 		}
	// 	}
	// case TypeFloat:
	// 	switch r.unionType {
	// 	case TypeInt:
	// 		return l.f == r.Float()
	// 	case TypeFloat:
	// 		return l.f == r.f
	// 	case TypeTokens:
	// 		switch r.t[0].Type {
	// 		case gqlscan.TokenInt:
	// 			return l.f == float64(atoi.MustI32(r.t[0].Value))
	// 		case gqlscan.TokenFloat:
	// 			return l.f == atoi.MustF64(r.t[0].Value)
	// 		}
	// 	}
	// case TypeEnum:
	// 	return l.Enum() == r.Enum()
	// case TypeBoolean:
	// 	return l.Bool() == r.Bool()
	// case TypeNull:
	// 	return r.unionType == TypeNull
	// case TypeString:
	// 	switch r.unionType {
	// 	case TypeString:
	// 		return l.s == r.s
	// 	case TypeTokens:
	// 		switch r.t[0].Type {
	// 		case gqlscan.TokenStr:
	// 			return l.s == string(r.t[0].Value)
	// 		case gqlscan.TokenStrBlock:
	// 			panic("todo")
	// 		}
	// 	}
	// case TypeTokens:
	// 	if r.unionType != TypeTokens {
	// 		return false
	// 	}
	// 	if len(l.t) != len(r.t) {
	// 		return false
	// 	}
	// 	for i := range r.t {
	// 		if l.t[i].Type != r.t[i].Type {
	// 			return false
	// 		}
	// 		if string(l.t[i].Value) != string(r.t[i].Value) {
	// 			return false
	// 		}
	// 	}
	// 	return true
	// }
	// return false
}
