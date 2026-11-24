package encoder

import (
	"strings"
	"testing"
)

func TestDeriveCodes(t *testing.T) {
	// 测试2字词 AaAbBaBb: 深度 -> ud + du = uddu
	charsShenDu := []CharPinyin{
		{Char: '深', Initial: "sh", Final: "en", Flypy: "ud"},
		{Char: '度', Initial: "d", Final: "u", Flypy: "du"},
	}
	code1, code2, err := DeriveCodes(charsShenDu, "flypy.custom")
	if err != nil || code1 != "uddu" || code2 != "" {
		t.Fatalf("want uddu, '', got %q, %q, err: %v", code1, code2, err)
	}

	// 测试人名3字双码: 张三丰 (names) -> 3码 vsf, 4码 vsfg
	charsZhangSanFeng := []CharPinyin{
		{Char: '张', Initial: "zh", Final: "ang", Flypy: "vh"},
		{Char: '三', Initial: "s", Final: "an", Flypy: "sj"},
		{Char: '丰', Initial: "f", Final: "eng", Flypy: "fg"},
	}
	c3, c4, err := DeriveCodes(charsZhangSanFeng, "flypy.names")
	if err != nil || c3 != "vsf" || c4 != "vsfg" {
		t.Fatalf("want vsf, vsfg, got %q, %q, err: %v", c3, c4, err)
	}

	// 测试普通3字词 AaBaCaCb: 计算机 -> j + s + ji = jsji
	charsJiSuanJi := []CharPinyin{
		{Char: '计', Initial: "j", Final: "i", Flypy: "ji"},
		{Char: '算', Initial: "s", Final: "uan", Flypy: "sr"},
		{Char: '机', Initial: "j", Final: "i", Flypy: "ji"},
	}
	code1, code2, err = DeriveCodes(charsJiSuanJi, "flypy.custom")
	if err != nil || code1 != "jsji" || code2 != "" {
		t.Fatalf("want jsji, '', got %q, %q, err: %v", code1, code2, err)
	}

	// 测试4字词 AaBaCaZa: 深度学习 -> u + d + x + x = udxx
	charsShenDuXueXi := []CharPinyin{
		{Char: '深', Initial: "sh", Final: "en", Flypy: "ud"},
		{Char: '度', Initial: "d", Final: "u", Flypy: "du"},
		{Char: '学', Initial: "x", Final: "ue", Flypy: "xt"},
		{Char: '习', Initial: "x", Final: "i", Flypy: "xi"},
	}
	code1, code2, err = DeriveCodes(charsShenDuXueXi, "flypy.custom")
	if err != nil || code1 != "udxx" || code2 != "" {
		t.Fatalf("want udxx, '', got %q, %q, err: %v", code1, code2, err)
	}

	// 测试5字词 AaBaCaZa: 人工智能化 -> r + r + v + h = rrvh
	charsRenGong := []CharPinyin{
		{Char: '人', Initial: "r", Final: "en", Flypy: "rf"},
		{Char: '工', Initial: "g", Final: "ong", Flypy: "gs"},
		{Char: '智', Initial: "zh", Final: "i", Flypy: "vi"},
		{Char: '能', Initial: "n", Final: "eng", Flypy: "ng"},
		{Char: '化', Initial: "h", Final: "ua", Flypy: "hx"},
	}
	code1, code2, err = DeriveCodes(charsRenGong, "flypy.custom")
	if err != nil || code1 != "rgvh" || code2 != "" {
		t.Fatalf("want rgvh, '', got %q, %q, err: %v", code1, code2, err)
	}

	// 测试6字词 AaBaCaZa: 卷积神经网络 -> j + j + u + l = jjul
	charsJuanJi := []CharPinyin{
		{Char: '卷', Initial: "j", Final: "uan", Flypy: "jr"},
		{Char: '积', Initial: "j", Final: "i", Flypy: "ji"},
		{Char: '神', Initial: "sh", Final: "en", Flypy: "uf"},
		{Char: '经', Initial: "j", Final: "ing", Flypy: "jk"},
		{Char: '网', Initial: "w", Final: "ang", Flypy: "wh"},
		{Char: '络', Initial: "l", Final: "uo", Flypy: "lo"},
	}
	code1, code2, err = DeriveCodes(charsJuanJi, "flypy.custom")
	if err != nil || code1 != "jjul" || code2 != "" {
		t.Fatalf("want jjul, '', got %q, %q, err: %v", code1, code2, err)
	}
}

