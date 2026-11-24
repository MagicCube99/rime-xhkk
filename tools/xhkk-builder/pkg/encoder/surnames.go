package encoder

import (
	"strings"
	"unicode"
)

// commonSurnames contains the top 100 most frequent Chinese surnames covering >85% of Chinese population.
var commonSurnames = map[rune]bool{
	'王': true, '李': true, '张': true, '刘': true, '陈': true, '杨': true, '黄': true, '赵': true, '周': true, '吴': true,
	'徐': true, '孙': true, '马': true, '朱': true, '胡': true, '郭': true, '何': true, '高': true, '林': true, '郑': true,
	'梁': true, '谢': true, '宋': true, '唐': true, '许': true, '韩': true, '冯': true, '邓': true, '曹': true, '彭': true,
	'曾': true, '萧': true, '田': true, '董': true, '潘': true, '袁': true, '于': true, '蒋': true, '蔡': true, '余': true,
	'杜': true, '叶': true, '程': true, '苏': true, '魏': true, '吕': true, '丁': true, '任': true, '沈': true, '姚': true,
	'卢': true, '姜': true, '崔': true, '钟': true, '谭': true, '陆': true, '汪': true, '范': true, '金': true, '石': true,
	'廖': true, '贾': true, '夏': true, '韦': true, '付': true, '方': true, '白': true, '邹': true, '孟': true, '熊': true,
	'秦': true, '邱': true, '江': true, '尹': true, '薛': true, '闫': true, '段': true, '雷': true, '侯': true, '龙': true,
	'史': true, '陶': true, '黎': true, '贺': true, '顾': true, '毛': true, '郝': true, '龚': true, '邵': true, '万': true,
	'钱': true, '严': true, '覃': true, '武': true, '戴': true, '莫': true, '孔': true, '向': true, '汤': true, '柯': true,
}

// commonCompoundSurnames contains common 2-character Chinese compound surnames.
var commonCompoundSurnames = map[string]bool{
	"欧阳": true, "诸葛": true, "司马": true, "上官": true, "皇甫": true,
	"申屠": true, "尉迟": true, "公孙": true, "完颜": true, "司徒": true,
	"夏侯": true, "端木": true, "令狐": true, "慕容": true, "长孙": true,
}

// IsCommonChineseSurname reports whether r is in the common Chinese single-character surname set.
func IsCommonChineseSurname(r rune) bool {
	return commonSurnames[r]
}

// IsLikelyPersonName checks heuristically whether the word resembles a Chinese personal name (2 or 3 characters).
func IsLikelyPersonName(word string) bool {
	word = strings.TrimSpace(word)
	runes := []rune(word)
	n := len(runes)

	if n < 2 || n > 3 {
		return false
	}

	for _, r := range runes {
		if !unicode.Is(unicode.Han, r) {
			return false
		}
	}

	// 1. Check compound surname (e.g. 欧阳修)
	if n == 3 && commonCompoundSurnames[string(runes[:2])] {
		return true
	}

	// 2. Check single-character surname (e.g. 王洁, 张小凡)
	if commonSurnames[runes[0]] {
		return true
	}

	return false
}
