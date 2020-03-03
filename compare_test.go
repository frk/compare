package compare

import (
	"fmt"
	"math"
	"reflect"
	"testing"
	"time"
)

type Basic struct {
	x int
	y float32
}

type NotBasic Basic

type Tagged struct {
	f1 string `cmp:"-"`
	f2 string `cmp:"+"`
}

type CompareTest struct {
	a, b interface{}
	err  error
}

// Simple functions for Compare tests.
var (
	fn1 func()             // nil.
	fn2 func()             // nil.
	fn3 = func() { fn1() } // Not nil.
)

type self struct{}

type Loop *Loop
type Loopy interface{}

var loop1, loop2 Loop
var loopy1, loopy2 Loopy

func init() {
	loop1 = &loop2
	loop2 = &loop1

	loopy1 = &loopy2
	loopy2 = &loopy1
}

func elist(errs ...error) *errorList {
	list := new(errorList)
	list.List = errs
	return list
}

var rvof = reflect.ValueOf
var rtof = reflect.TypeOf

var compareTests = []CompareTest{
	// Equalities
	{a: nil, b: nil, err: nil},
	{a: 1, b: 1, err: nil},
	{a: int32(1), b: int32(1), err: nil},
	{a: 0.5, b: 0.5, err: nil},
	{a: float32(0.5), b: float32(0.5), err: nil},
	{a: "hello", b: "hello", err: nil},
	{a: make([]int, 10), b: make([]int, 10), err: nil},
	{a: &[3]int{1, 2, 3}, b: &[3]int{1, 2, 3}, err: nil},
	{a: Basic{1, 0.5}, b: Basic{1, 0.5}, err: nil},
	{a: error(nil), b: error(nil), err: nil},
	{a: map[int]string{1: "one", 2: "two"}, b: map[int]string{2: "two", 1: "one"}, err: nil},
	{a: fn1, b: fn2, err: nil},
	{a: Tagged{"abc", "foo"}, b: Tagged{"abc", "foo"}, err: nil},
	{a: Tagged{"abc", "foo"}, b: Tagged{"def", "bar"}, err: nil},
	{a: Tagged{"abc", ""}, b: Tagged{"", ""}, err: nil},

	// Inequalities
	{
		a: 1, b: 2,
		err: elist(&valueError{
			got: int64(1), want: int64(2),
			path: path{rootnode{rtof(2)}},
		}),
	}, {
		a: int32(3), b: int32(4),
		err: elist(&valueError{
			got: int64(3), want: int64(4),
			path: path{rootnode{rtof(int32(4))}},
		}),
	}, {
		a: 0.5, b: 0.6,
		err: elist(&valueError{
			got: float64(0.5), want: float64(0.6),
			path: path{rootnode{rtof(0.6)}},
		}),
	}, {
		a: float32(0.7), b: float32(0.8),
		err: elist(&valueError{
			got: float32(0.7), want: float32(0.8),
			path: path{rootnode{rtof(float32(0.8))}},
		}),
	}, {
		a: "hello", b: "hey",
		err: elist(
			newStringError("hello", "hey", path{rootnode{rtof("")}}),
		),
	}, {
		a: make([]int, 10), b: make([]int, 11),
		err: elist(&lenError{
			got: rvof(make([]int, 10)), want: rvof(make([]int, 11)),
			path: path{rootnode{rtof([]int{})}},
		}),
	}, {
		a: &[3]int{1, 2, 3},
		b: &[3]int{1, 2, 4},
		err: elist(&valueError{
			got: int(3), want: int(4),
			path: path{
				rootnode{rtof(&[3]int{})},
				arrnode{index: 2},
			},
		}),
	}, {
		a: Basic{x: 1, y: 0.5},
		b: Basic{x: 1, y: 0.6},
		err: elist(&valueError{
			got: float32(0.5), want: float32(0.6),
			path: path{
				rootnode{rtof(Basic{})},
				structnode{field: "y"},
			},
		}),
	}, {
		a: Basic{x: 1, y: 0},
		b: Basic{x: 2, y: 0},
		err: elist(&valueError{
			got: int(1), want: int(2),
			path: path{
				rootnode{rtof(Basic{})},
				structnode{field: "x"},
			},
		}),
	}, {
		a: map[int]string{1: "one", 3: "two"},
		b: map[int]string{2: "two", 1: "one"},
		err: elist(&validityError{
			got: rvof(nil), want: rvof("two"),
			path: path{
				rootnode{rtof(map[int]string{})},
				mapnode{key: rvof(2)},
			},
		}),
	}, {
		a: map[int]string{1: "one", 2: "txo"},
		b: map[int]string{2: "two", 1: "one"},
		err: elist(newStringError(
			"txo", "two",
			path{
				rootnode{rtof(map[int]string{})},
				mapnode{key: rvof(2)},
			})),
	}, {
		a: map[int]string{1: "one"},
		b: map[int]string{2: "two", 1: "one"},
		err: elist(&lenError{
			got:  rvof(map[int]string{1: "one"}),
			want: rvof(map[int]string{2: "two", 1: "one"}),
			path: path{rootnode{rtof(map[int]string{})}},
		}),
	}, {
		a: map[int]string{2: "two", 1: "one"},
		b: map[int]string{1: "one"},
		err: elist(&lenError{
			got:  rvof(map[int]string{2: "two", 1: "one"}),
			want: rvof(map[int]string{1: "one"}),
			path: path{rootnode{rtof(map[int]string{})}},
		}),
	}, {
		a: nil, b: 1,
		err: elist(&validityError{
			got: rvof(nil), want: rvof(1),
			path: path{rootnode{rtof(1)}},
		}),
	}, {
		a: 1, b: nil,
		err: elist(&validityError{
			got: rvof(1), want: rvof(nil),
			path: path{rootnode{rtof(nil)}},
		}),
	}, {
		a: fn1, b: fn3,
		err: elist(&funcError{
			got: rvof(fn1), want: rvof(fn3),
			path: path{rootnode{rtof(fn3)}},
		}),
	}, {
		a: fn3, b: fn3,
		err: elist(&funcError{
			got: rvof(fn3), want: rvof(fn3),
			path: path{rootnode{rtof(fn3)}},
		}),
	}, {
		a: [][]int{{1}},
		b: [][]int{{2}},
		err: elist(&valueError{
			got: rvof(1), want: rvof(2),
			path: path{
				rootnode{rtof([][]int{})},
				arrnode{index: 0},
				arrnode{index: 0},
			},
		}),
	}, {
		a: math.NaN(), b: math.NaN(),
		err: elist(&valueError{
			got:  rvof(math.NaN()),
			want: rvof(math.NaN()),
			path: path{rootnode{rtof(math.NaN())}},
		}),
	}, {
		a: &[1]float64{math.NaN()}, b: &[1]float64{math.NaN()},
		err: elist(&valueError{
			got:  rvof(math.NaN()),
			want: rvof(math.NaN()),
			path: path{
				rootnode{rtof(&[1]float64{})},
				arrnode{index: 0},
			},
		}),
	}, {
		a: &[1]float64{math.NaN()}, b: self{},
		err: nil,
	}, {
		a: []float64{math.NaN()}, b: []float64{math.NaN()},
		err: elist(&valueError{
			got:  rvof(math.NaN()),
			want: rvof(math.NaN()),
			path: path{
				rootnode{rtof([]float64{})},
				arrnode{index: 0},
			},
		}),
	}, {
		a: []float64{math.NaN()}, b: self{},
		err: nil,
	}, {
		a: map[float64]float64{math.NaN(): 43}, b: map[float64]float64{1: 43},
		err: elist(&validityError{
			got: rvof(nil), want: rvof(43),
			path: path{
				rootnode{rtof(map[float64]float64{})},
				mapnode{key: rvof(1)},
			},
		}),
	}, {
		a: map[float64]float64{math.NaN(): 1}, b: self{},
		err: nil,
	},

	// Nil vs empty: not the same.
	{
		a: []int{}, b: []int(nil),
		err: elist(&nilError{
			got: rvof([]int{}), want: rvof([]int(nil)),
			path: path{rootnode{rtof([]int{})}},
		}),
	}, {
		a: []int{}, b: []int{}, err: nil,
	}, {
		a: []int(nil), b: []int(nil), err: nil,
	}, {
		a: map[int]int{}, b: map[int]int(nil),
		err: elist(&nilError{
			got: rvof(map[int]int{}), want: rvof(map[int]int(nil)),
			path: path{rootnode{rtof(map[int]int{})}},
		}),
	}, {
		a: map[int]int{}, b: map[int]int{}, err: nil,
	}, {
		a: map[int]int(nil), b: map[int]int(nil), err: nil,
	},

	// Mismatched types
	{
		a: 1, b: 1.0,
		err: elist(&typeError{
			got: rvof(1), want: rvof(1.0),
			path: path{rootnode{rtof(1.0)}},
		}),
	}, {
		a: int32(1), b: int64(1),
		err: elist(&typeError{
			got: rvof(int32(1)), want: rvof(int64(1)),
			path: path{rootnode{rtof(int64(1))}},
		}),
	}, {
		a: 0.5, b: "hello",
		err: elist(&typeError{
			got: rvof(0.5), want: rvof("hello"),
			path: path{rootnode{rtof("")}},
		}),
	}, {
		a: []int{1, 2, 3}, b: [3]int{1, 2, 3},
		err: elist(&typeError{
			got: rvof([]int{1, 2, 3}), want: rvof([3]int{1, 2, 3}),
			path: path{rootnode{rtof([3]int{})}},
		}),
	}, {
		a: &[3]interface{}{1, 2, 4}, b: &[3]interface{}{1, 2, "s"},
		err: elist(&typeError{
			got: rvof(4), want: rvof("s"),
			path: path{
				rootnode{rtof(&[3]interface{}{})},
				arrnode{index: 2},
			},
		}),
	}, {
		a: Basic{1, 0.5}, b: NotBasic{1, 0.5},
		err: elist(&typeError{
			got: rvof(Basic{1, 0.5}), want: rvof(NotBasic{1, 0.5}),
			path: path{rootnode{rtof(NotBasic{})}},
		}),
	}, {
		a: map[uint]string{1: "one", 2: "two"},
		b: map[int]string{2: "two", 1: "one"},
		err: elist(&typeError{
			got:  rvof(map[uint]string{1: "one", 2: "two"}),
			want: rvof(map[int]string{2: "two", 1: "one"}),
			path: path{rootnode{rtof(map[int]string{})}},
		}),
	},

	// Possible loops.
	{
		a: &loop1, b: &loop1, err: nil,
	}, {
		a: &loop1, b: &loop2, err: nil,
	}, {
		a: &loopy1, b: &loopy1, err: nil,
	}, {
		a: &loopy1, b: &loopy2, err: nil,
	},

	// Tags
	{
		a: Tagged{f2: "foo"}, b: Tagged{f2: ""},
		err: elist(&zeroError{false, true, path{
			rootnode{rtof(Tagged{})},
			structnode{field: "f2"},
		}}),
	}, {
		a: Tagged{f2: ""}, b: Tagged{f2: "bar"},
		err: elist(&zeroError{true, false, path{
			rootnode{rtof(Tagged{})},
			structnode{field: "f2"},
		}}),
	},
}

