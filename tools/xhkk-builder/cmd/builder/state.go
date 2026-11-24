package main

import (
	"errors"
	"fmt"

	"os/exec"

	"regexp"

	"runtime"

	"strings"

	"sync"

	"github.com/egoist/mygo"

	"github.com/egoist/mygo/ui"

	"rime-xhkk-builder/pkg/encoder"

	"rime-xhkk-builder/pkg/rime"
)

// BadgeKind defines the visual classification of the status badge.

type BadgeKind int

const (
	BadgeGray BadgeKind = iota

	BadgeGreen

	BadgeYellow

	BadgeRed
)

// CharState represents the active pronunciation and all alternates for a single character.

type CharState struct {
	Char rune

	Flypy string

	Initial string

	Final string

	Pinyin string

	Candidates []string

	Readings []encoder.CharPinyin

	ActiveIndex int
}

// BatchItem holds derivation and check status for a single word in a multi-word input.

type BatchItem struct {
	Word string

	Codes []string

	DisplayCode string

	Status rime.StatusType

	StatusMessage string

	BadgeKind BadgeKind
}

// Target dictionary constants
const (
	TargetCustom  = "flypy.custom"
	TargetNames   = "flypy.names"
	TargetUserTop = "flypy.user.top"
)

// hanRegexp validates that the input consists of at least two Chinese Han characters.

var hanRegexp = regexp.MustCompile(`^[\p{Han}]{2,}$`)

// openPathFn is the launcher function for opening files with default editor, customizable for testing.

var openPathFn = func(path string) error {

	if err := mygo.Shell.OpenPath(path); err == nil {

		return nil

	}

	if runtime.GOOS == "windows" {

		return exec.Command("notepad.exe", path).Start()

	}

	return nil

}

// AppState manages the application state reactively.

type AppState struct {
	mu sync.RWMutex

	// Input fields

	Word string

	lastProcessedWord string

	TargetDict string // TargetCustom or TargetNames

	lastTargetDict string

	// Derived encoding (single-word mode)

	CharStates []CharState

	PrimaryCode string

	SecondaryCode string

	DisplayCode string

	// Multi-word batch mode

	IsBatch bool

	BatchItems []BatchItem

	// 3-character Person name smart fallback & user override

	HasShortCode bool

	IncludeShortCode bool

	includeShortCodeInitialized bool

	lastIncludeShortCode bool

	ShortCode string

	FullCode string

	HasShortCodeCollision bool

	ShortCodeCollisionWord string

	FullCodeCollisionWord string

	// Top override (shadow write into flypy.user.top)
	IsTopOverride            bool
	CanTopOverride           bool
	TopOverrideCollisionWord string

	// Surname heuristic hint
	SurnameHint bool

	// Cross-dictionary detection & migration
	ExistingDictSource string // "flypy.custom" or "flypy.names"
	CanMigrate         bool

	// Rime directory health validation

	RimeDirValid bool

	RimeDirError string

	// Status and validation

	Status rime.StatusType

	StatusMessage string

	BadgeText string

	BadgeKind BadgeKind

	CanSubmit bool

	// System integration

	RimeDir string

	DetectedRimeDir string

	DetectedRimeDirValid bool

	DeployerPath string

	DeployerReady bool

	ServerPath string

	ServerReady bool

	ServerRunning bool

	DeployStatusMsg string

	// Async / UI control

	Submitting bool

	InputHandle ui.Handle
}

// NewAppState initializes an AppState with auto-detected Rime dir and Weasel deployer.

func NewAppState(rimeDir string) *AppState {

	detectedDir := rime.DetectRimeDir()

	detectedValid, _ := rime.ValidateRimeDir(detectedDir)

	if strings.TrimSpace(rimeDir) == "" {

		rimeDir = detectedDir

	}

	deployer, ready := rime.DetectWeaselDeployer()

	serverPath, serverReady, serverRunning := rime.DetectWeaselServer()

	valid, rErr := rime.ValidateRimeDir(rimeDir)

	s := &AppState{

		TargetDict: TargetCustom,

		lastTargetDict: TargetCustom,

		RimeDir: rimeDir,

		RimeDirValid: valid,

		RimeDirError: rErr,

		DetectedRimeDir: detectedDir,

		DetectedRimeDirValid: detectedValid,

		DeployerPath: deployer,

		DeployerReady: ready,

		ServerPath: serverPath,

		ServerReady: serverReady,

		ServerRunning: serverRunning,
	}

	s.resetWordStateLocked()

	return s

}

