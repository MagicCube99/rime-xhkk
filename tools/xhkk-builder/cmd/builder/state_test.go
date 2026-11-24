package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"rime-xhkk-builder/pkg/rime"
)

func createValidRimeDir(t *testing.T) string {

	t.Helper()

	dir := t.TempDir()

	_ = os.WriteFile(filepath.Join(dir, "flypy.dict.yaml"), []byte("---\nname: flypy\nversion: \"1.0\"\n...\n"), 0644)

	return dir

}

func TestInitialState(t *testing.T) {

	tempDir := createValidRimeDir(t)

	s := NewAppState(tempDir)

	if s.Word != "" {

		t.Errorf("expected empty initial word, got %q", s.Word)

	}

	if s.TargetDict != TargetCustom {

		t.Errorf("expected TargetCustom initially, got %q", s.TargetDict)

	}

	if s.BadgeText != "【≥2字中文】" {

		t.Errorf("expected badge '[仅支持2字及以上中文]', got %q", s.BadgeText)

	}

	if s.CanSubmit {

		t.Errorf("expected CanSubmit to be false initially, got true")

	}

	if s.PrimaryCode != "" {

		t.Errorf("expected empty PrimaryCode, got %q", s.PrimaryCode)

	}

}

func TestInvalidInputs(t *testing.T) {

	tempDir := createValidRimeDir(t)

	s := NewAppState(tempDir)

	testCases := []string{

		"",

		" ",

		"中",

		"测",

		"hello",

		"1234",

		"中a",

		"测 试",

		"词组!",
	}

	for _, tc := range testCases {

		s.SetWord(tc)

		if s.CanSubmit {

			t.Errorf("input %q should not be submittable", tc)

		}

		if s.BadgeKind != BadgeGray {

			t.Errorf("input %q should have BadgeGray, got %v", tc, s.BadgeKind)

		}

		if !strings.Contains(s.BadgeText, "≥2字") && !strings.Contains(s.BadgeText, "非法") {

			t.Errorf("input %q should have invalid badge, got %q", tc, s.BadgeText)

		}

		if s.PrimaryCode != "" {

			t.Errorf("input %q should have empty PrimaryCode, got %q", tc, s.PrimaryCode)

		}

	}

}

func TestValidNewWord(t *testing.T) {

	tempDir := createValidRimeDir(t)

	s := NewAppState(tempDir)

	s.SetWord("人工智能")

	if !s.CanSubmit {

		t.Errorf("expected valid word '人工智能' to be submittable")

	}

	if s.BadgeKind != BadgeGreen {

		t.Errorf("expected BadgeGreen for new word, got %v", s.BadgeKind)

	}

	if s.BadgeText != "【未收录】" {

		t.Errorf("expected badge '【未收录】', got %q", s.BadgeText)

	}

	if len(s.PrimaryCode) != 4 {

		t.Errorf("expected 4-letter primary code, got %q", s.PrimaryCode)

	}

	if s.DisplayCode != s.PrimaryCode {

		t.Errorf("expected DisplayCode %q, got %q", s.PrimaryCode, s.DisplayCode)

	}

	if len(s.CharStates) != 4 {

		t.Errorf("expected 4 CharStates, got %d", len(s.CharStates))

	}

}

func TestWordExists(t *testing.T) {

	tempDir := createValidRimeDir(t)

	dictsDir := filepath.Join(tempDir, "flypy_dicts")

	if err := os.MkdirAll(dictsDir, 0755); err != nil {

		t.Fatalf("failed to create dicts dir: %v", err)

	}

	customDict := filepath.Join(dictsDir, "flypy.custom.dict.yaml")

	content := "# Rime dictionary\n---\nname: flypy.custom\nversion: \"1.0\"\n...\n深度\tuddu\n"

	if err := os.WriteFile(customDict, []byte(content), 0644); err != nil {

		t.Fatalf("failed to write dict file: %v", err)

	}

	s := NewAppState(tempDir)

	s.SetWord("深度")

	if s.Status != rime.StatusWordExists {

		t.Errorf("expected StatusWordExists, got %v", s.Status)

	}

	if s.BadgeKind != BadgeRed {

		t.Errorf("expected BadgeRed, got %v", s.BadgeKind)

	}

	if !strings.Contains(s.BadgeText, "已存在") {

		t.Errorf("expected badge to mention '已存在', got %q", s.BadgeText)

	}

	if s.CanSubmit {

		t.Errorf("existing word must not be submittable")

	}

}

