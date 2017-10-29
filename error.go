package compare

import (
	"fmt"
	"reflect"
	"strings"
)

const (
	// ANSI color values used to colorize terminal output.
	redColor    = "\033[91m"
	yellowColor = "\033[93m"
	purpleColor = "\033[95m"
	cyanColor   = "\033[96m"
	greenColor  = "\033[92m"
	stopColor   = "\033[0m"
)

type ErrorList struct {
	List []error
}

func (li *ErrorList) Add(err error) {
	li.List = append(li.List, err)
}

func (li *ErrorList) Error() (res string) {
	for _, err := range li.List {
		res += fmt.Sprintf("%s\n", err)
	}
	return strings.TrimRight(res, "\n")
}

func (li *ErrorList) Err() error {
	if len(li.List) > 0 {
		return li
	}
	return nil
}

type ValidityError struct {
	got, want reflect.Value
	path      path
}

func NewValidityError(got, want reflect.Value, p path) *ValidityError {
	return &ValidityError{got, want, p}
}

func (err *ValidityError) Error() string {
	got := yellowColor + fmtvalidity(err.got) + stopColor
	want := cyanColor + fmtvalidity(err.want) + stopColor
	return fmt.Sprintf("%s: Validity mismatch; got=%s, want=%s", err.path, got, want)
}

func fmtvalidity(v reflect.Value) string {
	if v.IsValid() {
		return "VALID"
	}
	return "INVALID"
}

type TypeError struct {
	got, want reflect.Value
	path      path
}

func NewTypeError(got, want reflect.Value, p path) *TypeError {
	return &TypeError{got, want, p}
}

func (err *TypeError) Error() string {
	got := yellowColor + fmttype(err.got) + stopColor
	want := cyanColor + fmttype(err.want) + stopColor
	return fmt.Sprintf("%s: Type mismatch; got=%s, want=%s", err.path, got, want)
}

func fmttype(v reflect.Value) string {
	return v.Type().String()
}

type NilError struct {
	got, want reflect.Value
	path      path
}

func NewNilError(got, want reflect.Value, p path) *NilError {
	return &NilError{got, want, p}
}

func (err *NilError) Error() string {
	got := yellowColor + fmtnil(err.got) + stopColor
	want := cyanColor + fmtnil(err.want) + stopColor
	return fmt.Sprintf("%s: Nil mismatch; got=%s, want=%s", err.path, got, want)
}

func fmtnil(v reflect.Value) string {
	if v.IsNil() {
		return "<nil>"
	}
	return fmt.Sprintf("%#v", v)
}

type LenError struct {
	got, want reflect.Value
	path      path
}

func NewLenError(got, want reflect.Value, p path) *LenError {
	return &LenError{got, want, p}
}

func (err *LenError) Error() string {
	got := yellowColor + fmtlen(err.got) + stopColor
	want := cyanColor + fmtlen(err.want) + stopColor
	kind := err.want.Kind()
	return fmt.Sprintf("%s: Length of %s mismatch; got=%s, want=%s", err.path, kind, got, want)
}

func fmtlen(v reflect.Value) string {
	return fmt.Sprintf("%d", v.Len())
}

type FuncError struct {
	got, want reflect.Value
	path      path
}

func NewFuncError(got, want reflect.Value, p path) *FuncError {
	return &FuncError{got, want, p}
}

func (err *FuncError) Error() string {
	got := yellowColor + fmtfunc(err.got) + stopColor
	want := cyanColor + fmtfunc(err.want) + stopColor
	return fmt.Sprintf("%s: Func mismatch; got=%s, want=%s (Can only match if both are <nil>)", err.path, got, want)
}

func fmtfunc(v reflect.Value) string {
	if v.IsNil() {
		return "<nil>"
	}
	return v.Type().String()
}

type ValueError struct {
	got, want reflect.Value
	path      path
}

func NewValueError(got, want reflect.Value, p path) *ValueError {
	return &ValueError{got, want, p}
}

func (err *ValueError) Error() string {
	got := yellowColor + fmtvalue(err.got) + stopColor
	want := cyanColor + fmtvalue(err.want) + stopColor
	return fmt.Sprintf("%s: Value mismatch; got=%s, want=%s", err.path, got, want)
}

func fmtvalue(v reflect.Value) string {
	return fmt.Sprintf("%v", valueInterface(v))
}

type path []pathnode

func (p path) add(n pathnode) path {
	p = append(p, n)
	return p
}

func (p path) String() (s string) {
	for _, n := range p {
		s += n.str(nil)
	}
	return s
}

type pathnode interface {
	str(color interface{}) string
}

type rootnode struct {
	typ reflect.Type
}

var niltyp = reflect.TypeOf(nil)

func (n rootnode) str(color interface{}) string {
	if n.typ == niltyp {
		return fmt.Sprintf("<%s%s%s>", purpleColor, "nil", stopColor)
	}
	return fmt.Sprintf("(%s)", n.typ)
}

type arrnode struct {
	index int
}

func (n arrnode) str(color interface{}) string {
	return fmt.Sprintf("[%d]", n.index)
}

type mapnode struct {
	key reflect.Value
}

func (n mapnode) str(color interface{}) string {
	return fmt.Sprintf("[%v]", n.key)
}

type structnode struct {
	field string
}

func (n structnode) str(color interface{}) string {
	return fmt.Sprintf(".%s", n.field)
}
