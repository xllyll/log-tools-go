package xmatch

import (
	"fmt"
	"regexp"
	"sync"
)

var regexCache sync.Map // pattern -> *regexp.Regexp

func compilePattern(pattern string) (*regexp.Regexp, error) {
	if pattern == "" {
		return nil, nil
	}
	if cached, ok := regexCache.Load(pattern); ok {
		return cached.(*regexp.Regexp), nil
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	actual, _ := regexCache.LoadOrStore(pattern, re)
	return actual.(*regexp.Regexp), nil
}

func matchSub(re *regexp.Regexp, str string) string {
	if re == nil {
		return ""
	}
	match := re.FindStringSubmatch(str)
	if len(match) > 1 {
		return match[1]
	}
	return ""
}

// Match 通过正则提取第一个捕获组（pattern 会缓存编译结果）
func Match(pattern string, str string) string {
	if pattern == "" {
		return ""
	}
	re, err := compilePattern(pattern)
	if err != nil {
		fmt.Println(err)
		return ""
	}
	return matchSub(re, str)
}

// RuleMatcher 预编译一组字段正则，避免每行重复 Compile
type RuleMatcher struct {
	timestamp *regexp.Regexp
	process   *regexp.Regexp
	thread    *regexp.Regexp
	level     *regexp.Regexp
	module    *regexp.Regexp
	class     *regexp.Regexp
	classLine *regexp.Regexp
	tag       *regexp.Regexp
	message   *regexp.Regexp
}

func NewRuleMatcher(timestamp, process, thread, level, module, class, classLine, tag, message string) *RuleMatcher {
	must := func(p string) *regexp.Regexp {
		re, _ := compilePattern(p)
		return re
	}
	return &RuleMatcher{
		timestamp: must(timestamp),
		process:   must(process),
		thread:    must(thread),
		level:     must(level),
		module:    must(module),
		class:     must(class),
		classLine: must(classLine),
		tag:       must(tag),
		message:   must(message),
	}
}

func (m *RuleMatcher) Timestamp(s string) string { return matchSub(m.timestamp, s) }
func (m *RuleMatcher) Process(s string) string   { return matchSub(m.process, s) }
func (m *RuleMatcher) Thread(s string) string    { return matchSub(m.thread, s) }
func (m *RuleMatcher) Level(s string) string     { return matchSub(m.level, s) }
func (m *RuleMatcher) Module(s string) string    { return matchSub(m.module, s) }
func (m *RuleMatcher) Class(s string) string     { return matchSub(m.class, s) }
func (m *RuleMatcher) ClassLine(s string) string { return matchSub(m.classLine, s) }
func (m *RuleMatcher) Tag(s string) string       { return matchSub(m.tag, s) }
func (m *RuleMatcher) Message(s string) string   { return matchSub(m.message, s) }