func TestCodeCollision(t *testing.T) {

	tempDir := createValidRimeDir(t)

	dictsDir := filepath.Join(tempDir, "flypy_dicts")

	if err := os.MkdirAll(dictsDir, 0755); err != nil {

		t.Fatalf("failed to create dicts dir: %v", err)

	}

	customDict := filepath.Join(dictsDir, "flypy.custom.dict.yaml")

	content := "# Rime dictionary\n---\nname: flypy.custom\nversion: \"1.0\"\n...\n首选词\tufdu\n"

	if err := os.WriteFile(customDict, []byte(content), 0644); err != nil {

		t.Fatalf("failed to write dict file: %v", err)

	}

	s := NewAppState(tempDir)

	s.SetWord("深度") // derives code "uddu"

	if s.Status != rime.StatusCodeCollision {

		t.Errorf("expected StatusCodeCollision, got %v", s.Status)

	}

	if s.BadgeKind != BadgeYellow {

		t.Errorf("expected BadgeYellow, got %v", s.BadgeKind)

	}

	if !strings.Contains(s.BadgeText, "重码") || !strings.Contains(s.BadgeText, "首选词") {

		t.Errorf("expected badge to mention collision with '首选词', got %q", s.BadgeText)

	}

	if !s.CanSubmit {

		t.Errorf("code collision should still be submittable as secondary candidate")

	}

}

func TestPolyphonicReadingCycle(t *testing.T) {

	tempDir := createValidRimeDir(t)

	s := NewAppState(tempDir)

	s.SetWord("服务调用")

	if len(s.CharStates) != 4 {

		t.Fatalf("expected 4 CharStates, got %d", len(s.CharStates))

	}

	diaoState := s.CharStates[2]

	if diaoState.Char != '调' {

		t.Fatalf("expected char '调' at index 2, got %c", diaoState.Char)

	}

	if diaoState.Flypy != "dn" {

		t.Fatalf("expected chosen Flypy 'dn' for '调', got %q", diaoState.Flypy)

	}

	initialCode := s.PrimaryCode

	// Cycle reading of '调'

	s.CycleReading(2)

	afterCycle1 := s.CharStates[2]

	if afterCycle1.Flypy != "tn" {

		t.Errorf("expected cycled Flypy to be 'tn', got %q", afterCycle1.Flypy)

	}

	cycledCode := s.PrimaryCode

	if cycledCode == initialCode {

		t.Errorf("primary code should have changed after cycling polyphonic character, got %q", cycledCode)

	}

	if !strings.HasPrefix(cycledCode, "fwt") {

		t.Errorf("expected cycled code to start with 'fwt', got %q", cycledCode)

	}

	// Cycle reading again (should cycle back)

	s.CycleReading(2)

	afterCycle2 := s.CharStates[2]

	if afterCycle2.Flypy != "dn" {

		t.Errorf("expected Flypy to return to 'dn', got %q", afterCycle2.Flypy)

	}

	if s.PrimaryCode != initialCode {

		t.Errorf("expected PrimaryCode to return to %q, got %q", initialCode, s.PrimaryCode)

	}

}