func TestCompare(t *testing.T) {
	var errstr = func(err error) string {
		if err == nil {
			return "<nil>"
		}
		return err.Error()
	}

	conf := Config{ObserveFieldTag: "cmp"}

	for _, test := range compareTests {
		if test.b == (self{}) {
			test.b = test.a
		}

		if err := conf.Compare(test.a, test.b); errstr(err) != errstr(test.err) {
			t.Errorf("Compare(%v, %v) = %v\n\n", test.a, test.b, err)
			t.Errorf("\"%s\" != \"%s\"", errstr(err), errstr(test.err))
		}
	}
}

// Below is the example code used for generating the example output.

type Author struct {
	FirstName  string
	LastName   string
	MiddleName string
}

type Publisher struct {
	Name string
	HQ   interface{}
}

type Book struct {
	ISBN       interface{}
	Title      string
	ReleasedAt time.Time
	Authors    []*Author
	Publisher  *Publisher
}

func TestExample(t *testing.T) {
	t.Skip()
	got := &Book{
		ISBN:       4101001545,
		Title:      "海辺のカフカ",
		ReleasedAt: time.Date(2005, time.March, 1, 0, 0, 0, 0, time.UTC),
		Authors: []*Author{{
			LastName:  "村上",
			FirstName: "春樹",
		}},
		Publisher: &Publisher{
			Name: "新潮社",
			HQ:   "JP",
		},
	}
	want := &Book{
		ISBN:       "0099458322",
		Title:      "Kafka on the Shore",
		ReleasedAt: time.Date(2005, time.October, 6, 0, 0, 0, 0, time.UTC),
		Authors: []*Author{{
			FirstName: "Haruki",
			LastName:  "Murakami",
		}},
		Publisher: &Publisher{
			Name: "Vintage",
			HQ:   nil,
		},
	}

	if err := Compare(got, want); err != nil {
		fmt.Println(err)
	}
}