// clearTransientStatusLocked clears success/switch/restore feedback on subsequent user interactions.

func (s *AppState) clearTransientStatusLocked() {

	if strings.Contains(s.DeployStatusMsg, "已还原") ||

		strings.Contains(s.DeployStatusMsg, "已切换") ||

		strings.Contains(s.DeployStatusMsg, "完成") {

		s.DeployStatusMsg = ""

	}

}

// SetWord updates the input word and reprocesses phonetic and dictionary status.

func (s *AppState) SetWord(word string) {

	s.mu.Lock()

	defer s.mu.Unlock()

	s.clearTransientStatusLocked()

	s.Word = word

	s.processWordLocked()

	s.lastProcessedWord = word

}

// SetTargetDict updates the target dictionary (flypy.custom vs flypy.names) and recalculates codes.

func (s *AppState) SetTargetDict(target string) {

	s.mu.Lock()

	defer s.mu.Unlock()

	s.clearTransientStatusLocked()

	if target != TargetCustom && target != TargetNames {

		target = TargetCustom

	}

	s.TargetDict = target

	s.includeShortCodeInitialized = false

	s.updateCodesAndStatusLocked()

	s.lastTargetDict = target

}

// SetRimeDir updates the target Rime directory and re-checks status against its dictionaries.

func (s *AppState) SetRimeDir(dir string) {

	s.mu.Lock()

	defer s.mu.Unlock()

	dir = strings.TrimSpace(dir)

	valid, rErr := rime.ValidateRimeDir(dir)

	s.RimeDir = dir

	s.RimeDirValid = valid

	s.RimeDirError = rErr

	s.updateCodesAndStatusLocked()

}

// RestoreDetectedRimeDir restores the user directory back to the automatically detected default directory.

func (s *AppState) RestoreDetectedRimeDir() bool {

	s.mu.RLock()

	detected := s.DetectedRimeDir

	valid := s.DetectedRimeDirValid

	s.mu.RUnlock()

	if !valid || strings.TrimSpace(detected) == "" {

		return false

	}

	s.SetRimeDir(detected)

	s.mu.Lock()

	s.DeployStatusMsg = "已还原至默认用户文件夹"

	s.mu.Unlock()

	return true

}

// SetTopOverride sets whether this entry will be shadow-written into flypy.user.top to override first choice.
func (s *AppState) SetTopOverride(override bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.IsTopOverride = override
	s.updateCodesAndStatusLocked()
}

// SetIncludeShortCode allows the user to explicitly include or exclude the 3-code short code for 3-char names.

func (s *AppState) SetIncludeShortCode(include bool) {

	s.mu.Lock()

	defer s.mu.Unlock()

	s.IncludeShortCode = include

	s.lastIncludeShortCode = include

	s.updateCodesAndStatusLocked()

}

// CycleReading advances the active pronunciation of the character at charIndex to the next alternate.

func (s *AppState) CycleReading(charIndex int) {

	s.mu.Lock()

	defer s.mu.Unlock()

	if charIndex < 0 || charIndex >= len(s.CharStates) {

		return

	}

	cs := &s.CharStates[charIndex]

	if len(cs.Readings) <= 1 {

		return

	}

	cs.ActiveIndex = (cs.ActiveIndex + 1) % len(cs.Readings)

	selected := cs.Readings[cs.ActiveIndex]

	cs.Flypy = selected.Flypy

	cs.Initial = selected.Initial

	cs.Final = selected.Final

	cs.Pinyin = selected.Pinyin

	s.updateCodesAndStatusLocked()

}

// Sync synchronizes any state changes made by UI bindings (e.g. TextInput or Radio).

func (s *AppState) Sync() {

	s.mu.Lock()

	defer s.mu.Unlock()

	if s.Word != s.lastProcessedWord {

		s.processWordLocked()

		s.lastProcessedWord = s.Word

	}

	if s.TargetDict != s.lastTargetDict {

		s.includeShortCodeInitialized = false

		s.updateCodesAndStatusLocked()

		s.lastTargetDict = s.TargetDict

	}

	if s.IncludeShortCode != s.lastIncludeShortCode {

		s.lastIncludeShortCode = s.IncludeShortCode

		s.updateCodesAndStatusLocked()

	}

}