func TestTargetDictSwitching(t *testing.T) {

	tempDir := createValidRimeDir(t)

	s := NewAppState(tempDir)

	s.SetWord("张小凡")

	if s.TargetDict != TargetCustom {

		t.Errorf("expected TargetCustom, got %q", s.TargetDict)

	}

	if len(s.PrimaryCode) != 4 {

		t.Errorf("expected 4-char PrimaryCode in TargetCustom, got %q", s.PrimaryCode)

	}

	if s.SecondaryCode != "" {

		t.Errorf("expected empty SecondaryCode in TargetCustom, got %q", s.SecondaryCode)

	}

	// Switch to TargetNames and opt in to 3-code

	s.SetTargetDict(TargetNames)

	s.SetIncludeShortCode(true)

	if s.TargetDict != TargetNames {

		t.Errorf("expected TargetNames, got %q", s.TargetDict)

	}

	if len(s.PrimaryCode) != 3 {

		t.Errorf("expected 3-char PrimaryCode in TargetNames, got %q", s.PrimaryCode)

	}

	if len(s.SecondaryCode) != 4 {

		t.Errorf("expected 4-char SecondaryCode in TargetNames, got %q", s.SecondaryCode)

	}

	if !strings.Contains(s.DisplayCode, s.PrimaryCode) || !strings.Contains(s.DisplayCode, s.SecondaryCode) {

		t.Errorf("DisplayCode should contain both primary and secondary codes, got %q", s.DisplayCode)

	}

	// Switch back

	s.SetTargetDict(TargetCustom)

	if len(s.PrimaryCode) != 4 || s.SecondaryCode != "" {

		t.Errorf("expected switch back to have 4-char primary and empty secondary, got primary=%q, secondary=%q", s.PrimaryCode, s.SecondaryCode)

	}

}

func TestSubmitLifecycle(t *testing.T) {

	tempDir := createValidRimeDir(t)

	s := NewAppState(tempDir)

	s.SetWord("自我实现")

	if !s.CanSubmit {

		t.Fatalf("word should be submittable")

	}

	expectedCode := s.PrimaryCode

	submitted := s.Submit(nil)

	if !submitted {

		t.Fatalf("Submit returned false")

	}

	if s.Word != "" {

		t.Errorf("expected s.Word to be cleared, got %q", s.Word)

	}

	if s.CanSubmit {

		t.Errorf("expected CanSubmit to be false after submit")

	}

	if s.BadgeText != "【≥2字中文】" {

		t.Errorf("expected reset badge, got %q", s.BadgeText)

	}

	filePath := filepath.Join(tempDir, "flypy_dicts", "flypy.custom.dict.yaml")

	data, err := os.ReadFile(filePath)

	if err != nil {

		t.Fatalf("failed to read generated dict file: %v", err)

	}

	content := string(data)

	expectedEntry := "自我实现\t" + expectedCode

	if !strings.Contains(content, expectedEntry) {

		t.Errorf("expected dict file to contain %q, got:\n%s", expectedEntry, content)

	}

	time.Sleep(50 * time.Millisecond)

	s.mu.RLock()

	deployMsg := s.DeployStatusMsg

	s.mu.RUnlock()

	if !strings.Contains(deployMsg, "已添加") && !strings.Contains(deployMsg, "已写入") {

		t.Errorf("expected deploy status message to mention '已添加' or '已写入', got %q", deployMsg)

	}

}

