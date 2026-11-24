package encoder

import (
	"testing"
)

func TestIsCommonChineseSurname(t *testing.T) {
	validSurnames := []rune{'王', '李', '张', '刘', '陈', '杨', '赵', '黄', '周', '吴', '柯', '孔'}
	for _, r := range validSurnames {
		if !IsCommonChineseSurname(r) {
			t.Errorf("expected %c to be recognized as common surname", r)
		}
	}

	invalidSurnames := []rune{'一', '不', '我', '这', '的', '是', '了'}
	for _, r := range invalidSurnames {
		if IsCommonChineseSurname(r) {
			t.Errorf("expected %c not to be common surname", r)
		}
	}
}

func TestIsLikelyPersonName(t *testing.T) {
	positiveCases := []string{
		"王洁",
		"张小凡",
		"李明",
		"欧阳修",
		"诸葛亮",
		"陈晨",
		"林志玲",
	}

	for _, w := range positiveCases {
		if !IsLikelyPersonName(w) {
			t.Errorf("expected %q to be recognized as likely person name", w)
		}
	}

	negativeCases := []string{
		"",
		"王",
		"人工智能", // 4 chars
		"深度学习架构",
		"hello",
		"张a",
		"中国",
		"测试",
	}

	for _, w := range negativeCases {
		if IsLikelyPersonName(w) {
			t.Errorf("expected %q NOT to be recognized as likely person name", w)
		}
	}
}
