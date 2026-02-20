package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/ontypehq/vox/internal/audio"
	"github.com/ontypehq/vox/internal/config"
	"github.com/ontypehq/vox/internal/dashscope"
	"github.com/ontypehq/vox/internal/ui"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
)

type ListenCmd struct {
	Channel string `short:"c" help:"Channel name or ID to listen to (default: all)"`
	Voice   string `short:"v" help:"Voice to use for TTS"`
	Speed   float64 `short:"s" default:"1.2" help:"Speech rate (0.5-2.0)"`
}

func (c *ListenCmd) Run(cfg *config.AppConfig) error {
	apiKey, err := cfg.RequireAPIKey()
	if err != nil {
		return err
	}
	botToken, appToken, err := cfg.RequireSlack()
	if err != nil {
		return err
	}

	// Resolve voice
	voice := c.Voice
	if voice == "" {
		voice = cfg.State.LastVoice
	}
	if voice == "" {
		voice = "Cherry"
	}

	api := slack.New(botToken, slack.OptionAppLevelToken(appToken))
	client := socketmode.New(api, socketmode.OptionLog(log.New(os.Stderr, "", 0)))

	// Get bot's own user ID to skip self messages
	authResp, err := api.AuthTest()
	if err != nil {
		return fmt.Errorf("slack auth test: %w", err)
	}
	botUserID := authResp.UserID

	// Resolve channel filter
	var filterChannelID string
	if c.Channel != "" {
		filterChannelID = c.resolveChannel(api, c.Channel)
	}

	// Build user name cache
	userNames := map[string]string{}
	getName := func(userID string) string {
		if name, ok := userNames[userID]; ok {
			return name
		}
		if info, err := api.GetUserInfo(userID); err == nil {
			name := info.Profile.DisplayName
			if name == "" {
				name = info.RealName
			}
			if name == "" {
				name = info.Name
			}
			userNames[userID] = name
			return name
		}
		return "someone"
	}

	ttsClient := dashscope.NewRealtimeClient(apiKey)
	model := dashscope.ModelForVoice(voice)

	ui.Success("Listening on Slack")
	ui.KV("Voice", voice)
	if filterChannelID != "" {
		ui.KV("Channel", c.Channel)
	} else {
		ui.KV("Channel", "all")
	}
	ui.Info("%s", ui.Dim("Press Ctrl+C to stop"))

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	go func() {
		for evt := range client.Events {
			switch evt.Type {
			case socketmode.EventTypeEventsAPI:
				eventsAPIEvent, ok := evt.Data.(slackevents.EventsAPIEvent)
				if !ok {
					continue
				}
				client.Ack(*evt.Request)

				switch ev := eventsAPIEvent.InnerEvent.Data.(type) {
				case *slackevents.MessageEvent:
					// Skip bot's own messages, bot messages, and message changes
					if ev.User == botUserID || ev.BotID != "" || ev.SubType != "" {
						continue
					}

					// Channel filter
					if filterChannelID != "" && ev.Channel != filterChannelID {
						continue
					}

					text := ev.Text
					if text == "" {
						continue
					}

					// Clean up slack formatting
					text = cleanSlackText(text)

					sender := getName(ev.User)
					spoken := fmt.Sprintf("%s says: %s", sender, text)

					ui.Info("%s %s: %s", ui.Dim(time.Now().Format("15:04")), ui.Key(sender), text)

					// Speak it
					player := audio.NewStreamPlayer()
					opts := dashscope.TTSOptions{
						Model:      model,
						Voice:      voice,
						Text:       spoken,
						Lang:       "auto",
						SpeechRate: c.Speed,
					}
					ttsCtx, ttsCancel := context.WithTimeout(context.Background(), 30*time.Second)
					ttsClient.StreamTTS(ttsCtx, opts, func(pcm []byte) {
						player.Write(pcm)
					})
					player.Close()
					ttsCancel()
				}

			case socketmode.EventTypeConnectionError:
				ui.Warn("Connection error, reconnecting...")

			case socketmode.EventTypeConnecting:
				ui.Info("%s", ui.Dim("connecting..."))

			case socketmode.EventTypeConnected:
				ui.Info("%s", ui.Dim("connected"))
			}
		}
	}()

	go client.RunContext(ctx)

	<-ctx.Done()
	ui.Info("\n%s", ui.Dim("stopped"))
	return nil
}

func (c *ListenCmd) resolveChannel(api *slack.Client, channel string) string {
	// If it looks like an ID already
	if strings.HasPrefix(channel, "C") || strings.HasPrefix(channel, "G") || strings.HasPrefix(channel, "D") {
		return channel
	}
	// Search by name
	channel = strings.TrimPrefix(channel, "#")
	params := &slack.GetConversationsParameters{Limit: 200}
	channels, _, err := api.GetConversations(params)
	if err != nil {
		return channel
	}
	for _, ch := range channels {
		if ch.Name == channel {
			return ch.ID
		}
	}
	return channel
}

// cleanSlackText removes slack markup like <@U123> mentions, <url|label> links, etc.
func cleanSlackText(text string) string {
	// Replace user mentions <@U123> with empty (we already prefix with sender name)
	result := text
	for {
		start := strings.Index(result, "<@")
		if start == -1 {
			break
		}
		end := strings.Index(result[start:], ">")
		if end == -1 {
			break
		}
		result = result[:start] + result[start+end+1:]
	}

	// Replace <url|label> with label, or <url> with url
	for {
		start := strings.Index(result, "<")
		if start == -1 {
			break
		}
		end := strings.Index(result[start:], ">")
		if end == -1 {
			break
		}
		inner := result[start+1 : start+end]
		if pipe := strings.Index(inner, "|"); pipe != -1 {
			inner = inner[pipe+1:]
		}
		result = result[:start] + inner + result[start+end+1:]
	}

	return strings.TrimSpace(result)
}
