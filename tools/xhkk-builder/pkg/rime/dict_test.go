package rime

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDictOperations(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "rime-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// 初始化测试词库文件
	customFile := filepath.Join(tmpDir, "flypy_dicts", "flypy.custom.dict.yaml")
	_ = os.MkdirAll(filepath.Dir(customFile), 0755)
	_ = os.WriteFile(customFile, []byte("旧词\tjiuc\n"), 0644)

	// 1. 查重已存在词
	status, detail := CheckStatus(tmpDir, "旧词", "jiuc")
	if status != StatusWordExists {
		t.Fatalf("want StatusWordExists, got %v (%s)", status, detail)
	}

	// 2. 写入新词与人名双码
	err = AppendWord(tmpDir, "flypy.names", "张三丰", []string{"vsf", "vsfg"})
	if err != nil {
		t.Fatalf("AppendWord failed: %v", err)
	}

	content, _ := os.ReadFile(filepath.Join(tmpDir, "flypy_dicts", "flypy.names.dict.yaml"))
	if !strings.Contains(string(content), "张三丰\tvsf\n张三丰\tvsfg\n") {
		t.Fatalf("unexpected content:\n%s", string(content))
	}
}

func TestCheckStatus_DetailedScenarios(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "rime-check-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	dictsDir := filepath.Join(tmpDir, "flypy_dicts")
	_ = os.MkdirAll(dictsDir, 0755)

	customContent := `# Rime dictionary
---
name: flypy.custom
version: "1.0"
...
自用词	zyci
双码词	smc1
双码词	smc2
`
	namesContent := `# Rime dictionary
---
name: flypy.names
version: "1.0"
...
张三丰	vsf
李四	lisi
`
	primaryContent := `# Rime dictionary
---
name: flypy.primary
version: "1.0"
...
深度	uddu
适度	uddu
`
	_ = os.WriteFile(filepath.Join(dictsDir, "flypy.custom.dict.yaml"), []byte(customContent), 0644)
	_ = os.WriteFile(filepath.Join(dictsDir, "flypy.names.dict.yaml"), []byte(namesContent), 0644)
	_ = os.WriteFile(filepath.Join(dictsDir, "flypy.primary.dict.yaml"), []byte(primaryContent), 0644)

	// 1. Existing word in custom with exact code
	st, detail := CheckStatus(tmpDir, "自用词", "zyci")
	if st != StatusWordExists || detail != "zyci" {
		t.Errorf("want StatusWordExists with zyci, got %v (%s)", st, detail)
	}

	// 2. Existing word in names
	st, detail = CheckStatus(tmpDir, "张三丰", "vsf")
	if st != StatusWordExists || detail != "vsf" {
		t.Errorf("want StatusWordExists with vsf, got %v (%s)", st, detail)
	}

	// 3. Existing word in primary
	st, detail = CheckStatus(tmpDir, "深度", "uddu")
	if st != StatusWordExists || detail != "uddu" {
		t.Errorf("want StatusWordExists with uddu, got %v (%s)", st, detail)
	}

	// 4. Existing word with multiple codes - querying second code
	st, detail = CheckStatus(tmpDir, "双码词", "smc2")
	if st != StatusWordExists || detail != "smc2" {
		t.Errorf("want StatusWordExists with smc2, got %v (%s)", st, detail)
	}

	// 5. Existing word queried with unrecorded code
	st, detail = CheckStatus(tmpDir, "自用词", "other")
	if st != StatusWordExists || detail != "zyci" {
		t.Errorf("want StatusWordExists with zyci, got %v (%s)", st, detail)
	}

	// 6. Code collision with custom dictionary
	st, detail = CheckStatus(tmpDir, "自由词", "zyci")
	if st != StatusCodeCollision || detail != "自用词" {
		t.Errorf("want StatusCodeCollision with 自用词, got %v (%s)", st, detail)
	}

	// 7. Code collision with primary dictionary - should return first candidate
	st, detail = CheckStatus(tmpDir, "制度", "uddu")
	if st != StatusCodeCollision || detail != "深度" {
		t.Errorf("want StatusCodeCollision with 深度, got %v (%s)", st, detail)
	}

	// 8. Brand new word and code
	st, detail = CheckStatus(tmpDir, "全新的词", "qxci")
	if st != StatusNew || detail != "" {
		t.Errorf("want StatusNew with empty detail, got %v (%s)", st, detail)
	}

	// 9. Edge cases: empty word or empty code
	st, detail = CheckStatus(tmpDir, "", "zyci")
	if st != StatusNew || detail != "" {
		t.Errorf("want StatusNew for empty word, got %v (%s)", st, detail)
	}

	st, detail = CheckStatus(tmpDir, "自用词", "")
	if st != StatusWordExists || detail != "zyci" {
		t.Errorf("want StatusWordExists with zyci for empty code, got %v (%s)", st, detail)
	}

	st, detail = CheckStatus(tmpDir, "全新的词", "")
	if st != StatusNew || detail != "" {
		t.Errorf("want StatusNew for brand new word with empty code, got %v (%s)", st, detail)
	}
}

