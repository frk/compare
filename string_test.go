package compare

import (
	"reflect"
	"testing"
)

func Test_sdiff(t *testing.T) {
	tests := []struct {
		a, b string
		want *diff
		str  string
	}{{
		a: "", b: "",
	}, {
		a: "abc", b: "abc",
	}, {
		a: "日本語", b: "日本語",
	}, {
		a: "a", b: "b",
		want: &diff{0, 1},
	}, {
		a: "abc", b: "adc",
		want: &diff{1, 2},
	}, {
		a: "hello world", b: "hell0\tWorld",
		want: &diff{4, 7},
	}, {
		a: "hello world!!", b: "hello world",
		want: &diff{11, 13},
	}, {
		a: "hello world", b: "hello world!!",
		want: &diff{11, 11},
	}, {
		a: "hello worlb", b: "hello world!!",
		want: &diff{10, 11},
	}, {
		a: "日木語", b: "日本語",
		want: &diff{3, 6},
	}}

	for i, tt := range tests {
		got := sdiff(tt.a, tt.b)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("#%d: %q<>%q diff got=%v, want=%v", i, tt.a, tt.b, got, tt.want)
		}
	}
}

func Test_strim(t *testing.T) {
	tests := []struct {
		pos, max int
		s, want  string
	}{{
		pos: 0, max: 0,
		s: "",
	}, {
		pos: 0, max: 5,
		s: "lorem ipsum", want: "lorem",
	}, {
		pos: 10, max: 5,
		s: "", want: "",
	}, {
		pos: 0, max: 5,
		s: "lorem ipsum", want: "lorem",
	}, {
		pos: 6, max: 5,
		s: "lorem ipsum", want: "m ip",
	}, {
		pos: 8, max: 5,
		s: "lorem ipsum", want: "ipsu",
	}, {
		pos: 10, max: 5,
		s: "lorem ipsum", want: "psum",
	}, {
		pos: 15, max: 5,
		s: "lorem ipsum", want: "psum",
	}}

	for i, tt := range tests {
		s := strim(tt.s, tt.pos, tt.max)
		if s != tt.want {
			t.Errorf("#%d: diff got=%q, want=%q", i, s, tt.want)
		}
	}
}
