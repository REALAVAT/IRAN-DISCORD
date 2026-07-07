package presence

import (
	"log/slog"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type assetResolver struct {
	byName map[string]string
}

func newAssetResolver() *assetResolver {
	return &assetResolver{byName: make(map[string]string)}
}

func (r *assetResolver) load(session *discordgo.Session, appID string) {
	if appID == "" {
		return
	}

	assets, err := session.ApplicationAssets(appID)
	if err != nil {
		slog.Warn("presence: failed to load assets", "err", err)
		return
	}

	for _, a := range assets {
		name := strings.ToLower(strings.TrimSpace(a.Name))
		id := strings.TrimSpace(a.ID)
		if name != "" && id != "" {
			r.byName[name] = id
		}
	}

	slog.Info("presence: assets loaded", "count", len(r.byName))
}

func (r *assetResolver) resolve(key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return ""
	}
	if strings.HasPrefix(key, "http://") || strings.HasPrefix(key, "https://") {
		return key
	}
	if _, err := strconv.ParseUint(key, 10, 64); err == nil {
		return key
	}
	if id, ok := r.byName[strings.ToLower(key)]; ok {
		return id
	}
	return key
}
