package presence

import (
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/iran-discord-services/stats/internal/config"
)

func TestActivityType(t *testing.T) {
	cases := map[string]discordgo.ActivityType{
		"streaming": discordgo.ActivityTypeStreaming,
		"watching":  discordgo.ActivityTypeWatching,
		"":          discordgo.ActivityTypeWatching,
		"playing":   discordgo.ActivityTypeWatching,
	}
	for input, want := range cases {
		cfg := config.Config{PresenceActivityType: input}
		if got := activityType(cfg); got != want {
			t.Fatalf("activityType(%q) = %v, want %v", input, got, want)
		}
	}
}
