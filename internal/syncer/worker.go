package syncer

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/iran-discord-services/stats/internal/channels"
	"github.com/iran-discord-services/stats/internal/config"
	"github.com/iran-discord-services/stats/internal/guild"
	"github.com/iran-discord-services/stats/internal/members"
	"github.com/iran-discord-services/stats/internal/store"
)

type Worker struct {
	cfg      config.Config
	cache    *guild.Cache
	members  *members.Client
	store    *store.Store
	session  *discordgo.Session
	lastSync map[string]time.Time
	mu       sync.Mutex

	syncNow chan string
}

func NewWorker(cfg config.Config, cache *guild.Cache, st *store.Store, session *discordgo.Session) *Worker {
	return &Worker{
		cfg:      cfg,
		cache:    cache,
		members:  members.NewClient(cfg.MembersFetchTimeout, cfg.DiscordToken),
		store:    st,
		session:  session,
		lastSync: make(map[string]time.Time),
		syncNow:  make(chan string, 32),
	}
}

func (w *Worker) RequestSync(guildID string) {
	if guildID == "" {
		return
	}
	select {
	case w.syncNow <- guildID:
	default:
		slog.Debug("sync now queue full, dropping", "guild", guildID)
	}
}

func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.cfg.SyncCycleInterval)
	defer ticker.Stop()

	w.runCycle(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case guildID := <-w.syncNow:
			w.syncGuildImmediate(ctx, guildID)
		case <-ticker.C:
			w.runCycle(ctx)
		}
	}
}

func (w *Worker) syncGuildImmediate(ctx context.Context, guildID string) {
	inviteByGuild, err := w.store.ListListedGuilds(ctx)
	if err != nil {
		slog.Warn("could not load listed guild invites", "err", err)
		inviteByGuild = map[string]string{}
	}
	w.syncGuild(ctx, guildID, inviteByGuild[guildID], true)
}

func (w *Worker) runCycle(ctx context.Context) {
	guildIDs := w.cache.GuildIDs()
	if len(guildIDs) == 0 {
		slog.Debug("sync cycle skipped — bot is not in any guild yet")
		return
	}

	inviteByGuild, err := w.store.ListListedGuilds(ctx)
	if err != nil {
		slog.Warn("could not load listed guild invites", "err", err)
		inviteByGuild = map[string]string{}
	}

	slog.Debug("sync cycle started", "guilds", len(guildIDs))

	for i, guildID := range guildIDs {
		if ctx.Err() != nil {
			return
		}

		if i > 0 {
			select {
			case <-ctx.Done():
				return
			case <-time.After(w.cfg.SyncGuildDelay):
			}
		}

		w.syncGuild(ctx, guildID, inviteByGuild[guildID], false)
	}

	slog.Debug("sync cycle finished", "guilds", len(guildIDs))
}

func (w *Worker) syncGuild(ctx context.Context, guildID, inviteURL string, force bool) {
	membersLoaded := w.cache.MembersLoaded(guildID)

	var humansOnline *int
	if membersLoaded {
		humans := w.cache.HumansOnline(guildID)
		humansOnline = &humans
	}

	fetchCtx, cancel := context.WithTimeout(ctx, w.cfg.MembersFetchTimeout)
	membersTotal, membersErr := w.members.FetchTotalMembers(fetchCtx, guildID, inviteURL)
	cancel()

	if membersErr != nil {
		slog.Debug("member total fetch fallback", "guild", guildID, "err", membersErr)
	}

	widgetOK := w.members.WidgetEnabled(ctx, guildID)

	row := store.GuildStats{
		GuildID:      guildID,
		MembersTotal: membersTotal,
		HumansOnline: humansOnline,
		BotPresent:   true,
		SyncedAt:     time.Now().UTC(),
	}

	if guildID == w.cfg.MainGuildID {
		w.attachMainGuildStats(&row, guildID)
	}

	if !membersLoaded {
		slog.Debug("sync skipped — registry not ready", "guild", guildID)
		return
	}

	if membersTotal == nil {
		slog.Debug("sync skipped — member total unavailable", "guild", guildID)
		return
	}

	if !force && !w.shouldWrite(guildID, row) {
		slog.Debug("sync skipped (debounced)", "guild", guildID, "humans", derefInt(humansOnline), "widget", widgetOK)
		return
	}

	if err := w.store.UpsertStats(ctx, row); err != nil {
		slog.Warn("supabase upsert failed", "guild", guildID, "err", err)
		return
	}

	w.markWritten(guildID)
	registry := w.cache.RegistryStats(guildID)
	slog.Info(
		"guild synced",
		"guild", guildID,
		"humans_online", derefInt(humansOnline),
		"registry_humans", registry.Humans,
		"registry_bots", registry.Bots,
		"members_loaded", registry.MembersLoaded,
		"members_total", derefInt(membersTotal),
		"widget_enabled", widgetOK,
	)
}

func (w *Worker) attachMainGuildStats(row *store.GuildStats, guildID string) {
	tallies := w.cache.TalliesFor(guildID)
	if tallies.Ready {
		realUsers := tallies.RealUsers
		botsTotal := tallies.BotsTotal
		botsOnline := tallies.BotsOnline
		row.RealUsers = &realUsers
		row.BotsTotal = &botsTotal
		row.BotsOnline = &botsOnline
	}

	chans, chansErr := w.session.GuildChannels(guildID)
	roles, rolesErr := w.session.GuildRoles(guildID)
	if chansErr != nil || rolesErr != nil {
		slog.Debug("main guild channels fetch failed", "chans_err", chansErr, "roles_err", rolesErr)
		return
	}

	total := channels.CountMemberVisible(chans, roles, guildID, w.cfg.MemberRoleID)
	row.ChannelsTotal = &total
}

func (w *Worker) shouldWrite(guildID string, next store.GuildStats) bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	prevAt, ok := w.lastSync[guildID]
	if !ok {
		return true
	}
	return time.Since(prevAt) >= w.cfg.SyncDebounce
}

func (w *Worker) markWritten(guildID string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.lastSync[guildID] = time.Now().UTC()
}

func derefInt(v *int) any {
	if v == nil {
		return nil
	}
	return *v
}