// processWordLocked handles validation, disambiguation, and code generation.

func (s *AppState) processWordLocked() {

	raw := strings.TrimSpace(s.Word)

	if raw == "" {

		s.resetWordStateLocked()

		return

	}

	words := strings.Fields(raw)

	if len(words) == 0 {

		s.resetWordStateLocked()

		return

	}

	if len(words) == 1 {

		s.IsBatch = false

		s.BatchItems = nil

		trimmed := words[0]

		if !hanRegexp.MatchString(trimmed) {

			s.resetWordStateLocked()

			s.BadgeText = "【≥2字中文】"

			s.BadgeKind = BadgeGray

			s.CanSubmit = false

			return

		}

		chars, err := encoder.Disambiguate(trimmed)

		if err != nil {

			s.resetWordStateLocked()

			return

		}

		s.CharStates = make([]CharState, len(chars))

		for i, cp := range chars {

			readings := []encoder.CharPinyin{cp}

			readings = append(readings, cp.Alternates...)

			s.CharStates[i] = CharState{

				Char: cp.Char,

				Flypy: cp.Flypy,

				Initial: cp.Initial,

				Final: cp.Final,

				Pinyin: cp.Pinyin,

				Candidates: cp.Candidates,

				Readings: readings,

				ActiveIndex: 0,
			}

		}

		s.updateCodesAndStatusLocked()

		return

	}

	// Multi-word batch mode

	s.IsBatch = true

	s.CharStates = nil

	s.BatchItems = make([]BatchItem, 0, len(words))

	var invalidWord string

	for _, w := range words {

		if !hanRegexp.MatchString(w) {

			invalidWord = w

			break

		}

	}

	if invalidWord != "" {

		s.PrimaryCode = ""

		s.SecondaryCode = ""

		s.DisplayCode = ""

		s.Status = rime.StatusNew

		s.StatusMessage = fmt.Sprintf("'%s' 格式错误", invalidWord)

		s.BadgeText = "【含非法词】"

		s.BadgeKind = BadgeGray

		s.CanSubmit = false

		return

	}

	var codeSummaries []string

	addableCount := 0

	existingCount := 0

	for _, w := range words {

		chars, err := encoder.Disambiguate(w)

		if err != nil {

			continue

		}

		pCode, sCode, err := encoder.DeriveCodes(chars, s.TargetDict)

		if err != nil {

			continue

		}

		var codes []string

		if sCode != "" {

			codes = []string{pCode, sCode}

		} else {

			codes = []string{pCode}

		}

		status, extra := rime.CheckStatus(s.RimeDir, w, pCode)

		bKind := BadgeGreen

		var sMsg string

		switch status {

		case rime.StatusWordExists:

			bKind = BadgeRed

			existingCount++

			sMsg = "已存在"

		case rime.StatusCodeCollision:

			bKind = BadgeYellow

			addableCount++

			sMsg = fmt.Sprintf("重码: %s", extra)

		default:

			addableCount++

			sMsg = "未收录"

		}

		disp := pCode

		if sCode != "" {

			disp = pCode + "/" + sCode

		}

		codeSummaries = append(codeSummaries, fmt.Sprintf("%s(%s)", w, pCode))

		s.BatchItems = append(s.BatchItems, BatchItem{

			Word: w,

			Codes: codes,

			DisplayCode: disp,

			Status: status,

			StatusMessage: sMsg,

			BadgeKind: bKind,
		})

	}

	s.PrimaryCode = strings.Join(codeSummaries, " ")

	s.SecondaryCode = ""

	s.DisplayCode = s.PrimaryCode

	if addableCount == 0 {

		s.BadgeKind = BadgeRed

		s.BadgeText = "【全部已存在】"

		s.CanSubmit = false

	} else if existingCount > 0 {

		s.BadgeKind = BadgeYellow

		s.BadgeText = fmt.Sprintf("【%d/%d词待加】", addableCount, len(words))

		s.CanSubmit = true

	} else {

		s.BadgeKind = BadgeGreen

		s.BadgeText = fmt.Sprintf("【%d词可添加】", addableCount)

		s.CanSubmit = true

	}

	if !s.RimeDirValid {

		s.CanSubmit = false

		s.BadgeKind = BadgeGray

		s.BadgeText = "【未核查】"

		s.StatusMessage = "未连接有效词库，尚未进行去重与重码核验"

	}

}

