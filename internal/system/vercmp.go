package system

import "strings"

// compareKernelVersions orders two kernel version strings (the part after "vmlinuz-", e.g.
// "4.18.0-553.157.1.el8_10.x86_64" or "6.8.0-134-generic") the way RPM's rpmvercmp does. It
// returns -1 if a is older than b, 1 if a is newer, and 0 if they are equal.
//
// "sort -V" is deliberately not used for this: GNU version sort ranks the RHEL point release
// "4.18.0-553.el8_10" above every z-stream update such as "4.18.0-553.157.1.el8_10", because at
// the character after "553." it compares a letter as greater than a digit. rpmvercmp instead
// treats a numeric segment as always newer than an alphabetic one, which matches the order the
// package managers actually use.
func compareKernelVersions(a, b string) int {
	if a == b {
		return 0
	}

	i, j := 0, 0
	for i < len(a) || j < len(b) {
		i = skipSeparators(a, i)
		j = skipSeparators(b, j)

		// A leading '~' sorts before everything else, including the end of the string.
		for (i < len(a) && a[i] == '~') || (j < len(b) && b[j] == '~') {
			if i >= len(a) || a[i] != '~' {
				return 1
			}
			if j >= len(b) || b[j] != '~' {
				return -1
			}
			i++
			j++
		}

		if i >= len(a) || j >= len(b) {
			break
		}

		numeric := isDigit(a[i])
		want := isAlpha
		if numeric {
			want = isDigit
		}

		var segA, segB string
		segA, i = grabSegment(a, i, want)
		segB, j = grabSegment(b, j, want)

		// The other string has no segment of this type here. A numeric segment is newer
		// than a missing one; an alphabetic segment is older than a numeric one.
		if segB == "" {
			if numeric {
				return 1
			}
			return -1
		}

		if diff := compareSegments(segA, segB, numeric); diff != 0 {
			return diff
		}
	}

	switch {
	case i >= len(a) && j >= len(b):
		return 0
	case i >= len(a):
		return -1
	default:
		return 1
	}
}

func compareSegments(segA, segB string, numeric bool) int {
	if numeric {
		segA = strings.TrimLeft(segA, "0")
		segB = strings.TrimLeft(segB, "0")
		if len(segA) != len(segB) {
			if len(segA) > len(segB) {
				return 1
			}
			return -1
		}
	}

	switch {
	case segA == segB:
		return 0
	case segA > segB:
		return 1
	default:
		return -1
	}
}

func skipSeparators(s string, i int) int {
	for i < len(s) && !isDigit(s[i]) && !isAlpha(s[i]) && s[i] != '~' {
		i++
	}
	return i
}

func grabSegment(s string, start int, want func(byte) bool) (segment string, next int) {
	end := start
	for end < len(s) && want(s[end]) {
		end++
	}
	return s[start:end], end
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

func isAlpha(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}
