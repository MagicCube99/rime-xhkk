package encoder

import (
	"errors"
	"fmt"
	"strings"
	"unicode"

	"github.com/mozillazg/go-pinyin"
)

// contextRule defines a pattern-based rule for resolving polyphonic characters in context.
type contextRule struct {
	Char    rune
	Keyword string
	Flypy   string
	Initial string
	Final   string
	Pinyin  string
}

// contextRules stores explicit context heuristics for high-frequency polyphonic words (Layer 3).
var contextRules = []contextRule{
	// 调: diào (dn) vs tiáo (tn)
	{Char: '调', Keyword: "调用", Flypy: "dn", Initial: "d", Final: "iao", Pinyin: "diào"},
	{Char: '调', Keyword: "调度", Flypy: "dn", Initial: "d", Final: "iao", Pinyin: "diào"},
	{Char: '调', Keyword: "调查", Flypy: "dn", Initial: "d", Final: "iao", Pinyin: "diào"},
	{Char: '调', Keyword: "调配", Flypy: "dn", Initial: "d", Final: "iao", Pinyin: "diào"},
	{Char: '调', Keyword: "调动", Flypy: "dn", Initial: "d", Final: "iao", Pinyin: "diào"},
	{Char: '调', Keyword: "音调", Flypy: "dn", Initial: "d", Final: "iao", Pinyin: "diào"},
	{Char: '调', Keyword: "单调", Flypy: "dn", Initial: "d", Final: "iao", Pinyin: "diào"},
	{Char: '调', Keyword: "声调", Flypy: "dn", Initial: "d", Final: "iao", Pinyin: "diào"},
	{Char: '调', Keyword: "调遣", Flypy: "dn", Initial: "d", Final: "iao", Pinyin: "diào"},
	{Char: '调', Keyword: "调任", Flypy: "dn", Initial: "d", Final: "iao", Pinyin: "diào"},
	{Char: '调', Keyword: "调换", Flypy: "dn", Initial: "d", Final: "iao", Pinyin: "diào"},
	{Char: '调', Keyword: "调取", Flypy: "dn", Initial: "d", Final: "iao", Pinyin: "diào"},
	{Char: '调', Keyword: "借调", Flypy: "dn", Initial: "d", Final: "iao", Pinyin: "diào"},
	{Char: '调', Keyword: "下调", Flypy: "dn", Initial: "d", Final: "iao", Pinyin: "diào"},
	{Char: '调', Keyword: "上调", Flypy: "dn", Initial: "d", Final: "iao", Pinyin: "diào"},
	{Char: '调', Keyword: "调令", Flypy: "dn", Initial: "d", Final: "iao", Pinyin: "diào"},
	{Char: '调', Keyword: "协调", Flypy: "dn", Initial: "d", Final: "iao", Pinyin: "diào"},
	{Char: '调', Keyword: "强调", Flypy: "dn", Initial: "d", Final: "iao", Pinyin: "diào"},
	{Char: '调', Keyword: "低调", Flypy: "dn", Initial: "d", Final: "iao", Pinyin: "diào"},
	{Char: '调', Keyword: "高调", Flypy: "dn", Initial: "d", Final: "iao", Pinyin: "diào"},
	{Char: '调', Keyword: "情调", Flypy: "dn", Initial: "d", Final: "iao", Pinyin: "diào"},
	{Char: '调', Keyword: "民调", Flypy: "dn", Initial: "d", Final: "iao", Pinyin: "diào"},
	{Char: '调', Keyword: "跑调", Flypy: "dn", Initial: "d", Final: "iao", Pinyin: "diào"},

	{Char: '调', Keyword: "调整", Flypy: "tn", Initial: "t", Final: "iao", Pinyin: "tiáo"},
	{Char: '调', Keyword: "调节", Flypy: "tn", Initial: "t", Final: "iao", Pinyin: "tiáo"},
	{Char: '调', Keyword: "调和", Flypy: "tn", Initial: "t", Final: "iao", Pinyin: "tiáo"},
	{Char: '调', Keyword: "烹调", Flypy: "tn", Initial: "t", Final: "iao", Pinyin: "tiáo"},
	{Char: '调', Keyword: "调料", Flypy: "tn", Initial: "t", Final: "iao", Pinyin: "tiáo"},
	{Char: '调', Keyword: "调制", Flypy: "tn", Initial: "t", Final: "iao", Pinyin: "tiáo"},
	{Char: '调', Keyword: "调味", Flypy: "tn", Initial: "t", Final: "iao", Pinyin: "tiáo"},
	{Char: '调', Keyword: "风调雨顺", Flypy: "tn", Initial: "t", Final: "iao", Pinyin: "tiáo"},
	{Char: '调', Keyword: "调皮", Flypy: "tn", Initial: "t", Final: "iao", Pinyin: "tiáo"},
	{Char: '调', Keyword: "调理", Flypy: "tn", Initial: "t", Final: "iao", Pinyin: "tiáo"},
	{Char: '调', Keyword: "调剂", Flypy: "tn", Initial: "t", Final: "iao", Pinyin: "tiáo"},
	{Char: '调', Keyword: "调停", Flypy: "tn", Initial: "t", Final: "iao", Pinyin: "tiáo"},

	// 重: chóng (is) vs zhòng (vs)
	{Char: '重', Keyword: "重复", Flypy: "is", Initial: "ch", Final: "ong", Pinyin: "chóng"},
	{Char: '重', Keyword: "重新", Flypy: "is", Initial: "ch", Final: "ong", Pinyin: "chóng"},
	{Char: '重', Keyword: "重启", Flypy: "is", Initial: "ch", Final: "ong", Pinyin: "chóng"},
	{Char: '重', Keyword: "重置", Flypy: "is", Initial: "ch", Final: "ong", Pinyin: "chóng"},
	{Char: '重', Keyword: "重播", Flypy: "is", Initial: "ch", Final: "ong", Pinyin: "chóng"},
	{Char: '重', Keyword: "重写", Flypy: "is", Initial: "ch", Final: "ong", Pinyin: "chóng"},
	{Char: '重', Keyword: "重叠", Flypy: "is", Initial: "ch", Final: "ong", Pinyin: "chóng"},
	{Char: '重', Keyword: "重温", Flypy: "is", Initial: "ch", Final: "ong", Pinyin: "chóng"},
	{Char: '重', Keyword: "重申", Flypy: "is", Initial: "ch", Final: "ong", Pinyin: "chóng"},
	{Char: '重', Keyword: "重构", Flypy: "is", Initial: "ch", Final: "ong", Pinyin: "chóng"},
	{Char: '重', Keyword: "重组", Flypy: "is", Initial: "ch", Final: "ong", Pinyin: "chóng"},
	{Char: '重', Keyword: "重连", Flypy: "is", Initial: "ch", Final: "ong", Pinyin: "chóng"},
	{Char: '重', Keyword: "重读", Flypy: "is", Initial: "ch", Final: "ong", Pinyin: "chóng"},
	{Char: '重', Keyword: "三重", Flypy: "is", Initial: "ch", Final: "ong", Pinyin: "chóng"},
	{Char: '重', Keyword: "重现", Flypy: "is", Initial: "ch", Final: "ong", Pinyin: "chóng"},
	{Char: '重', Keyword: "重重", Flypy: "is", Initial: "ch", Final: "ong", Pinyin: "chóng"},
	{Char: '重', Keyword: "双重", Flypy: "is", Initial: "ch", Final: "ong", Pinyin: "chóng"},
	{Char: '重', Keyword: "重生", Flypy: "is", Initial: "ch", Final: "ong", Pinyin: "chóng"},

	{Char: '重', Keyword: "重要", Flypy: "vs", Initial: "zh", Final: "ong", Pinyin: "zhòng"},
	{Char: '重', Keyword: "重量", Flypy: "vs", Initial: "zh", Final: "ong", Pinyin: "zhòng"},
	{Char: '重', Keyword: "重心", Flypy: "vs", Initial: "zh", Final: "ong", Pinyin: "zhòng"},
	{Char: '重', Keyword: "重力", Flypy: "vs", Initial: "zh", Final: "ong", Pinyin: "zhòng"},
	{Char: '重', Keyword: "严重", Flypy: "vs", Initial: "zh", Final: "ong", Pinyin: "zhòng"},
	{Char: '重', Keyword: "沉重", Flypy: "vs", Initial: "zh", Final: "ong", Pinyin: "zhòng"},
	{Char: '重', Keyword: "轻重", Flypy: "vs", Initial: "zh", Final: "ong", Pinyin: "zhòng"},
	{Char: '重', Keyword: "重点", Flypy: "vs", Initial: "zh", Final: "ong", Pinyin: "zhòng"},
	{Char: '重', Keyword: "尊重", Flypy: "vs", Initial: "zh", Final: "ong", Pinyin: "zhòng"},
	{Char: '重', Keyword: "慎重", Flypy: "vs", Initial: "zh", Final: "ong", Pinyin: "zhòng"},
	{Char: '重', Keyword: "重型", Flypy: "vs", Initial: "zh", Final: "ong", Pinyin: "zhòng"},
	{Char: '重', Keyword: "体重", Flypy: "vs", Initial: "zh", Final: "ong", Pinyin: "zhòng"},
	{Char: '重', Keyword: "负重", Flypy: "vs", Initial: "zh", Final: "ong", Pinyin: "zhòng"},
	{Char: '重', Keyword: "重任", Flypy: "vs", Initial: "zh", Final: "ong", Pinyin: "zhòng"},
	{Char: '重', Keyword: "重视", Flypy: "vs", Initial: "zh", Final: "ong", Pinyin: "zhòng"},
	{Char: '重', Keyword: "重伤", Flypy: "vs", Initial: "zh", Final: "ong", Pinyin: "zhòng"},
	{Char: '重', Keyword: "重病", Flypy: "vs", Initial: "zh", Final: "ong", Pinyin: "zhòng"},
	{Char: '重', Keyword: "重大", Flypy: "vs", Initial: "zh", Final: "ong", Pinyin: "zhòng"},

	// 行: háng (hh) vs xíng (xk)
	{Char: '行', Keyword: "银行", Flypy: "hh", Initial: "h", Final: "ang", Pinyin: "háng"},
	{Char: '行', Keyword: "行业", Flypy: "hh", Initial: "h", Final: "ang", Pinyin: "háng"},
	{Char: '行', Keyword: "行当", Flypy: "hh", Initial: "h", Final: "ang", Pinyin: "háng"},
	{Char: '行', Keyword: "行情", Flypy: "hh", Initial: "h", Final: "ang", Pinyin: "háng"},
	{Char: '行', Keyword: "同行", Flypy: "hh", Initial: "h", Final: "ang", Pinyin: "háng"},
	{Char: '行', Keyword: "商行", Flypy: "hh", Initial: "h", Final: "ang", Pinyin: "háng"},
	{Char: '行', Keyword: "投行", Flypy: "hh", Initial: "h", Final: "ang", Pinyin: "háng"},
	{Char: '行', Keyword: "车行", Flypy: "hh", Initial: "h", Final: "ang", Pinyin: "háng"},
	{Char: '行', Keyword: "琴行", Flypy: "hh", Initial: "h", Final: "ang", Pinyin: "háng"},
	{Char: '行', Keyword: "分行", Flypy: "hh", Initial: "h", Final: "ang", Pinyin: "háng"},
	{Char: '行', Keyword: "总行", Flypy: "hh", Initial: "h", Final: "ang", Pinyin: "háng"},
	{Char: '行', Keyword: "支行", Flypy: "hh", Initial: "h", Final: "ang", Pinyin: "háng"},
	{Char: '行', Keyword: "行规", Flypy: "hh", Initial: "h", Final: "ang", Pinyin: "háng"},
	{Char: '行', Keyword: "隔行", Flypy: "hh", Initial: "h", Final: "ang", Pinyin: "háng"},
	{Char: '行', Keyword: "内行", Flypy: "hh", Initial: "h", Final: "ang", Pinyin: "háng"},
	{Char: '行', Keyword: "外行", Flypy: "hh", Initial: "h", Final: "ang", Pinyin: "háng"},
	{Char: '行', Keyword: "行家", Flypy: "hh", Initial: "h", Final: "ang", Pinyin: "háng"},
	{Char: '行', Keyword: "入行", Flypy: "hh", Initial: "h", Final: "ang", Pinyin: "háng"},
	{Char: '行', Keyword: "字里行间", Flypy: "hh", Initial: "h", Final: "ang", Pinyin: "háng"},

	{Char: '行', Keyword: "行动", Flypy: "xk", Initial: "x", Final: "ing", Pinyin: "xíng"},
	{Char: '行', Keyword: "执行", Flypy: "xk", Initial: "x", Final: "ing", Pinyin: "xíng"},
	{Char: '行', Keyword: "行人", Flypy: "xk", Initial: "x", Final: "ing", Pinyin: "xíng"},
	{Char: '行', Keyword: "行李", Flypy: "xk", Initial: "x", Final: "ing", Pinyin: "xíng"},
	{Char: '行', Keyword: "行程", Flypy: "xk", Initial: "x", Final: "ing", Pinyin: "xíng"},
	{Char: '行', Keyword: "行为", Flypy: "xk", Initial: "x", Final: "ing", Pinyin: "xíng"},
	{Char: '行', Keyword: "行驶", Flypy: "xk", Initial: "x", Final: "ing", Pinyin: "xíng"},
	{Char: '行', Keyword: "流行", Flypy: "xk", Initial: "x", Final: "ing", Pinyin: "xíng"},
	{Char: '行', Keyword: "发行", Flypy: "xk", Initial: "x", Final: "ing", Pinyin: "xíng"},
	{Char: '行', Keyword: "可行", Flypy: "xk", Initial: "x", Final: "ing", Pinyin: "xíng"},
	{Char: '行', Keyword: "举行", Flypy: "xk", Initial: "x", Final: "ing", Pinyin: "xíng"},
	{Char: '行', Keyword: "进行", Flypy: "xk", Initial: "x", Final: "ing", Pinyin: "xíng"},
	{Char: '行', Keyword: "步行", Flypy: "xk", Initial: "x", Final: "ing", Pinyin: "xíng"},
	{Char: '行', Keyword: "行使", Flypy: "xk", Initial: "x", Final: "ing", Pinyin: "xíng"},
	{Char: '行', Keyword: "旅行", Flypy: "xk", Initial: "x", Final: "ing", Pinyin: "xíng"},
	{Char: '行', Keyword: "飞行", Flypy: "xk", Initial: "x", Final: "ing", Pinyin: "xíng"},
	{Char: '行', Keyword: "运行", Flypy: "xk", Initial: "x", Final: "ing", Pinyin: "xíng"},

	// 差: chāi (id) vs chā (ia)
	{Char: '差', Keyword: "出差", Flypy: "id", Initial: "ch", Final: "ai", Pinyin: "chāi"},
	{Char: '差', Keyword: "差事", Flypy: "id", Initial: "ch", Final: "ai", Pinyin: "chāi"},
	{Char: '差', Keyword: "公差", Flypy: "id", Initial: "ch", Final: "ai", Pinyin: "chāi"},
	{Char: '差', Keyword: "差遣", Flypy: "id", Initial: "ch", Final: "ai", Pinyin: "chāi"},
	{Char: '差', Keyword: "差别", Flypy: "ia", Initial: "ch", Final: "a", Pinyin: "chā"},
	{Char: '差', Keyword: "差异", Flypy: "ia", Initial: "ch", Final: "a", Pinyin: "chā"},
	{Char: '差', Keyword: "差距", Flypy: "ia", Initial: "ch", Final: "a", Pinyin: "chā"},
	{Char: '差', Keyword: "差错", Flypy: "ia", Initial: "ch", Final: "a", Pinyin: "chā"},

	// 乐: yuè (yt) vs lè (le)
	{Char: '乐', Keyword: "音乐", Flypy: "yt", Initial: "y", Final: "ue", Pinyin: "yuè"},
	{Char: '乐', Keyword: "乐器", Flypy: "yt", Initial: "y", Final: "ue", Pinyin: "yuè"},
	{Char: '乐', Keyword: "乐队", Flypy: "yt", Initial: "y", Final: "ue", Pinyin: "yuè"},
	{Char: '乐', Keyword: "乐团", Flypy: "yt", Initial: "y", Final: "ue", Pinyin: "yuè"},
	{Char: '乐', Keyword: "奏乐", Flypy: "yt", Initial: "y", Final: "ue", Pinyin: "yuè"},
	{Char: '乐', Keyword: "快乐", Flypy: "le", Initial: "l", Final: "e", Pinyin: "lè"},
	{Char: '乐', Keyword: "乐趣", Flypy: "le", Initial: "l", Final: "e", Pinyin: "lè"},
	{Char: '乐', Keyword: "娱乐", Flypy: "le", Initial: "l", Final: "e", Pinyin: "lè"},
	{Char: '乐', Keyword: "乐观", Flypy: "le", Initial: "l", Final: "e", Pinyin: "lè"},

	// 降: jiàng (jl) vs xiáng (xl)
	{Char: '降', Keyword: "降低", Flypy: "jl", Initial: "j", Final: "iang", Pinyin: "jiàng"},
	{Char: '降', Keyword: "下降", Flypy: "jl", Initial: "j", Final: "iang", Pinyin: "jiàng"},
	{Char: '降', Keyword: "降落", Flypy: "jl", Initial: "j", Final: "iang", Pinyin: "jiàng"},
	{Char: '降', Keyword: "降价", Flypy: "jl", Initial: "j", Final: "iang", Pinyin: "jiàng"},
	{Char: '降', Keyword: "投降", Flypy: "xl", Initial: "x", Final: "iang", Pinyin: "xiáng"},
	{Char: '降', Keyword: "降服", Flypy: "xl", Initial: "x", Final: "iang", Pinyin: "xiáng"},
}

