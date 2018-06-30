package compare

import (
	"unicode/utf8"
)

// diff contains the position info of where two strings differ.
type diff struct {
	// start and end are zero-based indexes that locate the difference in
	// one string when it is compared to another string. The start and end
	// values are intended to be used to slice out the portion of the string
	// that's different and can therefore be used without the need to do bound
	// checking.
	start, end int
}

// The sdiff function finds and retruns the diff between the two provided strings.
// The returned diff will hold the position info of the difference in the "a" string
// argument. In cases where the sole difference lies in the "b" argument being longer
// and so containing extra characters at its end when compared to "a", the retruned
// diff's start and end will equal "a"'s length.
func sdiff(a, b string) *diff {
	if a == b {
		return nil
	}

	// Get the length of the shorter of the two strings.
	length := len(a)
	if length > len(b) {
		length = len(b)
	}

	start, end := -1, -1
	for i := 0; i < length; i++ {
		// Find the first character that's different between the two strings.
		if start == -1 && a[i] != b[i] {
			start = i
			end = i + 1
		}

		// After the first mismatched character find the first one that
		// is the same in the two strings and exit the loop.
		if start > -1 && a[i] == b[i] {
			end = i
			break
		}
	}
	if start == -1 {
		return &diff{length, len(a)}
	}

	if start > -1 {
		// adjust for runes
		r, w := utf8.DecodeRuneInString(a[start:])
		if w > 1 || r == utf8.RuneError {
			for r == utf8.RuneError && start > 0 {
				start -= 1 // back up one byte in "a"
				r, w = utf8.DecodeRuneInString(a[start:])
			}
			if start+w > end {
				end = start + w
			}
		}
	}
	return &diff{start, end}
}

func (d *diff) length() int {
	return d.end - d.start
}

func strim(s string, pos, max int) string {
	length := len(s)
	if length <= max {
		return s
	}

	half := max / 2
	if length > max {
		lh, rh := 0, max
		if pos > half {
			lh = pos - half
			rh = pos + half
		}
		if rh > length {
			lh -= rh - length
			rh -= rh - length
		}
		s = s[lh:rh]
	}
	return s
}