// updateCodesAndStatusLocked recomputes flypy codes and queries Rime dictionary status.

func (s *AppState) updateCodesAndStatusLocked() {

	if s.IsBatch {

		s.processWordLocked()

		return

	}

	activeChars := s.getActiveCharPinyinsLocked()

	trimmed := strings.TrimSpace(s.Word)

	if len(activeChars) < 2 || !hanRegexp.MatchString(trimmed) {
		s.resetWordStateLocked()
		return
	}

	// Heuristic surname hint
	s.SurnameHint = encoder.IsLikelyPersonName(trimmed) && (s.TargetDict == TargetCustom)

	// Reset migration and top-override state before re-checking
	s.CanMigrate = false
	s.ExistingDictSource = ""
	s.CanTopOverride = false
	s.TopOverrideCollisionWord = ""

	// Check cross-dictionary existence for migration
	existDict, _, _ := rime.FindUserDictWord(s.RimeDir, trimmed)
	if existDict != "" {
		if s.TargetDict == TargetNames && existDict == "flypy.custom" {
			s.CanMigrate = true
			s.ExistingDictSource = "flypy.custom"
			s.BadgeKind = BadgeYellow
			s.BadgeText = "【已在自用库】"
			s.StatusMessage = "该词已在自用词库，提交可一键转为人名词库（补全3码）"
			s.CanSubmit = true
		} else if s.TargetDict == TargetCustom && existDict == "flypy.names" {
			s.CanMigrate = true
			s.ExistingDictSource = "flypy.names"
			s.BadgeKind = BadgeYellow
			s.BadgeText = "【已在人名库】"
			s.StatusMessage = "该词已在人名词库，提交可一键转为自用词库"
			s.CanSubmit = true
		}
	}

	// Check if this is a 3-character person name in flypy.names

	if s.TargetDict == TargetNames && len(activeChars) == 3 {

		s.HasShortCode = true

		c1, c2, c3 := activeChars[0].Flypy, activeChars[1].Flypy, activeChars[2].Flypy

		shortCode := string([]rune{rune(c1[0]), rune(c2[0]), rune(c3[0])})

		fullCode := string([]rune{rune(c1[0]), rune(c2[0])}) + c3

		s.ShortCode = shortCode

		s.FullCode = fullCode

		st3, ex3 := rime.CheckStatus(s.RimeDir, trimmed, shortCode)

		st4, ex4 := rime.CheckStatus(s.RimeDir, trimmed, fullCode)

		if !s.CanMigrate && (st3 == rime.StatusWordExists || st4 == rime.StatusWordExists) {
			s.BadgeKind = BadgeRed
			s.BadgeText = "【已存在】"
			s.StatusMessage = "词条已存在"
			s.CanSubmit = false
			s.PrimaryCode = fullCode
			s.DisplayCode = fullCode
			return
		}

		s.HasShortCodeCollision = (st3 == rime.StatusCodeCollision)
		s.ShortCodeCollisionWord = ex3

		if st4 == rime.StatusCodeCollision {
			s.FullCodeCollisionWord = ex4
			s.CanTopOverride = true
			s.TopOverrideCollisionWord = ex4
		} else {
			s.FullCodeCollisionWord = ""
		}

		// When word or target changed, decide default for IncludeShortCode:
		if !s.includeShortCodeInitialized {
			if s.HasShortCodeCollision {
				s.IncludeShortCode = false
			} else {
				s.IncludeShortCode = true
			}
			s.includeShortCodeInitialized = true
			s.lastIncludeShortCode = s.IncludeShortCode
		}

		if s.IncludeShortCode {
			s.PrimaryCode = shortCode
			s.SecondaryCode = fullCode
			s.DisplayCode = shortCode + "  " + fullCode

			if !s.CanMigrate {
				if s.HasShortCodeCollision {
					s.BadgeKind = BadgeYellow
					s.BadgeText = fmt.Sprintf("【3码顶替‘%s’】", s.ShortCodeCollisionWord)
					s.StatusMessage = fmt.Sprintf("3码将顶替 '%s'", s.ShortCodeCollisionWord)
				} else if s.FullCodeCollisionWord != "" {
					s.BadgeKind = BadgeYellow
					s.BadgeText = fmt.Sprintf("【重码：%s】", s.FullCodeCollisionWord)
					s.StatusMessage = fmt.Sprintf("4码重码，当前首选为 '%s'", s.FullCodeCollisionWord)
				} else {
					s.BadgeKind = BadgeGreen
					s.BadgeText = "【未收录】"
					s.StatusMessage = "未收录"
				}
			}
			s.CanSubmit = true
		} else {
			s.PrimaryCode = fullCode
			s.SecondaryCode = ""
			s.DisplayCode = fullCode

			if !s.CanMigrate {
				if s.FullCodeCollisionWord != "" {
					s.BadgeKind = BadgeYellow
					s.BadgeText = fmt.Sprintf("【重码：%s】", s.FullCodeCollisionWord)
					s.StatusMessage = fmt.Sprintf("当前首选为 '%s'，提交将作为次选", s.FullCodeCollisionWord)
				} else {
					s.BadgeKind = BadgeGreen
					s.BadgeText = "【未收录】"
					s.StatusMessage = "未收录"
				}
			}
			s.CanSubmit = true
		}

		if s.CanTopOverride && s.IsTopOverride {
			s.BadgeKind = BadgeGreen
			s.BadgeText = "【强制首选】"
			s.StatusMessage = fmt.Sprintf("将写入 user.top 夺得首选，'%s' 将降为次选", s.TopOverrideCollisionWord)
			s.CanSubmit = true
		}

		if !s.RimeDirValid {

			s.CanSubmit = false

			s.BadgeKind = BadgeGray

			s.BadgeText = "【未核查】"

			s.StatusMessage = "未连接有效词库，尚未进行去重与重码核验"

			s.HasShortCodeCollision = false

			s.ShortCodeCollisionWord = ""

			s.FullCodeCollisionWord = ""

		}

		return

	}

	// Regular word (2-char, 4+-char, or custom dict)

	s.HasShortCode = false

	s.IncludeShortCode = false

	s.ShortCode = ""

	s.FullCode = ""

	s.HasShortCodeCollision = false

	s.ShortCodeCollisionWord = ""

	s.FullCodeCollisionWord = ""

	primaryCode, secondaryCode, err := encoder.DeriveCodes(activeChars, s.TargetDict)

	if err != nil {

		s.resetWordStateLocked()

		return

	}

	s.PrimaryCode = primaryCode

	s.SecondaryCode = secondaryCode

	if secondaryCode != "" {

		s.DisplayCode = primaryCode + "  " + secondaryCode

	} else {

		s.DisplayCode = primaryCode

	}

	status, extra := rime.CheckStatus(s.RimeDir, trimmed, s.PrimaryCode)

	s.Status = status

	if !s.CanMigrate {
		switch status {
		case rime.StatusWordExists:
			s.BadgeKind = BadgeRed
			s.BadgeText = "【已存在】"
			if extra != "" {
				s.StatusMessage = fmt.Sprintf("已存在，编码为 '%s'", extra)
			} else {
				s.StatusMessage = "词条已存在"
			}
			s.CanSubmit = false

		case rime.StatusCodeCollision:
			s.CanTopOverride = true
			s.TopOverrideCollisionWord = extra
			if s.IsTopOverride {
				s.BadgeKind = BadgeGreen
				s.BadgeText = "【强制首选】"
				s.StatusMessage = fmt.Sprintf("将写入 user.top 夺得首选，"+"%q"+" 将降为次选", s.TopOverrideCollisionWord)
				s.CanSubmit = true
			} else {
				s.BadgeKind = BadgeYellow
				if extra != "" {
					s.BadgeText = fmt.Sprintf("【重码：%s】", extra)
					s.StatusMessage = fmt.Sprintf("当前首选为 '%s'，提交将作为次选", extra)
				} else {
					s.BadgeText = "[重码]"
					s.StatusMessage = "存在重码"
				}
				s.CanSubmit = true
			}

		default: // rime.StatusNew
			s.BadgeKind = BadgeGreen
			s.BadgeText = "【未收录】"
			s.StatusMessage = "未收录"
			s.CanSubmit = true
		}
	}

	if !s.RimeDirValid {

		s.CanSubmit = false

		s.BadgeKind = BadgeGray

		s.BadgeText = "【未核查】"

		s.StatusMessage = "未连接有效词库，尚未进行去重与重码核验"

	}

}

