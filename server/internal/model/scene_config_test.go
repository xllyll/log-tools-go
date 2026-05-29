package model

import (
	"encoding/json"
	"testing"
)

func TestSceneKeywordUnmarshalLegacy(t *testing.T) {
	raw := []byte(`{"keyword":"test","desc":"d","mode":"word","color":"#fff"}`)
	var kw SceneKeyword
	if err := json.Unmarshal(raw, &kw); err != nil {
		t.Fatal(err)
	}
	if kw.Mode != SceneKwModeWord || kw.CaseSensitive != SceneKwCaseIgnore {
		t.Fatalf("got mode=%d case=%d", kw.Mode, kw.CaseSensitive)
	}

	raw2 := []byte(`{"keyword":"x","mode":"regex","case_sensitive":1}`)
	if err := json.Unmarshal(raw2, &kw); err != nil {
		t.Fatal(err)
	}
	if kw.Mode != SceneKwModeRegex || kw.CaseSensitive != SceneKwCaseSensitive {
		t.Fatalf("got mode=%d case=%d", kw.Mode, kw.CaseSensitive)
	}
}

func TestParseSceneConfigJSONLegacy(t *testing.T) {
	raw := []byte(`{"modules":[{"name":"m","scenes":[{"name":"s","keywords":[{"keyword":"a","mode":"word"}]}]}]}`)
	cfg, err := ParseSceneConfigJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	kw := cfg.Modules[0].Scenes[0].Keywords[0]
	if kw.Mode != SceneKwModeWord || kw.CaseSensitive != SceneKwCaseIgnore {
		t.Fatalf("defaults: mode=%d case=%d", kw.Mode, kw.CaseSensitive)
	}
}
