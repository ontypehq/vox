package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const appDir = ".vox"

type DashScopeConfig struct {
	APIKey string `json:"api_key,omitempty"`
}

type SlackConfig struct {
	BotToken string `json:"bot_token,omitempty"`
	AppToken string `json:"app_token,omitempty"`
}

type Services struct {
	DashScope DashScopeConfig `json:"dashscope,omitempty"`
	Slack     SlackConfig     `json:"slack,omitempty"`
}

type ListenConfig struct {
	VoiceMap map[string]string `json:"voice_map,omitempty"` // slack user ID or display name → voice
}

type Config struct {
	Services Services     `json:"services"`
	Listen   ListenConfig `json:"listen,omitempty"`
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

	configPath := filepath.Join(dir, "config.json")
	if data, err := os.ReadFile(configPath); err == nil {
		// Try new format first
		if err := json.Unmarshal(data, &ac.Config); err != nil || ac.Config.Services.DashScope.APIKey == "" {
			// Try legacy format migration
			var legacy struct {
				Provider string `json:"provider"`
				APIKey   string `json:"api_key"`
			}
			if err := json.Unmarshal(data, &legacy); err == nil && legacy.APIKey != "" {
				ac.Config.Services.DashScope.APIKey = legacy.APIKey
				// Save migrated config
				ac.SaveConfig()
			}
		}
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
	key := ac.Config.Services.DashScope.APIKey
	if key == "" {
		return "", fmt.Errorf("not authenticated — run: vox auth login dashscope --token <key>")
	}
	return key, nil
}

func (ac *AppConfig) RequireSlack() (botToken, appToken string, err error) {
	s := ac.Config.Services.Slack
	if s.BotToken == "" || s.AppToken == "" {
		return "", "", fmt.Errorf("slack not configured — run: vox auth login slack --bot-token <token> --app-token <token>")
	}
	return s.BotToken, s.AppToken, nil
}

func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}
