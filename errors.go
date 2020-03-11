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

	gotColor  = redColor
	wantColor = cyanColor

	diffGotColor     = "\033[46m\033[30m"
	diffGotStopColor = "\033[0m"

	diffWantColor     = "\033[41m\033[30m"
	diffWantStopColor = "\033[0m"
)

type errorList struct {
	List []error
}

func (el *errorList) add(err error) {
	el.List = append(el.List, err)
}

func (el *errorList) err() error {
	if len(el.List) > 0 {
		return el
	}
	return nil
}

func (el *errorList) Error() (res string) {
	for _, err := range el.List {
		res += fmt.Sprintf("%s\n", err)
	}
	return strings.TrimRight(res, "\n")
}

type validityError struct {
	got  reflect.Value
	want reflect.Value
	path path
}

func (err *validityError) Error() string {
	got, want := "VALID", "VALID"
	if !err.got.IsValid() {
		got = "INVALID"
	}
	if !err.want.IsValid() {
		want = "INVALID"
	}
	got = gotColor + got + stopColor
	want = wantColor + want + stopColor
	return fmt.Sprintf("%s: Validity mismatch; got=%s, want=%s", err.path, got, want)
}

type typeError struct {
	got  reflect.Value
	want reflect.Value
	path path
}

func (err *typeError) Error() string {
	got := gotColor + err.got.Type().String() + stopColor
	want := wantColor + err.want.Type().String() + stopColor
	return fmt.Sprintf("%s: Type mismatch; got=%s, want=%s", err.path, got, want)
}

type nilError struct {
	got  reflect.Value
	want reflect.Value
	path path
}

func (err *nilError) Error() string {
	got, want := "<nil>", "<nil>"
	if !err.got.IsNil() {
		got = fmt.Sprintf("%#v", err.got)
	}
	if !err.want.IsNil() {
		want = fmt.Sprintf("%#v", err.want)
	}
	got = gotColor + got + stopColor
	want = wantColor + want + stopColor
	return fmt.Sprintf("%s: Nil mismatch; got=%s, want=%s", err.path, got, want)
}

type lenError struct {
	got  reflect.Value
	want reflect.Value
	path path
}

func (err *lenError) Error() string {
	got := gotColor + fmt.Sprintf("%d", err.got.Len()) + stopColor
	want := wantColor + fmt.Sprintf("%d", err.want.Len()) + stopColor
	kind := err.want.Kind()
	return fmt.Sprintf("%s: Length of %s mismatch; got=%s, want=%s", err.path, kind, got, want)
}

type funcError struct {
	got  reflect.Value
	want reflect.Value
	path path
}

func (err *funcError) Error() string {
	got, want := "<nil>", "<nil>"
	if !err.got.IsNil() {
		got = err.got.Type().String()
	}
	if !err.want.IsNil() {
		want = err.want.Type().String()
	}
	got = gotColor + got + stopColor
	want = wantColor + want + stopColor
	return fmt.Sprintf("%s: Func mismatch; got=%s, want=%s (Can only match if both are <nil>)", err.path, got, want)
}

type valueError struct {
	got  interface{}
	want interface{}
	path path
}

func (err *valueError) Error() string {
	got := gotColor + fmt.Sprintf("%v", err.got) + stopColor
	want := wantColor + fmt.Sprintf("%v", err.want) + stopColor
	return fmt.Sprintf("%s: Value mismatch; got=%s, want=%s", err.path, got, want)
}

type zeroError struct {
	got  interface{}
	want interface{}
	path path
}

func (err *zeroError) Error() string {
	var got, want string
	if err.got == true {
		got = gotColor + "<zero>" + stopColor
		want = wantColor + "<non-zero>" + stopColor
	} else {
		got = gotColor + "<non-zero>" + stopColor
		want = wantColor + "<zero>" + stopColor
	}
	return fmt.Sprintf("%s: Zero mismatch (both values must be either zero or non-zero); got=%s, want=%s", err.path, got, want)
}

type stringError struct {
	got  string
	want string
	path path
}

const maxlen = 30 // max string length displayable in an error message

func newStringError(got, want string, p path) *stringError {
	err := &stringError{
		got:  gotColor + `"` + got + `"` + stopColor,
		want: wantColor + `"` + want + `"` + stopColor,
		path: p,
	}
	if d := sdiff(got, want); d != nil {
		start, end := got[:d.start], got[d.end:]
		delta := got[d.start:d.end]

		err.got = gotColor + `"` +
			start + stopColor + diffGotColor +
			delta + diffGotStopColor + gotColor +
			end + `"` + stopColor

		if len(want) > d.start {
			start = want[:d.start]
			if len(want) > d.end {
				end = want[d.end:]
				delta = want[d.start:d.end]
			} else {
				end = ""
				delta = want[d.start:]
			}
			err.want = wantColor + `"` +
				start + stopColor + diffWantColor +
				delta + diffWantStopColor + wantColor +
				end + `"` + stopColor
		}
	}
	return err
}

func (err *stringError) Error() string {
	return fmt.Sprintf("%s: Value mismatch; got=%s, want=%s", err.path, err.got, err.want)
}

////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////

type path []pathnode

func (p path) add(n pathnode) path {
	q := append(path{}, p...)
	return append(q, n)
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
		return fmt.Sprintf("- <%s%s%s>", purpleColor, "nil", stopColor)
	}
	return fmt.Sprintf("- (%s)", n.typ)
}

type arrnode struct {
	index int
}

func (n arrnode) str(color interface{}) string {
	return fmt.Sprintf("[%d]", n.index)
}

type channode struct {
	index int
}

func (n channode) str(color interface{}) string {
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