// resetWordStateLocked clears derived representations when the input is invalid or empty.

func (s *AppState) resetWordStateLocked() {

	s.IsBatch = false

	s.BatchItems = nil

	s.CharStates = nil

	s.PrimaryCode = ""

	s.SecondaryCode = ""

	s.DisplayCode = ""

	s.HasShortCode = false

	s.IncludeShortCode = false

	s.includeShortCodeInitialized = false

	s.lastIncludeShortCode = false

	s.ShortCode = ""

	s.FullCode = ""

	s.HasShortCodeCollision = false

	s.ShortCodeCollisionWord = ""

	s.FullCodeCollisionWord = ""
	s.IsTopOverride = false
	s.CanTopOverride = false
	s.TopOverrideCollisionWord = ""
	s.SurnameHint = false
	s.ExistingDictSource = ""
	s.CanMigrate = false

	s.Status = rime.StatusNew

	s.StatusMessage = "仅支持2字及以上中文"

	s.BadgeText = "【≥2字中文】"

	s.BadgeKind = BadgeGray

	s.CanSubmit = false

}

// SubmitButtonLabel returns the concise adaptive text for the primary action button.

func (s *AppState) SubmitButtonLabel() string {

	s.mu.RLock()

	defer s.mu.RUnlock()

	trimmed := strings.TrimSpace(s.Word)

	if trimmed == "" {
		return "重新部署"
	}

	if s.CanMigrate {
		if s.TargetDict == TargetNames {
			return "转为人名"
		}
		return "转为自用"
	}

	if s.IsTopOverride {
		return "置顶并部署"
	}

	if s.IsBatch {

		addable := 0

		for _, it := range s.BatchItems {

			if it.Status != rime.StatusWordExists {

				addable++

			}

		}

		if addable > 0 {

			return fmt.Sprintf("批量添加 (%d)", addable)

		}

		return "词条已存在"

	}

	if s.Status == rime.StatusWordExists {

		return "词条已存在"

	}

	return "添加并部署"

}

