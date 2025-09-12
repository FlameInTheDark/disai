package config

import "github.com/ilyakaznacheev/cleanenv"

type Config struct {
	Token         string               `yaml:"token" env:"DISCORD_TOKEN"`
	MCPServers    map[string]MCPServer `yaml:"mcpServers"`
	OllamaServers map[string]string    `yaml:"ollamaServers"`
	Model         string               `yaml:"model"`
	Whitelist     []int64              `yaml:"whitelist"`
	Templates     Templates            `yaml:"templates"`
	ToolNames     map[string]string    `yaml:"toolNames"`
}

type MCPServer struct {
	URL     string   `yaml:"url"`
	Command string   `yaml:"command"`
	Args    []string `yaml:"args"`
	Env     []string `yaml:"env"`
}

type Templates struct {
	System string `yaml:"system"`
	User   string `yaml:"user"`
}

func NewConfig(path string) Config {
	var cfg Config
	err := cleanenv.ReadConfig(path, &cfg)
	if err != nil {
		panic(err)
	}
	return cfg
}
