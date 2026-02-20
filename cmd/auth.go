package cmd

import (
	"github.com/ontypehq/vox/internal/config"
	"github.com/ontypehq/vox/internal/ui"
)

type AuthCmd struct {
	Login  AuthLoginCmd  `cmd:"" help:"Save API credentials"`
	Status AuthStatusCmd `cmd:"" help:"Show current auth status"`
}

type AuthLoginCmd struct {
	Provider string `arg:"" help:"Auth provider (dashscope)" enum:"dashscope"`
	Token    string `required:"" help:"API key"`
}

func (c *AuthLoginCmd) Run(cfg *config.AppConfig) error {
	cfg.Config.Provider = c.Provider
	cfg.Config.APIKey = c.Token
	if err := cfg.SaveConfig(); err != nil {
		return err
	}
	ui.Success("Authenticated with %s", ui.Key(c.Provider))
	ui.KV("Config", cfg.Dir+"/config.json")
	return nil
}

type AuthStatusCmd struct{}

func (c *AuthStatusCmd) Run(cfg *config.AppConfig) error {
	if cfg.Config.APIKey == "" {
		ui.Warn("Not authenticated")
		ui.Info("  Run: %s", ui.Key("vox auth login dashscope --token <key>"))
		return nil
	}

	masked := cfg.Config.APIKey[:6] + "..." + cfg.Config.APIKey[len(cfg.Config.APIKey)-4:]
	ui.Success("Authenticated")
	ui.KV("Provider", cfg.Config.Provider)
	ui.KV("API Key", masked)
	return nil
}