// WeaselServiceStatusText returns the dual status description for both deployment and algorithm services.

func (s *AppState) WeaselServiceStatusText() (string, bool) {

	s.mu.RLock()

	defer s.mu.RUnlock()

	deployPart := "部署服务已就绪"

	if !s.DeployerReady {

		deployPart = "未检测到部署服务"

	}

	serverPart := "算法服务已就绪"

	if !s.ServerRunning {

		if s.ServerReady {

			serverPart = "算法服务未运行"

		} else {

			serverPart = "未检测到算法服务"

		}

	}

	text := fmt.Sprintf("%s  %s", deployPart, serverPart)

	allGood := s.DeployerReady && s.ServerRunning

	return text, allGood

}

// getActiveCharPinyinsLocked converts current CharStates to a slice of encoder.CharPinyin.

func (s *AppState) getActiveCharPinyinsLocked() []encoder.CharPinyin {

	res := make([]encoder.CharPinyin, len(s.CharStates))

	for i, cs := range s.CharStates {

		res[i] = encoder.CharPinyin{

			Char: cs.Char,

			Initial: cs.Initial,

			Final: cs.Final,

			Flypy: cs.Flypy,

			Pinyin: cs.Pinyin,

			Candidates: cs.Candidates,
		}

	}

	return res

}

// Submit appends the word(s) and their derived codes to the dictionary, triggers silent deploy,

// and resets the input box for the next word. If input is empty, it acts as a manual redeploy.

