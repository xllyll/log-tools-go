package model

// SceneKeywordMatches 任一场景关键字命中即可（OR）
func SceneKeywordMatches(content string, specs []SceneKeywordFilter) bool {
	if len(specs) == 0 {
		return true
	}
	for _, sk := range specs {
		if sceneKeywordHit(content, sk) {
			return true
		}
	}
	return false
}