// Disambiguate performs 4-layer polyphonic disambiguation on the given Chinese word.
// It returns a slice of CharPinyin structs containing the selected Flypy double-pinyin,
// initial, final, and alternative candidates for each character.
func Disambiguate(word string) ([]CharPinyin, error) {
	word = strings.TrimSpace(word)
	if len(word) == 0 {
		return nil, errors.New("word cannot be empty")
	}

	runes := []rune(word)
	for _, r := range runes {
		if !unicode.Is(unicode.Han, r) {
			return nil, fmt.Errorf("word contains non-Chinese character: %c", r)
		}
	}

	initPrimaryIndex()

	n := len(runes)
	targetFlypys := make([]string, n)
	resolved := make([]bool, n)
	targetInitials := make([]string, n)
	targetFinals := make([]string, n)
	targetPinyins := make([]string, n)

	// Layer 3: Context heuristics for high-frequency polyphonic words
	for _, rule := range contextRules {
		kwRunes := []rune(rule.Keyword)
		kwLen := len(kwRunes)
		for i := 0; i <= n-kwLen; i++ {
			if string(runes[i:i+kwLen]) == rule.Keyword {
				for j := 0; j < kwLen; j++ {
					if kwRunes[j] == rule.Char && !resolved[i+j] {
						targetFlypys[i+j] = rule.Flypy
						targetInitials[i+j] = rule.Initial
						targetFinals[i+j] = rule.Final
						targetPinyins[i+j] = rule.Pinyin
						resolved[i+j] = true
					}
				}
			}
		}
	}

	// Layer 1: Phrase dictionary matching (2-character subphrase matches from flypy.primary)
	for i := 0; i < n-1; i++ {
		if !resolved[i] || !resolved[i+1] {
			sub := string(runes[i : i+2])
			if codes, ok := GetPhraseFlypy(sub); ok && len(codes) == 2 {
				if !resolved[i] {
					targetFlypys[i] = codes[0]
					resolved[i] = true
				}
				if !resolved[i+1] {
					targetFlypys[i+1] = codes[1]
					resolved[i+1] = true
				}
			}
		}
	}

	// Layer 2: flypy.primary natural sequence fallback (official standard single-character sequence)
	for i := 0; i < n; i++ {
		if !resolved[i] {
			if pc, ok := GetPrimaryChar(runes[i]); ok {
				targetFlypys[i] = pc.Flypy
				targetInitials[i] = pc.Initial
				targetFinals[i] = pc.Final
				targetPinyins[i] = pc.Pinyin
				resolved[i] = true
			}
		}
	}

	// Layer 4 / go-pinyin fallback: rare characters outside the 7,468 primary characters
	for i := 0; i < n; i++ {
		if !resolved[i] {
			pys := pinyin.LazyPinyin(string(runes[i]), pinyin.NewArgs())
			if len(pys) > 0 {
				init, fin := SplitPinyin(pys[0])
				targetFlypys[i] = PinyinToFlypy(init, fin)
				targetInitials[i] = init
				targetFinals[i] = fin
				targetPinyins[i] = pys[0]
				resolved[i] = true
			} else {
				// Extreme fallback: double rune character
				targetFlypys[i] = strings.ToLower(string(runes[i]))
			}
		}
	}

	// Step 2: Build CharPinyin objects and populate Alternates / Candidates
	result := make([]CharPinyin, n)
	for i := 0; i < n; i++ {
		r := runes[i]
		chosenFlypy := targetFlypys[i]
		chosenInit := targetInitials[i]
		chosenFin := targetFinals[i]
		chosenPy := targetPinyins[i]

		// Query heteronym candidates from go-pinyin
		aNorm := pinyin.NewArgs()
		aNorm.Heteronym = true
		aNorm.Style = pinyin.Normal
		normPys := pinyin.SinglePinyin(r, aNorm)

		aTone := pinyin.NewArgs()
		aTone.Heteronym = true
		aTone.Style = pinyin.Tone
		tonePys := pinyin.SinglePinyin(r, aTone)

		// If initial and final are not yet set, attempt to find a matching pronunciation
		if chosenInit == "" && chosenFin == "" {
			for j, py := range normPys {
				init, fin := SplitPinyin(py)
				if PinyinToFlypy(init, fin) == chosenFlypy {
					chosenInit = init
					chosenFin = fin
					if j < len(tonePys) {
						chosenPy = tonePys[j]
					} else {
						chosenPy = py
					}
					break
				}
			}
			// If still empty, fall back to SplitPinyin using PrimaryChar or chosenFlypy
			if chosenInit == "" && chosenFin == "" {
				if pc, ok := GetPrimaryChar(r); ok && pc.Flypy == chosenFlypy {
					chosenInit = pc.Initial
					chosenFin = pc.Final
					chosenPy = pc.Pinyin
				}
			}
		}

		// Collect alternative pronunciations for polyphonic character capsule UI
		candidates := []string{chosenFlypy}
		alternates := make([]CharPinyin, 0, len(normPys))
		seenFlypy := map[string]bool{chosenFlypy: true}

		for j, py := range normPys {
			init, fin := SplitPinyin(py)
			fp := PinyinToFlypy(init, fin)
		if !IsLivingPolyphonicFlypy(r, fp) {
			continue
		}
			if !seenFlypy[fp] {
				seenFlypy[fp] = true
				candidates = append(candidates, fp)
				altTone := py
				if j < len(tonePys) {
					altTone = tonePys[j]
				}
				alternates = append(alternates, CharPinyin{
					Char:    r,
					Initial: init,
					Final:   fin,
					Flypy:   fp,
					Pinyin:  altTone,
				})
			}
		}

		result[i] = CharPinyin{
			Char:       r,
			Initial:    chosenInit,
			Final:      chosenFin,
			Flypy:      chosenFlypy,
			Pinyin:     chosenPy,
			Candidates: candidates,
			Alternates: alternates,
		}
	}

	return result, nil
}
