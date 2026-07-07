package presence

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/iran-discord-services/stats/internal/config"
	"github.com/iran-discord-services/stats/internal/guild"
	"github.com/iran-discord-services/stats/internal/persist"
)

type Manager struct {
	cfg     config.Config
	session *discordgo.Session
	cache   *guild.Cache
	persist *persist.Store

	mu          sync.Mutex
	lastPush    time.Time
	lastServers int
	lastUsers   int
}

func NewManager(cfg config.Config, session *discordgo.Session, cache *guild.Cache, store *persist.Store) *Manager {
	return &Manager{
		cfg:     cfg,
		session: session,
		cache:   cache,
		persist: store,
	}
}

func (m *Manager) Run(ctx context.Context) {
	if !m.cfg.PresenceEnabled {
		return
	}

	ticker := time.NewTicker(m.cfg.PresenceStatsInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.refreshIfChanged()
		}
	}
}

func (m *Manager) OnReady() {
	if !m.cfg.PresenceEnabled {
		return
	}
	if stats := m.currentStats(); stats.TotalUsers == 0 && stats.Servers > 0 {
		slog.Info("presence: waiting for member counts before first update", "servers", stats.Servers)
		return
	}
	m.push()
}

func (m *Manager) OnGuildSynced() {
	if !m.cfg.PresenceEnabled {
		return
	}
	m.push()
}

func (m *Manager) refreshIfChanged() {
	stats := m.currentStats()

	m.mu.Lock()
	changed := stats.Servers != m.lastServers || stats.TotalUsers != m.lastUsers
	improved := stats.TotalUsers > m.lastUsers
	debounced := time.Since(m.lastPush) < m.cfg.PresenceStatsInterval
	m.mu.Unlock()

	if !changed {
		return
	}
	if debounced && !improved {
		return
	}
	m.push()
}

func (m *Manager) currentStats() guild.PresenceStats {
	if m.persist == nil {
		return m.cache.PresenceStats()
	}
	return m.cache.PresenceStatsHybrid(m.persist.AllGuilds())
}

func (m *Manager) push() {
	stats := m.currentStats()
	if stats.TotalUsers == 0 && stats.Servers > 0 {
		slog.Debug("presence: skip update — member counts not ready", "servers", stats.Servers)
		return
	}
	actType := activityType(m.cfg)

	servers := formatCompact(stats.Servers)
	members := formatCompact(stats.TotalUsers)
	summary := fmt.Sprintf("%s servers & %s members", servers, members)

	activity := &discordgo.Activity{
		Type: actType,
	}

	switch actType {
	case discordgo.ActivityTypeStreaming:
		activity.Name = m.cfg.PresenceBrandName
		activity.State = fmt.Sprintf("%s · %s", summary, m.cfg.PresenceWebsiteURL)
		activity.URL = m.cfg.PresenceStreamURL
	default:
		activity.Name = summary
	}

	err := m.session.UpdateStatusComplex(discordgo.UpdateStatusData{
		Status:     string(discordgo.StatusOnline),
		Activities: []*discordgo.Activity{activity},
	})
	if err != nil {
		slog.Warn("presence: update failed", "err", err)
		return
	}

	m.mu.Lock()
	m.lastPush = time.Now()
	m.lastServers = stats.Servers
	m.lastUsers = stats.TotalUsers
	m.mu.Unlock()

	slog.Info(
		"presence: updated",
		"type", m.cfg.PresenceActivityType,
		"name", activity.Name,
		"state", activity.State,
		"url", activity.URL,
	)
}
