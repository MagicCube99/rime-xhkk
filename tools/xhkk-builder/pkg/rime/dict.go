package rime

import (
	"bufio"

	"errors"

	"fmt"

	"io"

	"os"

	"path/filepath"

	"strings"

	"sync"

	"time"
)

// StatusType represents the dictionary deduplication / collision status.

type StatusType int

const (

	// StatusNew indicates the word is not in the dictionary and the code has no collision.

	StatusNew StatusType = iota

	// StatusWordExists indicates the word already exists in user or system dictionaries.

	StatusWordExists

	// StatusCodeCollision indicates the word is new, but its code is already used by an existing entry.

	StatusCodeCollision
)

func (s StatusType) String() string {

	switch s {

	case StatusNew:

		return "StatusNew"

	case StatusWordExists:

		return "StatusWordExists"

	case StatusCodeCollision:

		return "StatusCodeCollision"

	default:

		return fmt.Sprintf("StatusType(%d)", s)

	}

}

// parsedDict stores parsed word-to-codes and code-to-first-candidate mappings for a single dict file.

type parsedDict struct {
	name string

	filePath string

	modTime time.Time

	size int64

	wordCodes map[string][]string // word -> list of codes

	codeWord map[string]string // code -> first candidate word
}

var (
	dictCacheMu sync.RWMutex

	dictCache = make(map[string]*parsedDict)

	ioMu sync.Mutex // Serializes write operations

)

// targetDictFiles defines the priority order of dictionaries to check.
var targetDictFiles = []string{
	"flypy.user.top.dict.yaml",
	"flypy.names.dict.yaml",
	"flypy.primary.dict.yaml",
	"flypy.custom.dict.yaml",
	"flypy.official.dict.yaml",
	"flypy.it.dict.yaml",
	"flypy.places.dict.yaml",
}

// CheckStatus checks whether a word or code already exists in Rime dictionaries.

// It checks user and modular dictionaries (flypy.user.top, flypy.custom, flypy.names,

// flypy.primary, flypy.official, flypy.it, flypy.places) if present in rimeDir.

// Return values:

//   - StatusWordExists, existingCode: word is already in a dictionary.

//   - StatusCodeCollision, firstCandidateWord: word is new, but code is already used.

//   - StatusNew, "": neither word nor code collision exists.

func CheckStatus(rimeDir string, word string, code string) (StatusType, string) {

	word = strings.TrimSpace(word)

	code = strings.TrimSpace(code)

	if word == "" {

		return StatusNew, ""

	}

	dicts := loadAvailableDicts(rimeDir)

	if len(dicts) == 0 {

		return StatusNew, ""

	}

	// 1. Check if word already exists in any dictionary (custom -> names -> primary)

	if word != "" {

		for _, d := range dicts {

			if codes, ok := d.wordCodes[word]; ok && len(codes) > 0 {

				// If user provided a specific code that matches one of the word's codes, return it.

				matchedCode := codes[0]

				for _, c := range codes {

					if c == code {

						matchedCode = c

						break

					}

				}

				return StatusWordExists, matchedCode

			}

		}

	}

	// 2. Check if code already collides with an existing word (first candidate)

	if code != "" {

		for _, d := range dicts {

			if candidate, ok := d.codeWord[code]; ok && candidate != "" {

				return StatusCodeCollision, candidate

			}

		}

	}

	// 3. Brand new word with free code

	return StatusNew, ""

}

// EnsureDictFile ensures that the target dictionary file exists under flypy_dicts in rimeDir,

// self-healing from .example if missing or creating a standard Rime header.

// Returns the absolute file path.

func EnsureDictFile(rimeDir string, targetDict string) (string, error) {

	if strings.TrimSpace(rimeDir) == "" {

		return "", errors.New("rimeDir cannot be empty")

	}

	dictName := strings.TrimSuffix(strings.TrimSpace(targetDict), ".dict.yaml")

	fileName := dictName + ".dict.yaml"

	dictsDir := filepath.Join(rimeDir, "flypy_dicts")

	filePath := filepath.Join(dictsDir, fileName)

	if err := os.MkdirAll(dictsDir, 0755); err != nil {

		return "", fmt.Errorf("failed to create dicts directory: %w", err)

	}

	info, err := os.Stat(filePath)

	if os.IsNotExist(err) {

		examplePath := filepath.Join(dictsDir, fileName+".example")

		if _, errEx := os.Stat(examplePath); errEx == nil {

			data, errRead := os.ReadFile(examplePath)

			if errRead != nil {

				return "", fmt.Errorf("failed to read example dict %s: %w", examplePath, errRead)

			}

			if errWrite := os.WriteFile(filePath, data, 0644); errWrite != nil {

				return "", fmt.Errorf("failed to initialize dict from example %s: %w", filePath, errWrite)

			}

		} else {

			header := fmt.Sprintf(`# Rime dictionary
# encoding: utf-8
---
name: %s
version: "1.0"
sort: by_weight
use_preset_vocabulary: false
...
`, dictName)
			if errWrite := os.WriteFile(filePath, []byte(header), 0644); errWrite != nil {

				return "", fmt.Errorf("failed to initialize dict header %s: %w", filePath, errWrite)

			}

		}

	} else if err != nil {

		return "", fmt.Errorf("failed to inspect dict file %s: %w", filePath, err)

	} else if info.IsDir() {

		return "", fmt.Errorf("dict path %s is a directory, not a file", filePath)

	}

	return filePath, nil

}