func TestBuildUIRendering(t *testing.T) {

	tempDir := createValidRimeDir(t)

	s := NewAppState(tempDir)

	s.SetWord("测试词条")

	tt := ui.NewTester(func(c *ui.Context) {

		BuildUI(c, s, nil)

	}, 490, 204)

	tt.Frame()

	if !tt.HasText("新词") {

		t.Errorf("expected UI to contain label '新词'")

	}

	if !tt.HasText("编码") {

		t.Errorf("expected UI to contain label '编码'")

	}

	if !tt.HasText("读音") {

		t.Errorf("expected UI to contain label '读音'")

	}

	if !tt.HasText("自用") || !tt.HasText("人名") {

		t.Errorf("expected UI to contain radio labels '自用' and '人名'")

	}

	if !tt.HasText("添加并部署") {

		t.Errorf("expected UI to contain primary submit button")

	}

	if !tt.HasText("【未收录】") {

		t.Errorf("expected UI to contain badge '【未收录】', texts: %v", tt.Texts())

	}

	if !tt.HasText("用户文件夹:") {

		t.Errorf("expected UI to contain '用户文件夹:'")

	}

	if !tt.HasText("自定义词库:") || !tt.HasText("自用词库 ↗") || !tt.HasText("人名词库 ↗") || !tt.HasText("置顶词库 ↗") {

		t.Errorf("expected UI to contain dictionary text links")

	}

	if !tt.HasText("小狼毫服务:") {

		t.Errorf("expected UI to contain '小狼毫服务:'")

	}

	if !tt.HasText("部署服务已就绪") && !tt.HasText("未检测到部署服务") {

		t.Errorf("expected UI to contain deployer status, texts: %v", tt.Texts())

	}

	if !tt.HasText("算法服务已就绪") && !tt.HasText("算法服务未运行") && !tt.HasText("未检测到算法服务") {

		t.Errorf("expected UI to contain server status, texts: %v", tt.Texts())

	}

}

func TestClearTransientStatus(t *testing.T) {
	tempDir := createValidRimeDir(t)
	s := NewAppState(tempDir)

	s.DeployStatusMsg = "已还原至默认用户文件夹"
	s.SetWord("新词测试")
	if s.DeployStatusMsg != "" {
		t.Errorf("expected transient status to be cleared on SetWord, got %q", s.DeployStatusMsg)
	}

	s.DeployStatusMsg = "已切换用户文件夹"
	s.SetTargetDict(TargetNames)
	if s.DeployStatusMsg != "" {
		t.Errorf("expected transient status to be cleared on SetTargetDict, got %q", s.DeployStatusMsg)
	}

	// Persistent error should NOT be cleared on SetWord
	s.DeployStatusMsg = "目录无效 (未找到小鹤方案)"
	s.SetWord("另一个词")
	if s.DeployStatusMsg != "目录无效 (未找到小鹤方案)" {
		t.Errorf("expected persistent error to remain, got %q", s.DeployStatusMsg)
	}
}

func TestBuildUIWithInvalidRimeDir(t *testing.T) {
	tempDir := t.TempDir()
	emptySubDir := filepath.Join(tempDir, "empty_folder")
	_ = os.MkdirAll(emptySubDir, 0755)

	s := NewAppState(emptySubDir)
	s.DeployStatusMsg = "目录无效 (未找到小鹤方案)"

	tt := ui.NewTester(func(c *ui.Context) {
		BuildUI(c, s, nil)
	}, 490, 204)
	tt.Frame()

	if !tt.HasText("[无效]") {
		t.Errorf("expected UI to contain '[无效]' indicator for invalid directory")
	}
	if !tt.HasText("目录无效 (未找到小鹤方案)") {
		t.Errorf("expected UI to contain invalid directory status message")
	}
}

func TestMultiWordBatch(t *testing.T) {

	tempDir := createValidRimeDir(t)

	s := NewAppState(tempDir)

	s.SetWord("张颖 李明 王伟")

	if !s.IsBatch {

		t.Fatalf("expected s.IsBatch to be true")

	}

	if len(s.BatchItems) != 3 {

		t.Fatalf("expected 3 batch items, got %d", len(s.BatchItems))

	}

	if !s.CanSubmit {

		t.Fatalf("expected multi-word to be submittable")

	}

	submitted := s.Submit(nil)

	if !submitted {

		t.Fatalf("expected batch submit to succeed")

	}

	filePath := filepath.Join(tempDir, "flypy_dicts", "flypy.custom.dict.yaml")

	data, err := os.ReadFile(filePath)

	if err != nil {

		t.Fatalf("failed to read dict file: %v", err)

	}

	content := string(data)

	if !strings.Contains(content, "张颖") || !strings.Contains(content, "李明") || !strings.Contains(content, "王伟") {

		t.Errorf("expected all words written, got:\n%s", content)

	}

}

