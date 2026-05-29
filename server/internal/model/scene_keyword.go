package model

import (
	"encoding/json"
	"regexp"
	"strings"
)

const (
	SceneKwModeWord  = 0
	SceneKwModeRegex = 1
	SceneKwCaseIgnore    = 0
	SceneKwCaseSensitive = 1
)

// NormalizeSceneKeywordFilter 统一 mode / case_sensitive（默认 0）
func NormalizeSceneKeywordFilter(sk *SceneKeywordFilter) {
	if sk == nil {
		return
	}
	sk.Mode = normalizeModeValue(sk.Mode)
	sk.CaseSensitive = normalizeCaseValue(sk.CaseSensitive)
}

func normalizeModeValue(v int) int {
	if v == SceneKwModeRegex {
		return SceneKwModeRegex
	}
	return SceneKwModeWord
}

func normalizeCaseValue(v int) int {
	if v == SceneKwCaseSensitive {
		return SceneKwCaseSensitive
	}
	return SceneKwCaseIgnore
}

// ParseSceneKeywordFilterJSON 兼容 mode/case_sensitive 为字符串的旧数据
func ParseSceneKeywordFilterJSON(raw json.RawMessage) SceneKeywordFilter {
	var sk SceneKeywordFilter
	var flex struct {
		Keyword       string          `json:"keyword"`
		Mode          json.RawMessage `json:"mode"`
		CaseSensitive json.RawMessage `json:"case_sensitive"`
	}
	if json.Unmarshal(raw, &flex) != nil {
		return sk
	}
	sk.Keyword = flex.Keyword
	sk.Mode = parseIntOrLegacyMode(flex.Mode)
	sk.CaseSensitive = parseIntOrZero(flex.CaseSensitive)
	NormalizeSceneKeywordFilter(&sk)
	return sk
}

func parseIntOrLegacyMode(raw json.RawMessage) int {
	if len(raw) == 0 {
		return SceneKwModeWord
	}
	var n int
	if json.Unmarshal(raw, &n) == nil {
		return normalizeModeValue(n)
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		switch strings.ToLower(strings.TrimSpace(s)) {
		case "1", "regex":
			return SceneKwModeRegex
		default:
			return SceneKwModeWord
		}
	}
	return SceneKwModeWord
}

func parseIntOrZero(raw json.RawMessage) int {
	if len(raw) == 0 {
		return SceneKwCaseIgnore
	}
	var n int
	if json.Unmarshal(raw, &n) == nil {
		return normalizeCaseValue(n)
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		switch strings.ToLower(strings.TrimSpace(s)) {
		case "1", "true", "yes":
			return SceneKwCaseSensitive
		}
	}
	return SceneKwCaseIgnore
}

// UnmarshalJSON 兼容旧配置：mode 为 "word"/"regex"，缺失 case_sensitive 默认为 0
func (sk *SceneKeyword) UnmarshalJSON(data []byte) error {
	var flex struct {
		Keyword       string          `json:"keyword"`
		Desc          string          `json:"desc"`
		Mode          json.RawMessage `json:"mode"`
		CaseSensitive json.RawMessage `json:"case_sensitive"`
		Color         string          `json:"color"`
	}
	if err := json.Unmarshal(data, &flex); err != nil {
		return err
	}
	sk.Keyword = flex.Keyword
	sk.Desc = flex.Desc
	sk.Color = flex.Color
	sk.Mode = parseIntOrLegacyMode(flex.Mode)
	sk.CaseSensitive = parseIntOrZero(flex.CaseSensitive)
	return nil
}

// ParseSceneConfigJSON 解析场景配置并填充新字段默认值
func ParseSceneConfigJSON(raw []byte) (SceneConfig, error) {
	var cfg SceneConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return cfg, err
	}
	NormalizeSceneConfig(&cfg)
	return cfg, nil
}

// NormalizeSceneConfig 确保每条关键词 mode/case_sensitive 为合法数值
func NormalizeSceneConfig(cfg *SceneConfig) {
	if cfg == nil {
		return
	}
	for mi := range cfg.Modules {
		for si := range cfg.Modules[mi].Scenes {
			for ki := range cfg.Modules[mi].Scenes[si].Keywords {
				kw := &cfg.Modules[mi].Scenes[si].Keywords[ki]
				kw.Mode = normalizeModeValue(kw.Mode)
				kw.CaseSensitive = normalizeCaseValue(kw.CaseSensitive)
			}
		}
	}
}

func sceneKeywordHit(content string, sk SceneKeywordFilter) bool {
	if sk.Keyword == "" {
		return false
	}
	if sk.Mode == SceneKwModeRegex {
		flags := ""
		if sk.CaseSensitive != SceneKwCaseSensitive {
			flags = "(?i)"
		}
		re, err := regexp.Compile(flags + sk.Keyword)
		return err == nil && re.MatchString(content)
	}
	if sk.CaseSensitive == SceneKwCaseSensitive {
		return strings.Contains(content, sk.Keyword)
	}
	return strings.Contains(strings.ToLower(content), strings.ToLower(sk.Keyword))
}
