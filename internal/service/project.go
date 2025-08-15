package service

import (
	"encoding/json"
	"log"
	"log-tools-go/internal/config"
	"log-tools-go/internal/model"
	"os"
)

type ProjectService struct {
	config   *config.Config
	parser   *LogParser
	database *model.Database
}

func NewProjectService(cfg *config.Config, parser *LogParser, database *model.Database) *ProjectService {
	return &ProjectService{
		config:   cfg,
		parser:   parser,
		database: database,
	}
}

func (s *ProjectService) GetAllProjects() ([]*config.LogProjectRule, error) {
	// 读取 /config/config.json 文件
	file, err := os.Open("config/config.json")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	// 读取文件内容
	ps := make([]*config.LogProjectRule, 0)
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&ps)
	if err != nil {
		log.Fatalf("解析 JSON 失败: %v", err)
	}
	return ps, nil
}
