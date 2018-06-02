package compare

import (
	"unicode/utf8"
)

type diff struct {
	start, end int
}

func _diff(got, want string) *diff {
	if got == want {
		return nil
	}

	ln := len(got)
	if ln > len(want) {
		ln = len(want)
	}

	start, end := -1, -1
	for i := 0; i < ln; i++ {
		if start == -1 && got[i] != want[i] {
			start = i
			end = i
		}

		if start > -1 && got[i] == want[i] {
			end = i - 1
			break
		}
	}
	if start == -1 {
		return &diff{len(got), len(want) - 1}
	}

	if start > -1 {
		// adjust for runes
		r, w := utf8.DecodeRuneInString(got[start:])
		if w > 1 || r == utf8.RuneError {
			for r == utf8.RuneError && start > 0 {
				start -= 1 // back up got byte
				r, w = utf8.DecodeRuneInString(got[start:])
			}
			if start+w > end {
				end = start + w
			}
		}
		return &diff{start, end}
	}
	return nil
}

func _trim(s string, pos, max int) string {
	slen := len(s)
	if slen <= max {
		return s
	}

	half := max / 2
	if slen > max {
		lh, rh := 0, max
		if pos > half {
			lh = pos - half
			rh = pos + half
		}
		if rh > slen {
			lh -= rh - slen
			rh -= rh - slen
		}
		s = s[lh:rh]
	}
	return s
}
