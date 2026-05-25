package xlog

import (
	"regexp"
	"strings"
)

// 常见 logcat: "05-25 10:12:03.123  1234  5678 E Tag: msg" 或 "E/Tag(pid): msg"
var (
	logcatLevelRe = regexp.MustCompile(`\s([VDIWEF])\s+\S`)
	slashLevelRe  = regexp.MustCompile(`\b([VDIWEF])/`)
	bracketRe     = regexp.MustCompile(`\[(VERBOSE|DEBUG|INFO|WARN(?:ING)?|ERROR|FATAL|ASSERT)\]`)
)

func ParseLevel(line string) string {
	u := strings.ToUpper(line)
	if m := bracketRe.FindStringSubmatch(u); len(m) > 1 {
		return normalizeLevel(m[1])
	}
	if m := slashLevelRe.FindStringSubmatch(line); len(m) > 1 {
		return letterLevel(m[1])
	}
	if m := logcatLevelRe.FindStringSubmatch(line); len(m) > 1 {
		return letterLevel(m[1])
	}
	for _, kw := range []struct{ k, v string }{
		{"FATAL", "FATAL"},
		{"ERROR", "ERROR"},
		{"WARN", "WARN"},
		{"WARNING", "WARN"},
		{"INFO", "INFO"},
		{"DEBUG", "DEBUG"},
		{"VERBOSE", "VERBOSE"},
		{"ASSERT", "ASSERT"},
	} {
		if strings.Contains(u, kw.k) {
			return kw.v
		}
	}
	return "INFO"
}

func letterLevel(c string) string {
	switch strings.ToUpper(c) {
	case "V":
		return "VERBOSE"
	case "D":
		return "DEBUG"
	case "I":
		return "INFO"
	case "W":
		return "WARN"
	case "E":
		return "ERROR"
	case "F":
		return "FATAL"
	case "A":
		return "ASSERT"
	default:
		return "INFO"
	}
}

func normalizeLevel(s string) string {
	s = strings.ToUpper(s)
	if strings.HasPrefix(s, "WARN") {
		return "WARN"
	}
	return s
}