func TestNamesShortCodeCollisionFallback(t *testing.T) {

	tempDir := createValidRimeDir(t)

	primaryFile := filepath.Join(tempDir, "flypy_dicts", "flypy.primary.dict.yaml")

	_ = os.MkdirAll(filepath.Dir(primaryFile), 0755)

	_ = os.WriteFile(primaryFile, []byte("文件夹\twjj\n"), 0644)

	s := NewAppState(tempDir)

	s.SetTargetDict(TargetNames)

	s.SetWord("翁嘉捷")

	if !s.HasShortCode {

		t.Fatalf("expected HasShortCode to be true for 3-char name")

	}

	if s.ShortCode != "wjj" {

		t.Errorf("expected ShortCode to be 'wjj', got %q", s.ShortCode)

	}

	if s.FullCode != "wjjp" {

		t.Errorf("expected FullCode to be 'wjjp', got %q", s.FullCode)

	}

	if !s.HasShortCodeCollision {

		t.Errorf("expected HasShortCodeCollision to be true due to collision with '文件夹'")

	}

	if s.ShortCodeCollisionWord != "文件夹" {

		t.Errorf("expected ShortCodeCollisionWord to be '文件夹', got %q", s.ShortCodeCollisionWord)

	}

	// Default Scheme 1: IncludeShortCode should be FALSE by default to protect system word!

	if s.IncludeShortCode {

		t.Errorf("expected IncludeShortCode to be false by default when collision occurs")

	}

	if s.DisplayCode != "wjjp" {

		t.Errorf("expected DisplayCode to be 'wjjp' (only 4-code), got %q", s.DisplayCode)

	}

	if !s.CanSubmit {

		t.Errorf("expected word to still be submittable under 4-code")

	}

	// User opt-in override: check IncludeShortCode

	s.SetIncludeShortCode(true)

	if s.DisplayCode != "wjj  wjjp" {

		t.Errorf("expected DisplayCode to be 'wjj  wjjp', got %q", s.DisplayCode)

	}

	if !strings.Contains(s.BadgeText, "3码顶替") || !strings.Contains(s.BadgeText, "文件夹") {

		t.Errorf("expected badge to warn about 3码顶替, got %q", s.BadgeText)

	}

	// Submit with override: should write both lines

	s.Submit(nil)

	content, _ := os.ReadFile(filepath.Join(tempDir, "flypy_dicts", "flypy.names.dict.yaml"))

	if !strings.Contains(string(content), "翁嘉捷\twjj\n翁嘉捷\twjjp\n") {

		t.Errorf("expected both codes written when opted in, got:\n%s", string(content))

	}

}

func TestRimeDirValidation(t *testing.T) {

	tempDir := t.TempDir()

	emptySubDir := filepath.Join(tempDir, "empty_folder")

	_ = os.MkdirAll(emptySubDir, 0755)

	s := NewAppState(emptySubDir)

	if s.RimeDirValid {

		t.Errorf("expected empty folder to be invalid Rime dir")

	}

	// Top badge strictly reports word status (never hijacked by directory error)
	if s.BadgeText != "【≥2字中文】" {
		t.Errorf("expected empty input to show badge '【≥2字中文】', got %q", s.BadgeText)
	}

	s.SetWord("测试词条")
	if s.CanSubmit {
		t.Errorf("expected CanSubmit to be false for invalid Rime dir")
	}
	if s.BadgeText != "【未核查】" {
		t.Errorf("expected word badge '【未核查】' for invalid dir, got %q", s.BadgeText)
	}
	if s.BadgeKind != BadgeGray {
		t.Errorf("expected BadgeGray for invalid dir, got %v", s.BadgeKind)
	}

	// Verify OpenFile is blocked when RimeDir is invalid
	errOpen := s.OpenFile(TargetCustom)
	if errOpen == nil {
		t.Errorf("expected OpenFile to fail when RimeDir is invalid")
	}

	// Verify that Redeploy is strictly blocked when RimeDir is invalid
	s.DeployStatusMsg = "初始状态"
	s.Redeploy(nil)
	if s.DeployStatusMsg != "初始状态" {
		t.Errorf("expected Redeploy to be blocked when RimeDir is invalid, got %q", s.DeployStatusMsg)
	}

}