func TestAppendWord_SelfHealingWithExample(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "rime-example-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	dictsDir := filepath.Join(tmpDir, "flypy_dicts")
	_ = os.MkdirAll(dictsDir, 0755)

	exampleContent := `# Rime dictionary
# encoding: utf-8
---
name: flypy.custom
version: "1.0"
sort: by_weight
use_preset_vocabulary: false
...

# 示例：
我的密码	wdmm
常用邮箱	cyyx
`
	exampleFile := filepath.Join(dictsDir, "flypy.custom.dict.yaml.example")
	_ = os.WriteFile(exampleFile, []byte(exampleContent), 0644)

	// AppendWord should copy the example file first, then append the new entry
	err = AppendWord(tmpDir, "flypy.custom", "内部代号", []string{"nbdh"})
	if err != nil {
		t.Fatalf("AppendWord failed: %v", err)
	}

	customFile := filepath.Join(dictsDir, "flypy.custom.dict.yaml")
	contentBytes, err := os.ReadFile(customFile)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	content := string(contentBytes)

	if !strings.Contains(content, "我的密码\twdmm") {
		t.Errorf("expected copied example content, got:\n%s", content)
	}
	if !strings.HasSuffix(content, "内部代号\tnbdh\n") {
		t.Errorf("expected appended entry with newline, got:\n%s", content)
	}

	// Verify that CheckStatus immediately sees both example words and newly added word
	st, _ := CheckStatus(tmpDir, "我的密码", "wdmm")
	if st != StatusWordExists {
		t.Errorf("want StatusWordExists for example word, got %v", st)
	}
	st, _ = CheckStatus(tmpDir, "内部代号", "nbdh")
	if st != StatusWordExists {
		t.Errorf("want StatusWordExists for newly appended word, got %v", st)
	}
}

func TestAppendWord_SelfHealingWithoutExample(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "rime-no-example-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	err = AppendWord(tmpDir, "flypy.custom", "测试词", []string{"csci"})
	if err != nil {
		t.Fatalf("AppendWord failed: %v", err)
	}

	contentBytes, err := os.ReadFile(filepath.Join(tmpDir, "flypy_dicts", "flypy.custom.dict.yaml"))
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	content := string(contentBytes)

	expectedHeader := "# Rime dictionary\n# encoding: utf-8\n---\nname: flypy.custom\nversion: \"1.0\"\nsort: by_weight\nuse_preset_vocabulary: false\n...\n"
	if !strings.HasPrefix(content, expectedHeader) {
		t.Errorf("expected standard header prefix, got:\n%s", content)
	}
	if !strings.HasSuffix(content, "测试词\tcsci\n") {
		t.Errorf("expected appended entry with newline, got:\n%s", content)
	}
}

func TestAppendWord_MissingTrailingNewline(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "rime-nonewline-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	dictsDir := filepath.Join(tmpDir, "flypy_dicts")
	_ = os.MkdirAll(dictsDir, 0755)

	// Existing file without trailing newline
	customFile := filepath.Join(dictsDir, "flypy.custom.dict.yaml")
	_ = os.WriteFile(customFile, []byte("第一行\tdyh"), 0644)

	err = AppendWord(tmpDir, "flypy.custom", "第二行", []string{"deh"})
	if err != nil {
		t.Fatalf("AppendWord failed: %v", err)
	}

	contentBytes, _ := os.ReadFile(customFile)
	content := string(contentBytes)
	expected := "第一行\tdyh\n第二行\tdeh\n"
	if content != expected {
		t.Errorf("expected clean separation:\nwant %q\ngot  %q", expected, content)
	}
}

func TestAppendWord_Errors(t *testing.T) {
	if err := AppendWord("", "flypy.custom", "词", []string{"ci"}); err == nil {
		t.Error("expected error for empty rimeDir")
	}
	if err := AppendWord("some/dir", "flypy.custom", "", []string{"ci"}); err == nil {
		t.Error("expected error for empty word")
	}
	if err := AppendWord("some/dir", "flypy.custom", "词", nil); err == nil {
		t.Error("expected error for empty codes")
	}
}

func TestStatusTypeString(t *testing.T) {
	if StatusNew.String() != "StatusNew" {
		t.Errorf("expected StatusNew, got %s", StatusNew.String())
	}
	if StatusWordExists.String() != "StatusWordExists" {
		t.Errorf("expected StatusWordExists, got %s", StatusWordExists.String())
	}
	if StatusCodeCollision.String() != "StatusCodeCollision" {
		t.Errorf("expected StatusCodeCollision, got %s", StatusCodeCollision.String())
	}
	unknown := StatusType(99)
	if !strings.Contains(unknown.String(), "99") {
		t.Errorf("expected StatusType(99), got %s", unknown.String())
	}
}

