package encoder

import (
	_ "embed"
	"strings"
	"sync"
)

//go:embed assets/primary_chars.tsv
var primaryCharsTSV string

//go:embed assets/phrase_exceptions.tsv
var phraseExceptionsTSV string

//go:embed assets/living_polyphonics.tsv
var livingPolyphonicsTSV string

// PrimaryChar stores the canonical Flypy double-pinyin data for a single Chinese character.
type PrimaryChar struct {
	Char    rune
	Flypy   string
	Initial string
	Final   string
	Pinyin  string
}

var (
	primaryCharIndex      map[rune]PrimaryChar
	phraseIndex           map[string][]string // 2-char word -> [char1Flypy, char2Flypy]
	livingPolyphonicIndex map[rune]map[string]bool // char -> set of valid living flypy codes
	initOnce              sync.Once
)

// initPrimaryIndex loads and parses the embedded primary character and phrase dictionary.
func initPrimaryIndex() {
	initOnce.Do(func() {
		primaryCharIndex = make(map[rune]PrimaryChar, 8000)
		phraseIndex = make(map[string][]string, 2000)
		livingPolyphonicIndex = make(map[rune]map[string]bool, 500)

		// 1. Load primary single characters
		charLines := strings.Split(primaryCharsTSV, "\n")
		for _, line := range charLines {
			line = strings.TrimSpace(line)
			if len(line) == 0 {
				continue
			}
			parts := strings.Split(line, "\t")
			if len(parts) >= 2 {
				runes := []rune(parts[0])
				if len(runes) == 1 {
					r := runes[0]
					fp := parts[1]
					init := ""
					fin := ""
					py := ""
					if len(parts) >= 4 {
						init = parts[2]
						fin = parts[3]
					}
					if len(parts) >= 5 {
						py = parts[4]
					}
					primaryCharIndex[r] = PrimaryChar{
						Char:    r,
						Flypy:   fp,
						Initial: init,
						Final:   fin,
						Pinyin:  py,
					}
				}
			}
		}

		// 2. Load exception phrases (words that deviate from single-character natural order)
		phraseLines := strings.Split(phraseExceptionsTSV, "\n")
		for _, line := range phraseLines {
			line = strings.TrimSpace(line)
			if len(line) == 0 {
				continue
			}
			parts := strings.Split(line, "\t")
			if len(parts) >= 3 {
				phraseIndex[parts[0]] = []string{parts[1], parts[2]}
			}
		}

		// 3. Load genuine living polyphonic characters from official dictionary
		polyLines := strings.Split(livingPolyphonicsTSV, "\n")
		for _, line := range polyLines {
			line = strings.TrimSpace(line)
			if len(line) == 0 {
				continue
			}
			parts := strings.Split(line, "\t")
			if len(parts) >= 2 {
				runes := []rune(parts[0])
				if len(runes) == 1 {
					r := runes[0]
					codes := strings.Split(parts[1], ",")
					codeMap := make(map[string]bool, len(codes))
					for _, c := range codes {
						codeMap[c] = true
					}
					livingPolyphonicIndex[r] = codeMap
				}
			}
		}
	})
}

// GetPrimaryChar returns the official standard Flypy record for a single character from flypy.primary.
func GetPrimaryChar(r rune) (PrimaryChar, bool) {
	initPrimaryIndex()
	pc, ok := primaryCharIndex[r]
	return pc, ok
}

// GetPhraseFlypy returns the official 2-character Flypy double-pinyin slice for a compound word if present.
func GetPhraseFlypy(phrase string) ([]string, bool) {
	initPrimaryIndex()
	codes, ok := phraseIndex[phrase]
	return codes, ok
}

// IsLivingPolyphonicFlypy reports whether character r has the given Flypy double-pinyin reading
// in the living official dictionary corpus.
func IsLivingPolyphonicFlypy(r rune, flypy string) bool {
	initPrimaryIndex()
	if codeMap, ok := livingPolyphonicIndex[r]; ok {
		return codeMap[flypy]
	}
	return false
}