func TestSwitchRimeDirDynamicValidation(t *testing.T) {
	validDir := createValidRimeDir(t)
	s := NewAppState(validDir)
	if !s.RimeDirValid {
		t.Fatalf("expected initial dir to be valid")
	}

	// Enter word on valid dir
	s.SetWord("测试词条")
	if s.BadgeText != "【未收录】" {
		t.Errorf("expected valid dir to show '【未收录】', got %q", s.BadgeText)
	}

	// Switch to invalid dir dynamically
	invalidDir := t.TempDir()
	s.SetRimeDir(invalidDir)
	if s.RimeDirValid {
		t.Errorf("expected dynamically switched invalid dir to set RimeDirValid=false")
	}
	if s.BadgeText != "【未核查】" {
		t.Errorf("expected dynamically switched invalid dir to degrade badge to '【未核查】', got %q", s.BadgeText)
	}

	// UI rendering with invalid dir
	tt := ui.NewTester(func(c *ui.Context) {
		BuildUI(c, s, nil)
	}, 490, 204)
	tt.Frame()

	// Verify Redeploy is blocked
	s.DeployStatusMsg = "未触发"
	s.Redeploy(nil)
	if s.DeployStatusMsg != "未触发" {
		t.Errorf("expected Redeploy to be blocked after switching to invalid dir")
	}

	// Switch back to valid dir
	s.SetRimeDir(validDir)
	if !s.RimeDirValid {
		t.Errorf("expected switching back to valid dir to restore RimeDirValid=true")
	}
}

func TestRestoreDetectedRimeDir(t *testing.T) {
	validDir := createValidRimeDir(t)
	s := NewAppState(validDir)
	s.DetectedRimeDir = validDir
	s.DetectedRimeDirValid = true

	// Switch to invalid dir
	invalidDir := t.TempDir()
	s.SetRimeDir(invalidDir)
	if s.RimeDirValid {
		t.Fatalf("expected RimeDirValid=false for invalidDir")
	}

	// Verify UI renders "还原 ↺" button when dir is invalid
	tt := ui.NewTester(func(c *ui.Context) {
		BuildUI(c, s, nil)
	}, 490, 204)
	tt.Frame()
	if !tt.HasText("还原 ↺") {
		t.Errorf("expected UI to show '还原 ↺' button when dir is invalid")
	}

	// Restore detected dir
	restored := s.RestoreDetectedRimeDir()
	if !restored {
		t.Errorf("expected RestoreDetectedRimeDir to succeed")
	}
	if !s.RimeDirValid {
		t.Errorf("expected RimeDirValid=true after restore")
	}
	if s.RimeDir != validDir {
		t.Errorf("expected RimeDir %q, got %q", validDir, s.RimeDir)
	}

	// Verify UI hides "还原 ↺" button after restore
	ttValid := ui.NewTester(func(c *ui.Context) {
		BuildUI(c, s, nil)
	}, 490, 204)
	ttValid.Frame()
	if ttValid.HasText("还原 ↺") {
		t.Errorf("expected UI to hide '还原 ↺' button when dir is valid")
	}
}