func TestCheckStatus_DirectRimeDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "rime-direct-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// File placed directly under rimeDir (no flypy_dicts subfolder)
	customFile := filepath.Join(tmpDir, "flypy.custom.dict.yaml")
	_ = os.WriteFile(customFile, []byte("直属词\tzsci\n"), 0644)

	st, detail := CheckStatus(tmpDir, "直属词", "zsci")
	if st != StatusWordExists || detail != "zsci" {
		t.Errorf("want StatusWordExists, got %v (%s)", st, detail)
	}

	st, detail = CheckStatus("", "直属词", "zsci")
	if st != StatusNew || detail != "" {
		t.Errorf("want StatusNew for empty rimeDir, got %v (%s)", st, detail)
	}
}

func TestFindUserDictWordAndRemoveWord(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "rime-find-remove-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	dictsDir := filepath.Join(tmpDir, "flypy_dicts")
	_ = os.MkdirAll(dictsDir, 0755)

	// Setup custom, names, and user.top
	_ = os.WriteFile(filepath.Join(dictsDir, "flypy.custom.dict.yaml"), []byte("张三\tvs\n"), 0644)
	_ = os.WriteFile(filepath.Join(dictsDir, "flypy.names.dict.yaml"), []byte("李四\tls\n"), 0644)
	_ = os.WriteFile(filepath.Join(dictsDir, "flypy.user.top.dict.yaml"), []byte("李四\tls\n"), 0644)

	// 1. Find 张三 in custom (not top)
	dictName, codes, isTop := FindUserDictWord(tmpDir, "张三")
	if dictName != "flypy.custom" || len(codes) != 1 || codes[0] != "vs" || isTop {
		t.Errorf("unexpected FindUserDictWord for 张三: dict=%s, codes=%v, isTop=%v", dictName, codes, isTop)
	}

	// 2. Find 李四 in names and top
	dictName, codes, isTop = FindUserDictWord(tmpDir, "李四")
	if dictName != "flypy.names" || len(codes) != 1 || codes[0] != "ls" || !isTop {
		t.Errorf("unexpected FindUserDictWord for 李四: dict=%s, codes=%v, isTop=%v", dictName, codes, isTop)
	}

	// 3. Remove 张三 from custom
	err = RemoveWord(tmpDir, "flypy.custom", "张三")
	if err != nil {
		t.Fatalf("RemoveWord failed: %v", err)
	}

	// Verify 张三 is gone from custom
	dictName, _, _ = FindUserDictWord(tmpDir, "张三")
	if dictName != "" {
		t.Errorf("expected 张三 to be removed from custom, but found in %s", dictName)
	}

	// Verify file content does not have 张三
	cData, _ := os.ReadFile(filepath.Join(dictsDir, "flypy.custom.dict.yaml"))
	if strings.Contains(string(cData), "张三") {
		t.Errorf("expected file not to contain 张三, got: %s", string(cData))
	}
}

func TestConcurrentAppendAndCheck(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "rime-concur-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	const n = 20
	errCh := make(chan error, n)

	for i := 0; i < n; i++ {
		go func(idx int) {
			word := fmt.Sprintf("并发词%d", idx)
			code := fmt.Sprintf("bf%02d", idx)
			err := AppendWord(tmpDir, "flypy.custom", word, []string{code})
			if err != nil {
				errCh <- err
				return
			}
			st, _ := CheckStatus(tmpDir, word, code)
			if st != StatusWordExists {
				errCh <- fmt.Errorf("expected StatusWordExists for %s, got %v", word, st)
				return
			}
			errCh <- nil
		}(i)
	}

	for i := 0; i < n; i++ {
		if err := <-errCh; err != nil {
			t.Errorf("concurrent operation failed: %v", err)
		}
	}
}

func TestCheckStatus_ModularDicts(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "rime-modular-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	dictsDir := filepath.Join(tmpDir, "flypy_dicts")
	_ = os.MkdirAll(dictsDir, 0755)

	// Write entries into modular dictionaries
	_ = os.WriteFile(filepath.Join(dictsDir, "flypy.user.top.dict.yaml"), []byte("置顶词\tzdci\n"), 0644)
	_ = os.WriteFile(filepath.Join(dictsDir, "flypy.official.dict.yaml"), []byte("公文词\tgwci\n"), 0644)
	_ = os.WriteFile(filepath.Join(dictsDir, "flypy.it.dict.yaml"), []byte("架构师\tjgsi\n"), 0644)
	_ = os.WriteFile(filepath.Join(dictsDir, "flypy.places.dict.yaml"), []byte("西湖区\txhqu\n"), 0644)

	tests := []struct {
		word string
		code string
	}{
		{"置顶词", "zdci"},
		{"公文词", "gwci"},
		{"架构师", "jgsi"},
		{"西湖区", "xhqu"},
	}

	for _, tc := range tests {
		st, detail := CheckStatus(tmpDir, tc.word, tc.code)
		if st != StatusWordExists || detail != tc.code {
			t.Errorf("expected StatusWordExists for %s (%s), got %v (%s)", tc.word, tc.code, st, detail)
		}
	}
}
