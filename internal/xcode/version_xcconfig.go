package xcode

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var (
	xcconfigAssignmentPattern = regexp.MustCompile(`^(\s*)([A-Za-z_][A-Za-z0-9_]*(?:\[[^\]\r\n]+\])*)(\s*(?:\+=|\?=|=)\s*)(.*?)([ \t]*)$`)
	xcconfigIncludePattern    = regexp.MustCompile(`^\s*#include(\?)?\s+"([^"]+)"\s*$`)
)

type xcconfigAssignment struct {
	lineIndex  int
	key        string
	baseKey    string
	value      string
	valueStart int
	valueEnd   int
}

type xcconfigInclude struct {
	lineIndex int
	path      string
	optional  bool
}

type xcconfigDocument struct {
	lines       []string
	assignments []xcconfigAssignment
	includes    []xcconfigInclude
}

type xcconfigResolvedValue struct {
	value string
	path  string
	found bool
	exact bool
}

func parseXCConfig(data []byte) (xcconfigDocument, error) {
	lines := splitLinesPreservingEndings(string(data))
	document := xcconfigDocument{lines: lines}
	inBlockComment := false

	for index, line := range lines {
		body := strings.TrimSuffix(line, "\n")
		body = strings.TrimSuffix(body, "\r")
		masked, nextInBlock := maskXCConfigComments(body, inBlockComment)
		inBlockComment = nextInBlock

		if matches := xcconfigIncludePattern.FindStringSubmatch(masked); matches != nil {
			document.includes = append(document.includes, xcconfigInclude{
				lineIndex: index,
				path:      matches[2],
				optional:  matches[1] == "?",
			})
			continue
		}

		indices := xcconfigAssignmentPattern.FindStringSubmatchIndex(masked)
		if indices == nil {
			continue
		}
		key := masked[indices[4]:indices[5]]
		valueStart, valueEnd := indices[8], indices[9]
		document.assignments = append(document.assignments, xcconfigAssignment{
			lineIndex:  index,
			key:        key,
			baseKey:    xcconfigBaseKey(key),
			value:      strings.TrimSpace(body[valueStart:valueEnd]),
			valueStart: valueStart,
			valueEnd:   valueEnd,
		})
	}

	if inBlockComment {
		return xcconfigDocument{}, fmt.Errorf("unterminated block comment in xcconfig")
	}
	return document, nil
}

