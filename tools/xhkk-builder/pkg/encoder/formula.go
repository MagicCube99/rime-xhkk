package encoder

import (
	"errors"
	"fmt"
	"strings"
)

// getCharFlypy extracts or calculates the 2-letter Flypy double-pinyin code for a CharPinyin.
// If Flypy is empty, it falls back to computing it from Initial and Final using PinyinToFlypy.
func getCharFlypy(cp CharPinyin) (string, error) {
	code := strings.ToLower(strings.TrimSpace(cp.Flypy))
	if code == "" {
		code = PinyinToFlypy(cp.Initial, cp.Final)
	}
	r := []rune(code)
	if len(r) < 2 {
		return "", fmt.Errorf("invalid flypy code %q for character %c: length must be at least 2", code, cp.Char)
	}
	return code, nil
}

// DeriveCodes computes the Flypy word codes according to the word length and target dictionary.
// Formula rules:
// - len(wordChars) < 2: returns error ("word length must be at least 2").
// - 2-char word: AaAbBaBb -> c1[0:2] + c2[0:2] (4 letters), secondaryCode is empty.
// - 3-char word:
//   - If targetDict == "flypy.names":
//   - primaryCode: AaBaCa (3 letters: c1[0] + c2[0] + c3[0])
//   - secondaryCode: AaBaCaCb (4 letters: c1[0] + c2[0] + c3[0:2])
//   - Else:
//   - primaryCode: AaBaCaCb (4 letters: c1[0] + c2[0] + c3[0:2])
//   - secondaryCode: empty ("")
//
// - 4-char and above words: AaBaCaZa -> c1[0] + c2[0] + c3[0] + cN[0] (4 letters), secondaryCode is empty.
func DeriveCodes(wordChars []CharPinyin, targetDict string) (primaryCode string, secondaryCode string, err error) {
	n := len(wordChars)
	if n < 2 {
		return "", "", errors.New("word length must be at least 2")
	}

	codeRunes := make([][]rune, n)
	for i, ch := range wordChars {
		c, err := getCharFlypy(ch)
		if err != nil {
			return "", "", err
		}
		codeRunes[i] = []rune(c)
	}

	targetDict = strings.TrimSpace(targetDict)

	switch n {
	case 2:
		primaryCode = string(codeRunes[0][:2]) + string(codeRunes[1][:2])
		secondaryCode = ""
		return primaryCode, secondaryCode, nil

	case 3:
		if targetDict == "flypy.names" {
			primaryCode = string(codeRunes[0][0]) + string(codeRunes[1][0]) + string(codeRunes[2][0])
			secondaryCode = string(codeRunes[0][0]) + string(codeRunes[1][0]) + string(codeRunes[2][:2])
			return primaryCode, secondaryCode, nil
		}
		primaryCode = string(codeRunes[0][0]) + string(codeRunes[1][0]) + string(codeRunes[2][:2])
		secondaryCode = ""
		return primaryCode, secondaryCode, nil

	default: // 4 or more characters
		primaryCode = string(codeRunes[0][0]) + string(codeRunes[1][0]) + string(codeRunes[2][0]) + string(codeRunes[n-1][0])
		secondaryCode = ""
		return primaryCode, secondaryCode, nil
	}
}