// AppendWord appends a word and its codes to the target dictionary (e.g. flypy.custom or flypy.names).

// If the target dictionary file does not exist:

//   - If a corresponding .example file exists, it copies it first.

//   - Otherwise, it writes a standard Rime dictionary header.

//

// Each entry is written as "%s\t%s\n". The file is cleanly flushed to disk.

func AppendWord(rimeDir string, targetDict string, word string, codes []string) error {

	rimeDir = strings.TrimSpace(rimeDir)

	if rimeDir == "" {

		return errors.New("rimeDir cannot be empty")

	}

	word = strings.TrimSpace(word)

	if word == "" {

		return errors.New("word cannot be empty")

	}

	if len(codes) == 0 {

		return errors.New("codes cannot be empty")

	}

	ioMu.Lock()

	defer ioMu.Unlock()

	filePath, err := EnsureDictFile(rimeDir, targetDict)

	if err != nil {

		return err

	}

	info, err := os.Stat(filePath)

	if err != nil {

		return fmt.Errorf("failed to inspect dict file %s: %w", filePath, err)

	}

	needLeadingNewline := false

	if info.Size() > 0 {

		f, errOpen := os.Open(filePath)

		if errOpen == nil {

			buf := make([]byte, 1)

			if _, errSeek := f.Seek(-1, io.SeekEnd); errSeek == nil {

				if _, errRead := f.Read(buf); errRead == nil && buf[0] != '\n' {

					needLeadingNewline = true

				}

			}

			_ = f.Close()

		}

	}

	// Open file in append mode

	f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)

	if err != nil {

		return fmt.Errorf("failed to open dict for append: %w", err)

	}

	defer f.Close()

	var b strings.Builder

	if needLeadingNewline {

		b.WriteString("\n")

	}

	for _, code := range codes {

		c := strings.TrimSpace(code)

		if c != "" {

			b.WriteString(fmt.Sprintf("%s\t%s\n", word, c))

		}

	}

	if _, err := f.WriteString(b.String()); err != nil {

		return fmt.Errorf("failed to write word entries: %w", err)

	}

	if err := f.Sync(); err != nil {

		return fmt.Errorf("failed to sync file to disk: %w", err)

	}

	// Invalidate memory cache for this file

	invalidateDictCache(filePath)

	return nil

}

// RemoveWord removes all entries of the given word from targetDict (e.g. flypy.custom, flypy.names, or flypy.user.top).
// It atomically updates the dictionary file and invalidates memory cache.
func RemoveWord(rimeDir string, targetDict string, word string) error {
	rimeDir = strings.TrimSpace(rimeDir)
	if rimeDir == "" {
		return errors.New("rimeDir cannot be empty")
	}
	word = strings.TrimSpace(word)
	if word == "" {
		return errors.New("word cannot be empty")
	}

	ioMu.Lock()
	defer ioMu.Unlock()

	dictName := strings.TrimSuffix(strings.TrimSpace(targetDict), ".dict.yaml")
	fileName := dictName + ".dict.yaml"
	filePath := resolveDictPath(rimeDir, fileName)
	if filePath == "" {
		return nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read dict file %s: %w", filePath, err)
	}

	lines := strings.Split(string(data), "\n")
	newLines := make([]string, 0, len(lines))
	removed := false
	inHeader := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "---" {
			inHeader = true
			newLines = append(newLines, line)
			continue
		}
		if inHeader {
			if trimmed == "..." {
				inHeader = false
			}
			newLines = append(newLines, line)
			continue
		}

		if strings.HasPrefix(trimmed, "#") || trimmed == "" {
			newLines = append(newLines, line)
			continue
		}

		parts := strings.Split(line, "\t")
		if len(parts) >= 1 && strings.TrimSpace(parts[0]) == word {
			removed = true
			continue // skip to remove
		}
		newLines = append(newLines, line)
	}

	if removed {
		for len(newLines) > 0 && strings.TrimSpace(newLines[len(newLines)-1]) == "" {
			newLines = newLines[:len(newLines)-1]
		}
		newContent := strings.Join(newLines, "\n") + "\n"
		if err := os.WriteFile(filePath, []byte(newContent), 0644); err != nil {
			return fmt.Errorf("failed to write updated dict %s: %w", filePath, err)
		}
		invalidateDictCache(filePath)
	}

	return nil
}