func splitLinesPreservingEndings(value string) []string {
	if value == "" {
		return []string{""}
	}
	lines := strings.SplitAfter(value, "\n")
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

func maskXCConfigComments(line string, inBlockComment bool) (string, bool) {
	masked := []byte(line)
	inQuote := byte(0)
	escaped := false

	for index := 0; index < len(masked); index++ {
		if inBlockComment {
			masked[index] = ' '
			if index+1 < len(masked) && line[index] == '*' && line[index+1] == '/' {
				masked[index+1] = ' '
				index++
				inBlockComment = false
			}
			continue
		}

		character := line[index]
		if inQuote != 0 {
			if escaped {
				escaped = false
				continue
			}
			if character == '\\' {
				escaped = true
				continue
			}
			if character == inQuote {
				inQuote = 0
			}
			continue
		}

		if character == '"' || character == '\'' {
			inQuote = character
			continue
		}
		if index+1 >= len(masked) {
			continue
		}
		if line[index:index+2] == "//" {
			for rest := index; rest < len(masked); rest++ {
				masked[rest] = ' '
			}
			break
		}
		if line[index:index+2] == "/*" {
			masked[index] = ' '
			masked[index+1] = ' '
			index++
			inBlockComment = true
		}
	}
	return string(masked), inBlockComment
}

func xcconfigBaseKey(key string) string {
	if index := strings.Index(key, "["); index >= 0 {
		return key[:index]
	}
	return key
}

func resolveXCConfigInclude(containingPath string, include xcconfigInclude) (string, error) {
	if strings.Contains(include.path, "$(") || strings.Contains(include.path, "${") {
		return "", fmt.Errorf("xcconfig include contains unresolved build setting: %s", include.path)
	}
	path := include.path
	if filepath.Ext(path) == "" {
		path += ".xcconfig"
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(filepath.Dir(containingPath), path)
	}
	return filepath.Clean(path), nil
}

func collectXCConfigFiles(root string) ([]string, error) {
	seen := make(map[string]bool)
	var paths []string
	var visit func(string, map[string]bool) error
	visit = func(path string, stack map[string]bool) error {
		path = filepath.Clean(path)
		if stack[path] || seen[path] {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		document, err := parseXCConfig(data)
		if err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
		seen[path] = true
		paths = append(paths, path)
		nextStack := clonePathSet(stack)
		nextStack[path] = true
		for _, include := range document.includes {
			includePath, err := resolveXCConfigInclude(path, include)
			if err != nil {
				return err
			}
			if _, err := os.Stat(includePath); err != nil {
				if include.optional && os.IsNotExist(err) {
					continue
				}
				return fmt.Errorf("read xcconfig include %s: %w", includePath, err)
			}
			if err := visit(includePath, nextStack); err != nil {
				return err
			}
		}
		return nil
	}
	if err := visit(root, make(map[string]bool)); err != nil {
		return nil, err
	}
	return paths, nil
}

func resolveXCConfigSetting(root, setting string) (xcconfigResolvedValue, error) {
	return resolveXCConfigSettingRecursive(filepath.Clean(root), setting, make(map[string]bool))
}

func resolveXCConfigSettingRecursive(path, setting string, stack map[string]bool) (xcconfigResolvedValue, error) {
	if stack[path] {
		return xcconfigResolvedValue{}, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return xcconfigResolvedValue{}, err
	}
	document, err := parseXCConfig(data)
	if err != nil {
		return xcconfigResolvedValue{}, fmt.Errorf("parse %s: %w", path, err)
	}
	nextStack := clonePathSet(stack)
	nextStack[path] = true

	type event struct {
		line       int
		assignment *xcconfigAssignment
		include    *xcconfigInclude
	}
	var events []event
	for index := range document.assignments {
		assignment := &document.assignments[index]
		events = append(events, event{line: assignment.lineIndex, assignment: assignment})
	}
	for index := range document.includes {
		include := &document.includes[index]
		events = append(events, event{line: include.lineIndex, include: include})
	}
	sort.SliceStable(events, func(left, right int) bool {
		return events[left].line < events[right].line
	})

	var resolved xcconfigResolvedValue
	for _, item := range events {
		if item.include != nil {
			includePath, err := resolveXCConfigInclude(path, *item.include)
			if err != nil {
				return xcconfigResolvedValue{}, err
			}
			if _, err := os.Stat(includePath); err != nil {
				if item.include.optional && os.IsNotExist(err) {
					continue
				}
				return xcconfigResolvedValue{}, fmt.Errorf("read xcconfig include %s: %w", includePath, err)
			}
			included, err := resolveXCConfigSettingRecursive(includePath, setting, nextStack)
			if err != nil {
				return xcconfigResolvedValue{}, err
			}
			if included.found && (!resolved.exact || included.exact) {
				resolved = included
			}
			continue
		}

		assignment := item.assignment
		if assignment.baseKey != setting {
			continue
		}
		exact := assignment.key == setting
		if !exact && resolved.exact {
			continue
		}
		value := assignment.value
		if strings.Contains(value, "$(inherited)") {
			value = strings.ReplaceAll(value, "$(inherited)", resolved.value)
		}
		resolved = xcconfigResolvedValue{value: strings.TrimSpace(value), path: path, found: true, exact: exact}
	}
	return resolved, nil
}

func editXCConfig(data []byte, setting, value string) ([]byte, []string, bool, error) {
	document, err := parseXCConfig(data)
	if err != nil {
		return nil, nil, false, err
	}
	assignmentsByLine := make(map[int]xcconfigAssignment)
	var oldValues []string
	for _, assignment := range document.assignments {
		if assignment.baseKey != setting {
			continue
		}
		assignmentsByLine[assignment.lineIndex] = assignment
		oldValues = append(oldValues, assignment.value)
	}
	if len(assignmentsByLine) == 0 {
		return data, nil, false, nil
	}

	changed := false
	for index, assignment := range assignmentsByLine {
		line := document.lines[index]
		if assignment.value == value {
			continue
		}
		document.lines[index] = line[:assignment.valueStart] + value + line[assignment.valueEnd:]
		changed = true
	}
	return []byte(strings.Join(document.lines, "")), oldValues, changed, nil
}

func clonePathSet(source map[string]bool) map[string]bool {
	clone := make(map[string]bool, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}
