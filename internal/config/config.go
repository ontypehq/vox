package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const appDir = ".vox"

type Config struct {
	Provider string `json:"provider,omitempty"`
	APIKey   string `json:"api_key,omitempty"`
}

type State struct {
	LastVoice string `json:"last_voice,omitempty"`
	LastLang  string `json:"last_lang,omitempty"`
}

type AppConfig struct {
	Config Config
	State  State
	Dir    string
}

func Dir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, appDir)
}

func Load() (*AppConfig, error) {
	dir := Dir()
	os.MkdirAll(dir, 0755)
	os.MkdirAll(filepath.Join(dir, "voices"), 0755)
	os.MkdirAll(filepath.Join(dir, "cache"), 0755)

	ac := &AppConfig{Dir: dir}

	if data, err := os.ReadFile(filepath.Join(dir, "config.json")); err == nil {
		json.Unmarshal(data, &ac.Config)
	}
	if data, err := os.ReadFile(filepath.Join(dir, "state.json")); err == nil {
		json.Unmarshal(data, &ac.State)
	}

	return ac, nil
}

func (ac *AppConfig) SaveConfig() error {
	return writeJSON(filepath.Join(ac.Dir, "config.json"), ac.Config)
}

func (ac *AppConfig) SaveState() error {
	return writeJSON(filepath.Join(ac.Dir, "state.json"), ac.State)
}

func (ac *AppConfig) RequireAPIKey() (string, error) {
	if ac.Config.APIKey == "" {
		return "", fmt.Errorf("not authenticated — run: vox auth login dashscope --token <key>")
	}
	return ac.Config.APIKey, nil
}

func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}
