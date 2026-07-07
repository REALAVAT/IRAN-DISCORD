package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	DiscordToken           string
	DiscordApplicationID   string
	SupabaseURL            string
	SupabaseServiceRoleKey string
	SyncCycleInterval      time.Duration
	SyncGuildDelay         time.Duration
	SyncDebounce           time.Duration
	MembersFetchTimeout    time.Duration
	LogLevel               string
	DataDir                string
	MemberSyncDelay        time.Duration
	StateSaveInterval      time.Duration
	MainGuildID            string
	MemberRoleID           string
	BotRoleID              string
	IranServiceRoleID      string
	PresenceEnabled        bool
	PresenceStatsInterval  time.Duration
	PresenceActivityType   string
	PresenceStreamURL      string
	PresenceWebsiteURL     string
	PresenceBrandName      string
}

func Load() (Config, error) {
	_ = godotenv.Load()

	cfg := Config{
		DiscordToken:           os.Getenv("DISCORD_BOT_TOKEN"),
		DiscordApplicationID:   os.Getenv("DISCORD_APPLICATION_ID"),
		SupabaseURL:            os.Getenv("NEXT_PUBLIC_SUPABASE_URL"),
		SupabaseServiceRoleKey: os.Getenv("SUPABASE_SERVICE_ROLE_KEY"),
		SyncCycleInterval:      durationEnv("SYNC_CYCLE_INTERVAL", 5*time.Minute),
		SyncGuildDelay:         time.Duration(intEnv("SYNC_GUILD_DELAY_MS", 2500)) * time.Millisecond,
		SyncDebounce:           durationEnv("SYNC_DEBOUNCE", 45*time.Second),
		MembersFetchTimeout:    durationEnv("MEMBERS_FETCH_TIMEOUT", 12*time.Second),
		LogLevel:               envOr("LOG_LEVEL", "warn"),
		DataDir:                envOr("DATA_DIR", "data"),
		MemberSyncDelay:        time.Duration(intEnv("MEMBER_SYNC_DELAY_MS", 5000)) * time.Millisecond,
		StateSaveInterval:      durationEnv("STATE_SAVE_INTERVAL", 60*time.Second),
		MainGuildID:            envOr("MAIN_GUILD_ID", "884862760926740510"),
		MemberRoleID:           envOr("MAIN_GUILD_MEMBER_ROLE_ID", "884905717323141161"),
		BotRoleID:              envOr("MAIN_GUILD_BOT_ROLE_ID", "884916642235154482"),
		IranServiceRoleID:      envOr("MAIN_GUILD_SERVICE_ROLE_ID", "896697098358124554"),
		PresenceEnabled:        boolEnv("PRESENCE_ENABLED", true),
		PresenceStatsInterval:  durationEnv("PRESENCE_STATS_INTERVAL", 30*time.Second),
		PresenceActivityType:   envOr("PRESENCE_ACTIVITY_TYPE", "watching"),
		PresenceStreamURL:      envOr("PRESENCE_STREAM_URL", "https://twitch.tv/discord"),
		PresenceWebsiteURL:     envOr("PRESENCE_WEBSITE_URL", "irandiscord.com"),
		PresenceBrandName:      envOr("PRESENCE_BRAND_NAME", "IRAN DISCORD"),
	}

	if cfg.DiscordToken == "" {
		return cfg, fmt.Errorf("DISCORD_BOT_TOKEN is required")
	}
	if cfg.SupabaseURL == "" || cfg.SupabaseServiceRoleKey == "" {
		return cfg, fmt.Errorf("NEXT_PUBLIC_SUPABASE_URL and SUPABASE_SERVICE_ROLE_KEY are required")
	}

	return cfg, nil
}

func (c Config) InviteURL() string {
	if c.DiscordApplicationID == "" {
		return ""
	}
	return fmt.Sprintf(
		"https://discord.com/api/oauth2/authorize?client_id=%s&permissions=0&scope=bot",
		c.DiscordApplicationID,
	)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func boolEnv(key string, fallback bool) bool {
	raw := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	if raw == "" {
		return fallback
	}
	switch raw {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func intEnv(key string, fallback int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return n
}

func durationEnv(key string, fallback time.Duration) time.Duration {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return fallback
	}
	return d
}