// FindUserDictWord reports whether and where word exists among user-modifiable dictionaries
// (flypy.custom, flypy.names, and flypy.user.top).
// Returns dictName (e.g. "flypy.custom" or "flypy.names"), existing codes, and whether it is also present in flypy.user.top.
func FindUserDictWord(rimeDir string, word string) (dictName string, codes []string, isTop bool) {
	word = strings.TrimSpace(word)
	if word == "" || rimeDir == "" {
		return "", nil, false
	}

	dicts := loadAvailableDicts(rimeDir)
	for _, d := range dicts {
		if strings.HasPrefix(d.name, "flypy.user.top") {
			if cds, ok := d.wordCodes[word]; ok && len(cds) > 0 {
				isTop = true
			}
		}
		if strings.HasPrefix(d.name, "flypy.names") {
			if cds, ok := d.wordCodes[word]; ok && len(cds) > 0 {
				dictName = "flypy.names"
				codes = cds
			}
		}
		if strings.HasPrefix(d.name, "flypy.custom") && dictName == "" {
			if cds, ok := d.wordCodes[word]; ok && len(cds) > 0 {
				dictName = "flypy.custom"
				codes = cds
			}
		}
	}
	return dictName, codes, isTop
}

// loadAvailableDicts loads and caches dictionaries in targetDictFiles priority order.

func loadAvailableDicts(rimeDir string) []*parsedDict {

	if rimeDir == "" {

		return nil

	}

	var result []*parsedDict

	for _, dictFileName := range targetDictFiles {

		filePath := resolveDictPath(rimeDir, dictFileName)

		if filePath == "" {

			continue

		}

		d := getOrLoadDict(filePath)

		if d != nil {

			result = append(result, d)

		}

	}

	return result

}

// resolveDictPath checks flypy_dicts/<name> first, then <name> directly in rimeDir.

func resolveDictPath(rimeDir, dictFileName string) string {

	p1 := filepath.Join(rimeDir, "flypy_dicts", dictFileName)

	if _, err := os.Stat(p1); err == nil {

		return p1

	}

	p2 := filepath.Join(rimeDir, dictFileName)

	if _, err := os.Stat(p2); err == nil {

		return p2

	}

	return ""

}

// getOrLoadDict retrieves the parsed dictionary from cache or parses it from disk.

func getOrLoadDict(filePath string) *parsedDict {

	info, err := os.Stat(filePath)

	if err != nil {

		return nil

	}

	dictCacheMu.RLock()

	cached, ok := dictCache[filePath]

	if ok && cached.modTime.Equal(info.ModTime()) && cached.size == info.Size() {

		dictCacheMu.RUnlock()

		return cached

	}

	dictCacheMu.RUnlock()

	// Parse file

	parsed, err := parseDictFile(filePath, info.ModTime(), info.Size())

	if err != nil {

		return nil

	}

	dictCacheMu.Lock()

	dictCache[filePath] = parsed

	dictCacheMu.Unlock()

	return parsed

}

// parseDictFile parses a Rime YAML dictionary file into word-to-codes and code-to-word maps.

func parseDictFile(filePath string, modTime time.Time, size int64) (*parsedDict, error) {

	f, err := os.Open(filePath)

	if err != nil {

		return nil, err

	}

	defer f.Close()

	wordCodes := make(map[string][]string)

	codeWord := make(map[string]string)

	scanner := bufio.NewScanner(f)

	inHeader := false

	for scanner.Scan() {

		line := scanner.Text()

		trimmed := strings.TrimSpace(line)

		if trimmed == "" {

			continue

		}

		// Rime dictionary header management

		if trimmed == "---" {

			inHeader = true

			continue

		}

		if inHeader {

			if trimmed == "..." {

				inHeader = false

			}

			continue

		}

		// Ignore comment lines

		if strings.HasPrefix(trimmed, "#") {

			continue

		}

		// Entry format: word \t code [\t weight ...]

		parts := strings.Split(line, "\t")

		if len(parts) < 2 {

			continue

		}

		w := strings.TrimSpace(parts[0])

		c := strings.TrimSpace(parts[1])

		if w == "" || c == "" {

			continue

		}

		wordCodes[w] = append(wordCodes[w], c)

		if _, exists := codeWord[c]; !exists {

			codeWord[c] = w

		}

	}

	if err := scanner.Err(); err != nil {

		return nil, err

	}

	return &parsedDict{
		name: filepath.Base(filePath),

		filePath: filePath,

		modTime: modTime,

		size: size,

		wordCodes: wordCodes,

		codeWord: codeWord,
	}, nil

}

// invalidateDictCache removes the given path from the cache.

func invalidateDictCache(filePath string) {

	dictCacheMu.Lock()

	delete(dictCache, filePath)

	dictCacheMu.Unlock()

}
