package encoder

import "testing"

func TestPinyinToFlypy(t *testing.T) {
	cases := []struct {
		initial, final, expected string
	}{
		// Cases from task specification
		{"zh", "ang", "vh"},
		{"ch", "uan", "ir"},
		{"sh", "u", "uu"},
		{"d", "u", "du"},
		{"x", "ue", "xt"},
		{"", "an", "an"},
		{"", "ang", "ah"},
		{"", "a", "aa"},
		{"", "ou", "ou"},

		// Initials mapping
		{"zh", "en", "vf"},
		{"ch", "e", "ie"},
		{"sh", "an", "uj"},
		{"b", "ai", "bd"},
		{"p", "an", "pj"},
		{"m", "ei", "mw"},
		{"f", "eng", "fg"},
		{"t", "ing", "tk"},
		{"n", "iu", "nq"},
		{"l", "ve", "lt"},
		{"g", "uai", "gk"},
		{"k", "uang", "kl"},
		{"h", "ao", "hc"},
		{"j", "iang", "jl"},
		{"q", "in", "qb"},
		{"r", "en", "rf"},
		{"z", "ao", "zc"},
		{"c", "ui", "cv"},
		{"s", "ong", "ss"},
		{"y", "an", "yj"},
		{"m", "ian", "mm"},
		{"w", "o", "wo"},

		// Dual-pinyin shared keys for finals
		{"g", "uo", "go"},
		{"x", "iong", "xs"},
		{"l", "ing", "lk"},
		{"j", "ia", "jx"},
		{"h", "ua", "hx"},
		{"j", "üan", "jr"},
		{"x", "van", "xr"},
	}

	for _, c := range cases {
		got := PinyinToFlypy(c.initial, c.final)
		if got != c.expected {
			t.Errorf("PinyinToFlypy(%q, %q) = %q; want %q", c.initial, c.final, got, c.expected)
		}
	}
}

func TestZeroInitialFlypy(t *testing.T) {
	cases := []struct {
		pinyin, expected string
	}{
		// Single letter: double click
		{"a", "aa"},
		{"o", "oo"},
		{"e", "ee"},

		// Two-letter: itself
		{"ai", "ai"},
		{"ei", "ei"},
		{"ao", "ao"},
		{"ou", "ou"},
		{"an", "an"},
		{"en", "en"},
		{"er", "er"},

		// Three-letter: initial letter + final key
		{"ang", "ah"},
		{"eng", "eg"},

		// Case insensitivity
		{"ANG", "ah"},
		{"An", "an"},
		{"E", "ee"},

		// Fallbacks & edge cases
		{"", ""},
		{"x", "xx"},
		{"xy", "xy"},
		{"xyz", "xyz"},
		{"aang", "ah"},
		{"ing", "ik"},
	}

	for _, c := range cases {
		got := ZeroInitialFlypy(c.pinyin)
		if got != c.expected {
			t.Errorf("ZeroInitialFlypy(%q) = %q; want %q", c.pinyin, got, c.expected)
		}
	}
}

func TestPinyinToFlypy_EdgeCases(t *testing.T) {
	cases := []struct {
		initial, final, expected string
	}{
		{"", "", ""},
		{" ZH ", " ANG ", "vh"},
		{"unknown", "unknown", "unknownunknown"},
		{"d", "unknown", "dunknown"},
		{"unknown", "ang", "unknownh"},
	}
	for _, c := range cases {
		got := PinyinToFlypy(c.initial, c.final)
		if got != c.expected {
			t.Errorf("PinyinToFlypy(%q, %q) = %q; want %q", c.initial, c.final, got, c.expected)
		}
	}
}
