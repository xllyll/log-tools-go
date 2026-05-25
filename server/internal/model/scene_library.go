package model

import "time"

type SceneLibraryItem struct {
	ID          string    `json:"id"`
	DeviceID    string    `json:"device_id"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	ModuleCount int       `json:"module_count"`
	SceneCount  int       `json:"scene_count"`
	UpdatedAt   time.Time `json:"updated_at"`
	IsMine      bool      `json:"is_mine,omitempty"`
}

type SceneLibraryDetail struct {
	SceneLibraryItem
	Config SceneConfig `json:"config"`
}
