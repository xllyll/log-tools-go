package model

import "testing"

func TestSceneKeywordMatches(t *testing.T) {
	content := "05-25 E Tag: onLocalAccChanged: true"
	if !SceneKeywordMatches(content, []SceneKeywordFilter{
		{Keyword: `onLocalAccChanged:\s*true`, Mode: SceneKwModeRegex, CaseSensitive: SceneKwCaseIgnore},
	}) {
		t.Fatal("expected regex case-insensitive match")
	}
	if SceneKeywordMatches(content, []SceneKeywordFilter{
		{Keyword: "onlocalaccchanged", Mode: SceneKwModeWord, CaseSensitive: SceneKwCaseSensitive},
	}) {
		t.Fatal("case-sensitive word should not match different case")
	}
	if !SceneKeywordMatches(content, []SceneKeywordFilter{
		{Keyword: "onLocalAccChanged", Mode: SceneKwModeWord, CaseSensitive: SceneKwCaseSensitive},
	}) {
		t.Fatal("expected case-sensitive word match")
	}
	if SceneKeywordMatches(content, []SceneKeywordFilter{{Keyword: "[", Mode: SceneKwModeRegex}}) {
		t.Fatal("invalid regex should not match")
	}
}
