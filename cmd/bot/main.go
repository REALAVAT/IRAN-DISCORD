package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/iran-discord-services/stats/internal/bot"
	"github.com/iran-discord-services/stats/internal/config"
	"github.com/iran-discord-services/stats/internal/guild"
	"github.com/iran-discord-services/stats/internal/membersync"
	"github.com/iran-discord-services/stats/internal/persist"
	"github.com/iran-discord-services/stats/internal/presence"
	"github.com/iran-discord-services/stats/internal/store"
	"github.com/iran-discord-services/stats/internal/syncer"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("config error", "err", err)
		os.Exit(1)
	}

	setupLogger(cfg.LogLevel)

	disk, err := persist.NewStore(cfg.DataDir)
	if err != nil {
		slog.Error("state store", "err", err)
		os.Exit(1)
	}

	cache := guild.NewCache()
	for _, snap := range disk.AllGuilds() {
		if snap.MembersLoaded {
			cache.EnsureGuild(snap.GuildID)
		}
	}

	savedGuilds, savedReady, savedOnline := disk.GlobalStats()
	slog.Info(
		"restored state",
		"guilds", savedGuilds,
		"guilds_ready", savedReady,
		"humans_online", savedOnline,
		"data_dir", cfg.DataDir,
	)

	st := store.New(cfg.SupabaseURL, cfg.SupabaseServiceRoleKey)

	session, err := discordgo.New("Bot " + cfg.DiscordToken)
	if err != nil {
		slog.Error("discord session", "err", err)
		os.Exit(1)
	}

	syncWorker := syncer.NewWorker(cfg, cache, st, session)

	pres := presence.NewManager(cfg, session, cache, disk)
	syncQ := membersync.NewQueue(session, cache, cfg.MemberSyncDelay)

	onGuildSynced := func(guildID string) {
		disk.UpsertGuild(cache.Snapshot(guildID))
		pres.OnGuildSynced()
		syncWorker.RequestSync(guildID)
	}

	onGuildRemoved := func(guildID string) {
		disk.RemoveGuild(guildID)
		_ = disk.Save()
		pres.OnGuildSynced()
	}

	session.Identify.Intents = discordgo.IntentsGuilds |
		discordgo.IntentsGuildMembers |
		discordgo.IntentsGuildPresences

	roleIDs := bot.RoleIDs{
		Member:  cfg.MemberRoleID,
		Bot:     cfg.BotRoleID,
		Service: cfg.IranServiceRoleID,
	}
	bot.NewHandler(cache, syncQ, roleIDs, pres.OnReady, onGuildSynced, onGuildRemoved).Register(session)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go disk.RunAutoSave(ctx, cfg.StateSaveInterval)
	go syncQ.Run(ctx)
	go workerSnapshotLoop(ctx, cache, disk, cfg.StateSaveInterval/2)
	go syncWorker.Run(ctx)
	go pres.Run(ctx)

	if invite := cfg.InviteURL(); invite != "" {
		slog.Info("stats bot invite url", "url", invite)
	}
	if cfg.DiscordApplicationID != "" {
		slog.Info("application id", "id", cfg.DiscordApplicationID)
	}
	slog.Info(
		"do not run PERSIAN-MUSIC (or any bot with the same token) alongside this process — only one gateway session per bot token",
	)

	if err := session.Open(); err != nil {
		slog.Error("discord gateway", "err", err)
		os.Exit(1)
	}
	defer session.Close()

	slog.Info("gateway connected")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	slog.Info("shutting down")
	cancel()
	flushSnapshots(cache, disk)
}

func workerSnapshotLoop(ctx context.Context, cache *guild.Cache, disk *persist.Store, interval time.Duration) {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			flushSnapshots(cache, disk)
		}
	}
}

func flushSnapshots(cache *guild.Cache, disk *persist.Store) {
	for _, snap := range cache.AllSnapshots() {
		if !snap.MembersLoaded {
			continue
		}
		disk.UpsertGuild(snap)
	}
	_ = disk.Save()
}

func setupLogger(level string) {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "info":
		lvl = slog.LevelInfo
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelWarn
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: lvl})))
}