func (s *AppState) Submit(win *mygo.Window) bool {

	s.mu.Lock()

	trimmed := strings.TrimSpace(s.Word)

	if trimmed == "" {

		s.mu.Unlock()

		// Empty input: trigger redeploy directly

		s.Redeploy(win)

		return true

	}

	if !s.CanSubmit || s.Submitting {

		s.mu.Unlock()

		return false

	}

	s.Submitting = true

	targetDict := s.TargetDict
	rimeDir := s.RimeDir
	deployerPath := s.DeployerPath
	canMigrate := s.CanMigrate
	migrateSource := s.ExistingDictSource
	isTopOverride := s.IsTopOverride

	type wordEntry struct {
		word string

		codes []string
	}

	var entries []wordEntry

	if s.IsBatch {

		for _, item := range s.BatchItems {

			if item.Status != rime.StatusWordExists {

				entries = append(entries, wordEntry{word: item.Word, codes: item.Codes})

			}

		}

	} else {

		if trimmed != "" {

			var codes []string

			if s.HasShortCode {

				if s.IncludeShortCode {

					codes = []string{s.ShortCode, s.FullCode}

				} else {

					codes = []string{s.FullCode}

				}

			} else {

				if s.SecondaryCode != "" {

					codes = []string{s.PrimaryCode, s.SecondaryCode}

				} else {

					codes = []string{s.PrimaryCode}

				}

			}

			entries = append(entries, wordEntry{word: trimmed, codes: codes})

		}

	}

	if len(entries) == 0 {

		s.Submitting = false

		s.mu.Unlock()

		return false

	}

	s.mu.Unlock()

	// Append all words to dictionary

	addedCount := 0

	for _, entry := range entries {
		if canMigrate && migrateSource != "" {
			_ = rime.RemoveWord(rimeDir, migrateSource, entry.word)
		}
		if err := rime.AppendWord(rimeDir, targetDict, entry.word, entry.codes); err == nil {
			addedCount++
			if isTopOverride {
				topCode := entry.codes[len(entry.codes)-1]
				_ = rime.AppendWord(rimeDir, "flypy.user.top", entry.word, []string{topCode})
			}
		}
	}

	// Success: clear input and reset word state

	s.mu.Lock()

	s.Word = ""

	s.lastProcessedWord = ""

	s.resetWordStateLocked()

	s.DeployStatusMsg = fmt.Sprintf("已写入 %d 词，正在部署...", addedCount)

	s.Submitting = false

	s.mu.Unlock()

	// Trigger deploy in background goroutine

	go func() {

		deployErr := rime.TriggerDeploy(deployerPath)

		updateMsg := func() {

			s.mu.Lock()

			defer s.mu.Unlock()

			if deployErr != nil {

				s.DeployStatusMsg = fmt.Sprintf("已写入 %d 词 (部署提醒: %v)", addedCount, deployErr)
			} else {
				if canMigrate {
					dictLabel := "人名词库"
					if targetDict == TargetCustom {
						dictLabel = "自用词库"
					}
					s.DeployStatusMsg = fmt.Sprintf("✓ 已成功转入%s并完成部署！", dictLabel)
				} else if isTopOverride {
					s.DeployStatusMsg = "✓ 已写入并强设首选，部署完成！"
				} else {
					s.DeployStatusMsg = fmt.Sprintf("✓ 已添加 %d 词并重新部署完成！", addedCount)
				}

			}

		}

		if win != nil {

			win.Update(updateMsg)

		} else {

			updateMsg()

		}

	}()

	return true

}

// OpenFile opens the target dictionary file (TargetCustom or TargetNames) in the system default text editor.

// If the file does not exist yet, it self-heals from .example before opening.

func (s *AppState) OpenFile(targetDict string) error {

	s.mu.Lock()

	if !s.RimeDirValid {

		s.DeployStatusMsg = "目录无效，无法打开词库文件"

		s.mu.Unlock()

		return errors.New("用户文件夹无效")

	}

	rimeDir := s.RimeDir

	s.mu.Unlock()

	filePath, err := rime.EnsureDictFile(rimeDir, targetDict)

	if err != nil {

		s.mu.Lock()

		s.DeployStatusMsg = fmt.Sprintf("打开失败: %v", err)

		s.mu.Unlock()

		return err

	}

	// Launch editor

	openErr := openPathFn(filePath)

	if openErr != nil {

		s.mu.Lock()

		s.DeployStatusMsg = fmt.Sprintf("打开失败: %v", openErr)

		s.mu.Unlock()

	}

	return openErr

}

// Redeploy manually triggers WeaselDeployer execution without adding a new word.

func (s *AppState) Redeploy(win *mygo.Window) {

	s.mu.Lock()

	if s.Submitting || !s.RimeDirValid || !s.DeployerReady {

		s.mu.Unlock()

		return

	}

	deployerPath := s.DeployerPath

	s.DeployStatusMsg = "正在重新部署小狼毫..."

	s.mu.Unlock()

	go func() {

		err := rime.TriggerDeploy(deployerPath)

		updateUI := func() {

			s.mu.Lock()

			defer s.mu.Unlock()

			if err != nil {

				s.DeployStatusMsg = fmt.Sprintf("部署提醒: %v", err)

			} else {

				s.DeployStatusMsg = "✓ 小狼毫重新部署完成！"

			}

		}

		if win != nil {

			win.Update(updateUI)

		} else {

			updateUI()

		}

	}()

}