func TestDeriveCodes_FallbackFromPinyin(t *testing.T) {
	// When Flypy is empty, it should be computed from Initial and Final
	// 深: sh + en -> uf, 度: d + u -> du => ufdu
	chars := []CharPinyin{
		{Char: '深', Initial: "sh", Final: "en", Flypy: ""},
		{Char: '度', Initial: "d", Final: "u", Flypy: ""},
	}
	code1, code2, err := DeriveCodes(chars, "flypy.custom")
	if err != nil || code1 != "ufdu" || code2 != "" {
		t.Fatalf("want ufdu, '', got %q, %q, err: %v", code1, code2, err)
	}
}

func TestDeriveCodes_ZeroInitials(t *testing.T) {
	// Zero initial word: 安全 (an, qv) -> anqv
	charsAnQuan := []CharPinyin{
		{Char: '安', Initial: "", Final: "an", Flypy: "an"},
		{Char: '全', Initial: "q", Final: "uan", Flypy: "qr"},
	}
	code1, code2, err := DeriveCodes(charsAnQuan, "flypy.custom")
	if err != nil || code1 != "anqr" || code2 != "" {
		t.Fatalf("want anqr, '', got %q, %q, err: %v", code1, code2, err)
	}

	// Zero initial 3-word: 天安门 (tm, an, mf) -> t + a + mf = tamf
	charsTianAnMen := []CharPinyin{
		{Char: '天', Initial: "t", Final: "ian", Flypy: "tm"},
		{Char: '安', Initial: "", Final: "an", Flypy: "an"},
		{Char: '门', Initial: "m", Final: "en", Flypy: "mf"},
	}
	code1, code2, err = DeriveCodes(charsTianAnMen, "flypy.custom")
	if err != nil || code1 != "tamf" || code2 != "" {
		t.Fatalf("want tamf, '', got %q, %q, err: %v", code1, code2, err)
	}

	// Zero initial 4-word: 阿里巴巴 (aa, li, ba, ba) -> a + l + b + b = albb
	charsAli := []CharPinyin{
		{Char: '阿', Initial: "", Final: "a", Flypy: "aa"},
		{Char: '里', Initial: "l", Final: "i", Flypy: "li"},
		{Char: '巴', Initial: "b", Final: "a", Flypy: "ba"},
		{Char: '巴', Initial: "b", Final: "a", Flypy: "ba"},
	}
	code1, code2, err = DeriveCodes(charsAli, "flypy.custom")
	if err != nil || code1 != "albb" || code2 != "" {
		t.Fatalf("want albb, '', got %q, %q, err: %v", code1, code2, err)
	}
}

func TestDeriveCodes_NamesNon3Char(t *testing.T) {
	// 2-character name in flypy.names should still produce 4-char primary, no secondary
	charsLiBai := []CharPinyin{
		{Char: '李', Initial: "l", Final: "i", Flypy: "li"},
		{Char: '白', Initial: "b", Final: "ai", Flypy: "bd"},
	}
	code1, code2, err := DeriveCodes(charsLiBai, "flypy.names")
	if err != nil || code1 != "libd" || code2 != "" {
		t.Fatalf("want libd, '', got %q, %q, err: %v", code1, code2, err)
	}

	// 4-character name in flypy.names should produce 4-char primary, no secondary
	charsOuyang := []CharPinyin{
		{Char: '欧', Initial: "", Final: "ou", Flypy: "ou"},
		{Char: '阳', Initial: "y", Final: "ang", Flypy: "yh"},
		{Char: '修', Initial: "x", Final: "iu", Flypy: "xq"},
		{Char: '之', Initial: "zh", Final: "i", Flypy: "vi"},
	}
	code1, code2, err = DeriveCodes(charsOuyang, "flypy.names")
	if err != nil || code1 != "oyxv" || code2 != "" {
		t.Fatalf("want oyxv, '', got %q, %q, err: %v", code1, code2, err)
	}
}

func TestDeriveCodes_Errors(t *testing.T) {
	// Empty slice
	_, _, err := DeriveCodes(nil, "flypy.custom")
	if err == nil || !strings.Contains(err.Error(), "word length must be at least 2") {
		t.Fatalf("expected 'word length must be at least 2', got: %v", err)
	}

	// 1-character word
	charsOne := []CharPinyin{
		{Char: '中', Initial: "zh", Final: "ong", Flypy: "vs"},
	}
	_, _, err = DeriveCodes(charsOne, "flypy.custom")
	if err == nil || !strings.Contains(err.Error(), "word length must be at least 2") {
		t.Fatalf("expected 'word length must be at least 2', got: %v", err)
	}

	// Invalid Flypy code (< 2 chars and cannot be computed)
	charsInvalid := []CharPinyin{
		{Char: '深', Initial: "", Final: "", Flypy: "u"},
		{Char: '度', Initial: "d", Final: "u", Flypy: "du"},
	}
	_, _, err = DeriveCodes(charsInvalid, "flypy.custom")
	if err == nil {
		t.Fatal("expected error for invalid flypy code, got nil")
	}
}
