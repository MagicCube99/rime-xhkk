package encoder

import (
	"testing"
)

func TestDisambiguateCommonTrapWords(t *testing.T) {
	traps := map[string]string{
		"读写":   "duxp", // 读误取 dou
		"安全合规": "aqhg", // 合误取 ge
		"服务调用": "fwdy", // 调误取 tiao
		"健康检查": "jkji", // 查误取 zha
		"单体架构": "dtjg", // 单误取 chan
		"无损发布": "wsfb", // 无误取 mo
		"权限提升": "qxtu", // 提误取 di
	}
	for word, expectedCode := range traps {
		chars, err := Disambiguate(word)
		if err != nil {
			t.Fatalf("Disambiguate(%q) failed: %v", word, err)
		}
		code, _, err := DeriveCodes(chars, "flypy.custom")
		if err != nil {
			t.Fatalf("DeriveCodes failed for %q: %v", word, err)
		}
		if code != expectedCode {
			t.Errorf("word %q got %q; want %q", word, code, expectedCode)
		}
	}
}

func TestDisambiguateContextHeuristics(t *testing.T) {
	cases := map[string]string{
		// 调: diào (dn) vs tiáo (tn)
		"调整": "tnvg",
		"调节": "tnjp",
		"调用": "dnys",
		"调度": "dndu",
		"调查": "dnia",

		// 重: chóng (is) vs zhòng (vs)
		"重复": "isfu",
		"重新": "isxb",
		"重量": "vsll",
		"重要": "vsyc",

		// 行: háng (hh) vs xíng (xk)
		"银行": "ybhh",
		"行业": "hhye",
		"行动": "xkds",
		"执行": "vixk",

		// 差: chāi (id) vs chā (ia)
		"出差": "iuid",
		"差别": "iabp",

		// 乐: yuè (yt) vs lè (le)
		"音乐": "ybyt",
		"快乐": "kkle",
	}

	for word, expectedCode := range cases {
		chars, err := Disambiguate(word)
		if err != nil {
			t.Fatalf("Disambiguate(%q) failed: %v", word, err)
		}
		code, _, err := DeriveCodes(chars, "flypy.custom")
		if err != nil {
			t.Fatalf("DeriveCodes failed for %q: %v", word, err)
		}
		if code != expectedCode {
			t.Errorf("word %q got %q; want %q", word, code, expectedCode)
		}
	}
}

func TestDisambiguatePolyphonicAlternates(t *testing.T) {
	// Test that polyphonic characters return candidate alternatives for UI capsules
	chars, err := Disambiguate("服务调用")
	if err != nil {
		t.Fatalf("Disambiguate failed: %v", err)
	}

	// Character at index 2 is '调'
	diaoChar := chars[2]
	if diaoChar.Char != '调' {
		t.Fatalf("expected char '调' at index 2, got %c", diaoChar.Char)
	}
	if diaoChar.Flypy != "dn" {
		t.Errorf("expected chosen Flypy 'dn' for '调', got %q", diaoChar.Flypy)
	}

	// Verify Alternates and Candidates contain the alternative "tn" (tiáo)
	hasAltTn := false
	for _, alt := range diaoChar.Alternates {
		if alt.Flypy == "tn" {
			hasAltTn = true
			if alt.Initial != "t" || alt.Final != "iao" {
				t.Errorf("expected alt 'tn' initial='t', final='iao', got initial=%q, final=%q", alt.Initial, alt.Final)
			}
		}
	}
	if !hasAltTn {
		t.Errorf("expected '调' to have alternate 'tn' in Alternates, got %+v", diaoChar.Alternates)
	}

	hasCandTn := false
	for _, c := range diaoChar.Candidates {
		if c == "tn" {
			hasCandTn = true
		}
	}
	if !hasCandTn {
		t.Errorf("expected '调' to have 'tn' in Candidates, got %v", diaoChar.Candidates)
	}

	// Verify non-polyphonic character or primary character alternates
	fuChar := chars[0]
	if fuChar.Char != '服' {
		t.Fatalf("expected char '服' at index 0, got %c", fuChar.Char)
	}
	if fuChar.Flypy != "fu" {
		t.Errorf("expected chosen Flypy 'fu' for '服', got %q", fuChar.Flypy)
	}
}

func TestDisambiguateNamesWorkflow(t *testing.T) {
	chars, err := Disambiguate("张三丰")
	if err != nil {
		t.Fatalf("Disambiguate failed: %v", err)
	}

	primary, secondary, err := DeriveCodes(chars, "flypy.names")
	if err != nil {
		t.Fatalf("DeriveCodes failed: %v", err)
	}
	if primary != "vsf" {
		t.Errorf("expected primary code 'vsf', got %q", primary)
	}
	if secondary != "vsfg" {
		t.Errorf("expected secondary code 'vsfg', got %q", secondary)
	}
}

func TestDisambiguateErrors(t *testing.T) {
	// Empty string
	if _, err := Disambiguate(""); err == nil {
		t.Error("expected error for empty string, got nil")
	}

	// Whitespace only
	if _, err := Disambiguate("   "); err == nil {
		t.Error("expected error for whitespace string, got nil")
	}

	// Non-Chinese characters
	if _, err := Disambiguate("abc"); err == nil {
		t.Error("expected error for English string, got nil")
	}
	if _, err := Disambiguate("深度学习123"); err == nil {
		t.Error("expected error for string with digits, got nil")
	}
}

func TestPrimaryIndexIntegrity(t *testing.T) {
	initPrimaryIndex()

	if len(primaryCharIndex) != 7468 {
		t.Errorf("expected exactly 7468 primary characters, got %d", len(primaryCharIndex))
	}

	// Verify key trap characters in primary index
	expected := map[rune]string{
		'读': "du",
		'合': "he",
		'调': "dn",
		'查': "ia",
		'单': "dj",
		'无': "wu",
		'提': "ti",
		'红': "hs",
	}

	for r, expFp := range expected {
		pc, ok := GetPrimaryChar(r)
		if !ok {
			t.Errorf("character %c not found in primaryCharIndex", r)
			continue
		}
		if pc.Flypy != expFp {
			t.Errorf("character %c expected flypy %q, got %q", r, expFp, pc.Flypy)
		}
	}
}

func TestUnambiguousCharacters(t *testing.T) {
	chars, err := Disambiguate("王洁")
	if err != nil {
		t.Fatal(err)
	}
	if len(chars) != 2 {
		t.Fatalf("expected 2 chars, got %d", len(chars))
	}
	if len(chars[0].Candidates) != 1 || chars[0].Flypy != "wh" {
		t.Errorf("expected 王 to have strictly 1 candidate 'wh', got %v", chars[0].Candidates)
	}
	if len(chars[1].Candidates) != 1 || chars[1].Flypy != "jp" {
		t.Errorf("expected 洁 to have strictly 1 candidate 'jp', got %v", chars[1].Candidates)
	}
}

