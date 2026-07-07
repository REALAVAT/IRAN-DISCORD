package presence

import (
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/iran-discord-services/stats/internal/config"
)

func activityType(cfg config.Config) discordgo.ActivityType {
	switch strings.ToLower(strings.TrimSpace(cfg.PresenceActivityType)) {
	case "streaming":
		return discordgo.ActivityTypeStreaming
	default:
		return discordgo.ActivityTypeWatching
	}
}
