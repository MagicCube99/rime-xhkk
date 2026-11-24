package encoder

import "strings"

// CharPinyin represents a single Chinese character with its pinyin components and flypy double-pinyin code.
type CharPinyin struct {
	Char       rune
	Initial    string
	Final      string
	Flypy      string
	Pinyin     string       // Full pinyin syllable with tone (e.g. "dú", "diào")
	Candidates []string     // Alternative double-pinyin codes for polyphonic characters (e.g. ["dn", "tn"])
	Alternates []CharPinyin // Alternative CharPinyin representations for polyphonic UI switching
}

// InitialMap maps Chinese pinyin compound initials to Flypy single key representations.
// Standard Flypy mappings: zh -> v, ch -> i, sh -> u.
// Single-letter initials stay identical.
var InitialMap = map[string]string{
	"zh": "v",
	"ch": "i",
	"sh": "u",
	"b":  "b",
	"p":  "p",
	"m":  "m",
	"f":  "f",
	"d":  "d",
	"t":  "t",
	"n":  "n",
	"l":  "l",
	"g":  "g",
	"k":  "k",
	"h":  "h",
	"j":  "j",
	"q":  "q",
	"x":  "x",
	"r":  "r",
	"z":  "z",
	"c":  "c",
	"s":  "s",
	"y":  "y",
	"w":  "w",
}

// FinalMap maps Chinese pinyin finals to Flypy single key representations.
var FinalMap = map[string]string{
	"iu":   "q",
	"ei":   "w",
	"e":    "e",
	"uan":  "r",
	"van":  "r",
	"üan":  "r",
	"ue":   "t",
	"ve":   "t",
	"üe":   "t",
	"un":   "y",
	"ün":   "y",
	"vn":   "y",
	"u":    "u",
	"i":    "i",
	"o":    "o",
	"uo":   "o",
	"ie":   "p",
	"a":    "a",
	"ong":  "s",
	"iong": "s",
	"ai":   "d",
	"en":   "f",
	"eng":  "g",
	"ang":  "h",
	"an":   "j",
	"uai":  "k",
	"ing":  "k",
	"iang": "l",
	"uang": "l",
	"ou":   "z",
	"ia":   "x",
	"ua":   "x",
	"ao":   "c",
	"ui":   "v",
	"v":    "v",
	"ü":    "v",
	"in":   "b",
	"iao":  "n",
	"ian":  "m",
	"er":   "er",
}

// ZeroInitialFlypy converts a zero-initial pinyin syllable to its 2-letter flypy code.
// Rules:
// - Single-letter: double click (a -> aa, o -> oo, e -> ee)
// - Two-letter: itself (ai -> ai, ei -> ei, ao -> ao, ou -> ou, an -> an, en -> en, er -> er)
// - Three-letter: initial letter + final key (ang -> a + h = ah, eng -> e + g = eg)
func ZeroInitialFlypy(pinyin string) string {
	p := strings.ToLower(strings.TrimSpace(pinyin))
	switch p {
	case "a":
		return "aa"
	case "o":
		return "oo"
	case "e":
		return "ee"
	case "ai", "ei", "ao", "ou", "an", "en", "er":
		return p
	case "ang":
		return "ah"
	case "eng":
		return "eg"
	default:
		if len(p) == 1 {
			return p + p
		}
		if len(p) == 2 {
			return p
		}
		if len(p) > 2 {
			if finKey, ok := FinalMap[p[1:]]; ok {
				return string(p[0]) + finKey
			}
			if finKey, ok := FinalMap[p]; ok {
				return string(p[0]) + finKey
			}
		}
		return p
	}
}

// toneReplacer replaces accented pinyin vowels with plain letters.
var toneReplacer = strings.NewReplacer(
	"ā", "a", "á", "a", "ǎ", "a", "à", "a",
	"ē", "e", "é", "e", "ě", "e", "è", "e",
	"ī", "i", "í", "i", "ǐ", "i", "ì", "i",
	"ō", "o", "ó", "o", "ǒ", "o", "ò", "o",
	"ū", "u", "ú", "u", "ǔ", "u", "ù", "u",
	"ǖ", "v", "ǘ", "v", "ǚ", "v", "ǜ", "v", "ü", "v",
)

// NormalizePinyin removes tone marks and tone numbers, returning a lowercase plain pinyin string.
func NormalizePinyin(pinyin string) string {
	p := strings.ToLower(strings.TrimSpace(pinyin))
	p = toneReplacer.Replace(p)
	p = strings.TrimRight(p, "0123456789")
	return p
}

// SplitPinyin splits a pinyin syllable (with or without tones) into initial and final components.
// For zero-initial syllables (starting with a, o, e), initial is empty string.
func SplitPinyin(pinyin string) (initial string, final string) {
	p := NormalizePinyin(pinyin)
	if len(p) == 0 {
		return "", ""
	}
	// Compound initials
	if strings.HasPrefix(p, "zh") || strings.HasPrefix(p, "ch") || strings.HasPrefix(p, "sh") {
		return p[:2], p[2:]
	}
	// Single consonant initials / y / w
	if strings.ContainsRune("bpmfdtnlgkhjqxrzcsyw", rune(p[0])) {
		return p[:1], p[1:]
	}
	// Zero initial (vowels: a, o, e)
	return "", p
}

// PinyinToFlypy converts an initial and final component into a 2-letter Flypy double-pinyin code.
// If initial is empty, it delegates to ZeroInitialFlypy with the final string.
func PinyinToFlypy(initial string, final string) string {
	initial = strings.ToLower(strings.TrimSpace(initial))
	final = strings.ToLower(strings.TrimSpace(final))

	if initial == "" {
		return ZeroInitialFlypy(final)
	}

	initKey, ok := InitialMap[initial]
	if !ok {
		initKey = initial
	}

	finKey, ok := FinalMap[final]
	if !ok {
		finKey = final
	}

	return initKey + finKey
}