func TestManualEditCreatesAndOpensDict(t *testing.T) {
	tempDir := createValidRimeDir(t)
	s := NewAppState(tempDir)
	var opened string
	origOpen := openPathFn
	openPathFn = func(path string) error {
		opened = path
		return nil
	}
	t.Cleanup(func() { openPathFn = origOpen })

	for _, tc := range []struct{ target, filename string }{
		{TargetCustom, "flypy.custom.dict.yaml"},
		{TargetNames, "flypy.names.dict.yaml"},
		{TargetUserTop, "flypy.user.top.dict.yaml"},
	} {
		t.Run(tc.filename, func(t *testing.T) {
			opened = ""
			want := filepath.Join(tempDir, "flypy_dicts", tc.filename)
			if err := s.OpenFile(tc.target); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(want); err != nil {
				t.Fatalf("expected dictionary to be created: %v", err)
			}
			if opened != want {
				t.Errorf("opened %q, want %q", opened, want)
			}
		})
	}
}

func TestRedeployWithMockDeployer(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Weasel deployment is only supported on Windows")
	}
	tempDir := createValidRimeDir(t)
	s := NewAppState(tempDir)
	s.DeployerPath = filepath.Join(tempDir, "mock_deployer.exe")
	// Exercise the not-installed guard even when the host has Weasel.
	s.DeployerReady = false
	s.Redeploy(nil)
	s.mu.RLock()
	msg := s.DeployStatusMsg
	s.mu.RUnlock()
	if msg != "" {
		t.Fatalf("deployment should not start when the deployer is unavailable, got %q", msg)
	}

	src := filepath.Join(tempDir, "mock_deployer.go")
	if err := os.WriteFile(src, []byte(`package main
import "os"
func main() {
	if len(os.Args) != 2 || os.Args[1] != "/deploy" {
		os.Exit(1)
	}
}
`), 0600); err != nil {
		t.Fatal(err)
	}
	goExe := filepath.Join(runtime.GOROOT(), "bin", "go.exe")
	if out, err := exec.Command(goExe, "build", "-o", s.DeployerPath, src).CombinedOutput(); err != nil {
		t.Fatalf("build mock deployer: %v\n%s", err, out)
	}

	s.DeployerReady = true // The test fixture is ready, regardless of host detection.
	s.Redeploy(nil)
	// TriggerDeploy has a 15-second timeout. Wait for an actual terminal status,
	// not for a fixed sleep or a message that may still say deployment is running.
	deadline := time.Now().Add(20 * time.Second)
	for {
		s.mu.RLock()
		msg := s.DeployStatusMsg
		s.mu.RUnlock()
		if msg == "✓ 小狼毫重新部署完成！" {
			return
		}
		if msg != "正在重新部署小狼毫..." || time.Now().After(deadline) {
			t.Fatalf("expected deployment success, got %q", msg)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestSurnameHint(t *testing.T) {
	tempDir := createValidRimeDir(t)
	s := NewAppState(tempDir)

	// "王洁" in TargetCustom should trigger SurnameHint
	s.SetWord("王洁")
	if !s.SurnameHint {
		t.Errorf("expected SurnameHint=true for 王洁 in TargetCustom")
	}

	// Switch to TargetNames -> SurnameHint should become false
	s.SetTargetDict(TargetNames)
	if s.SurnameHint {
		t.Errorf("expected SurnameHint=false in TargetNames")
	}

	// Non-name word in TargetCustom
	s.SetTargetDict(TargetCustom)
	s.SetWord("人工智能")
	if s.SurnameHint {
		t.Errorf("expected SurnameHint=false for 人工智能")
	}
}

func TestTopOverrideWorkflow(t *testing.T) {
	tempDir := createValidRimeDir(t)
	dictsDir := filepath.Join(tempDir, "flypy_dicts")
	_ = os.MkdirAll(dictsDir, 0755)

	// Setup primary word "往届 whjp"
	_ = os.WriteFile(filepath.Join(dictsDir, "flypy.primary.dict.yaml"), []byte("往届\twhjp\n"), 0644)

	s := NewAppState(tempDir)
	s.SetWord("王洁") // derives whjp

	if !s.CanTopOverride {
		t.Fatalf("expected CanTopOverride=true due to collision with 往届")
	}
	if s.TopOverrideCollisionWord != "往届" {
		t.Errorf("expected collision with 往届, got %q", s.TopOverrideCollisionWord)
	}

	// Default: not overridden -> Badge is yellow
	if s.BadgeKind != BadgeYellow || !strings.Contains(s.BadgeText, "往届") {
		t.Errorf("expected yellow collision badge, got %q (%v)", s.BadgeText, s.BadgeKind)
	}
	if s.SubmitButtonLabel() != "添加并部署" {
		t.Errorf("expected 添加并部署, got %q", s.SubmitButtonLabel())
	}

	// Toggle Top Override
	s.SetTopOverride(true)
	if s.BadgeKind != BadgeGreen || s.BadgeText != "【强制首选】" {
		t.Errorf("expected green 【强制首选】, got %q (%v)", s.BadgeText, s.BadgeKind)
	}
	if s.SubmitButtonLabel() != "置顶并部署" {
		t.Errorf("expected 置顶并部署, got %q", s.SubmitButtonLabel())
	}

	// Submit with top override
	submitted := s.Submit(nil)
	if !submitted {
		t.Fatalf("expected submit to succeed")
	}

	// Verify written to flypy.custom AND flypy.user.top
	customContent, _ := os.ReadFile(filepath.Join(dictsDir, "flypy.custom.dict.yaml"))
	if !strings.Contains(string(customContent), "王洁\twhjp\n") {
		t.Errorf("expected 王洁 in flypy.custom:\n%s", string(customContent))
	}

	topContent, _ := os.ReadFile(filepath.Join(dictsDir, "flypy.user.top.dict.yaml"))
	if !strings.Contains(string(topContent), "王洁\twhjp\n") {
		t.Errorf("expected 王洁 in flypy.user.top:\n%s", string(topContent))
	}
}

func TestCrossDictMigration(t *testing.T) {
	tempDir := createValidRimeDir(t)
	dictsDir := filepath.Join(tempDir, "flypy_dicts")
	_ = os.MkdirAll(dictsDir, 0755)

	// Pre-populate "张三丰" in flypy.custom.dict.yaml
	_ = os.WriteFile(filepath.Join(dictsDir, "flypy.custom.dict.yaml"), []byte("张三丰\tvsfg\n"), 0644)

	s := NewAppState(tempDir)
	s.SetTargetDict(TargetNames)
	s.SetWord("张三丰")

	if !s.CanMigrate {
		t.Fatalf("expected CanMigrate=true")
	}
	if s.ExistingDictSource != "flypy.custom" {
		t.Errorf("expected ExistingDictSource=flypy.custom, got %q", s.ExistingDictSource)
	}
	if s.BadgeText != "【已在自用库】" {
		t.Errorf("expected badge 【已在自用库】, got %q", s.BadgeText)
	}
	if s.SubmitButtonLabel() != "转为人名" {
		t.Errorf("expected button 转为人名, got %q", s.SubmitButtonLabel())
	}

	// Submit migration
	submitted := s.Submit(nil)
	if !submitted {
		t.Fatalf("expected migration submit to succeed")
	}

	// Verify "张三丰" was removed from custom and added to names (with 3-code and 4-code)
	customData, _ := os.ReadFile(filepath.Join(dictsDir, "flypy.custom.dict.yaml"))
	if strings.Contains(string(customData), "张三丰") {
		t.Errorf("expected 张三丰 to be removed from custom, got:\n%s", string(customData))
	}

	namesData, _ := os.ReadFile(filepath.Join(dictsDir, "flypy.names.dict.yaml"))
	if !strings.Contains(string(namesData), "张三丰\tvsf\n张三丰\tvsfg\n") {
		t.Errorf("expected 张三丰 migrated to names with dual codes, got:\n%s", string(namesData))
	}
}
