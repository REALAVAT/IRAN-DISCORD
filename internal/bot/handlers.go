package bot

import (
	"log/slog"

	"github.com/bwmarrin/discordgo"
	"github.com/iran-discord-services/stats/internal/guild"
	"github.com/iran-discord-services/stats/internal/membersync"
)

type RoleIDs struct {
	Member  string
	Bot     string
	Service string
}

type Handler struct {
	cache          *guild.Cache
	syncQ          *membersync.Queue
	roleIDs        RoleIDs
	presenceReady  func()
	onGuildSynced  func(guildID string)
	onGuildRemoved func(guildID string)
}

func NewHandler(
	cache *guild.Cache,
	syncQ *membersync.Queue,
	roleIDs RoleIDs,
	presenceReady func(),
	onGuildSynced func(string),
	onGuildRemoved func(string),
) *Handler {
	return &Handler{
		cache:          cache,
		syncQ:          syncQ,
		roleIDs:        roleIDs,
		presenceReady:  presenceReady,
		onGuildSynced:  onGuildSynced,
		onGuildRemoved: onGuildRemoved,
	}
}

func (h *Handler) roleFlags(roles []string) uint8 {
	var flags uint8
	for _, id := range roles {
		switch id {
		case h.roleIDs.Member:
			flags |= guild.FlagMemberRole
		case h.roleIDs.Bot:
			flags |= guild.FlagBotRole
		case h.roleIDs.Service:
			flags |= guild.FlagServiceRole
		}
	}
	return flags
}

func (h *Handler) Register(session *discordgo.Session) {
	session.AddHandler(h.onReady)
	session.AddHandler(h.onGuildCreate)
	session.AddHandler(h.onGuildDelete)
	session.AddHandler(h.onGuildMembersChunk)
	session.AddHandler(h.onGuildMemberAdd)
	session.AddHandler(h.onGuildMemberRemove)
	session.AddHandler(h.onGuildMemberUpdate)
	session.AddHandler(h.onPresenceUpdate)
}

func (h *Handler) onReady(s *discordgo.Session, event *discordgo.Ready) {
	slog.Info("gateway ready", "guilds", len(event.Guilds), "user", event.User.Username)

	current := make(map[string]struct{}, len(event.Guilds))
	ids := make([]string, 0, len(event.Guilds))
	for _, g := range event.Guilds {
		h.cache.EnsureGuild(g.ID)
		current[g.ID] = struct{}{}
		ids = append(ids, g.ID)
	}

	for _, id := range h.cache.GuildIDs() {
		if _, ok := current[id]; ok {
			continue
		}
		h.cache.RemoveGuild(id)
		if h.onGuildRemoved != nil {
			h.onGuildRemoved(id)
		}
		slog.Info("pruned stale guild from state", "id", id)
	}

	h.syncQ.EnqueueAll(ids)

	if h.presenceReady != nil {
		h.presenceReady()
	}
}

func (h *Handler) onGuildCreate(_ *discordgo.Session, event *discordgo.GuildCreate) {
	if event.Guild == nil {
		return
	}
	h.cache.EnsureGuild(event.Guild.ID)
	slog.Debug("joined guild", "guild", event.Guild.Name, "id", event.Guild.ID)
	h.syncQ.Enqueue(event.Guild.ID)
}

func (h *Handler) onGuildMembersChunk(_ *discordgo.Session, event *discordgo.GuildMembersChunk) {
	if event.ChunkCount > 0 && event.ChunkIndex == 0 {
		h.cache.BeginMemberSync(event.GuildID, event.ChunkCount)
	}

	for _, member := range event.Members {
		if member.User == nil {
			continue
		}
		h.cache.RegisterMember(event.GuildID, member.User.ID, member.User.Bot, h.roleFlags(member.Roles))
	}

	for _, presence := range event.Presences {
		if presence.User == nil {
			continue
		}
		h.cache.UpdatePresence(event.GuildID, presence.User.ID, presence.Status, presence.ClientStatus)
	}

	h.cache.MembersChunkReceived(event.GuildID)

	if h.cache.MembersLoaded(event.GuildID) {
		stats := h.cache.RegistryStats(event.GuildID)
		slog.Info(
			"guild registry ready",
			"guild", event.GuildID,
			"humans", stats.Humans,
			"bots", stats.Bots,
			"humans_online", stats.OnlineHumans,
		)
		if h.onGuildSynced != nil {
			h.onGuildSynced(event.GuildID)
		}
	}
}

func (h *Handler) onGuildDelete(_ *discordgo.Session, event *discordgo.GuildDelete) {
	if event.Guild == nil {
		return
	}
	if event.Guild.Unavailable {
		return
	}
	h.cache.RemoveGuild(event.Guild.ID)
	if h.onGuildRemoved != nil {
		h.onGuildRemoved(event.Guild.ID)
	}
	slog.Info("left guild", "id", event.Guild.ID)
}

func (h *Handler) onGuildMemberAdd(_ *discordgo.Session, event *discordgo.GuildMemberAdd) {
	if event.Member == nil || event.Member.User == nil {
		return
	}
	h.cache.RegisterMember(event.GuildID, event.Member.User.ID, event.Member.User.Bot, h.roleFlags(event.Member.Roles))
}

func (h *Handler) onGuildMemberRemove(_ *discordgo.Session, event *discordgo.GuildMemberRemove) {
	if event.User == nil {
		return
	}
	h.cache.RemoveMember(event.GuildID, event.User.ID)
}

func (h *Handler) onGuildMemberUpdate(_ *discordgo.Session, event *discordgo.GuildMemberUpdate) {
	if event.Member == nil || event.Member.User == nil {
		return
	}
	h.cache.RegisterMember(event.GuildID, event.Member.User.ID, event.Member.User.Bot, h.roleFlags(event.Member.Roles))
}

func (h *Handler) onPresenceUpdate(_ *discordgo.Session, event *discordgo.PresenceUpdate) {
	if event.User == nil {
		return
	}
	if !h.cache.MembersLoaded(event.GuildID) {
		return
	}
	h.cache.UpdatePresence(event.GuildID, event.User.ID, event.Status, event.ClientStatus)
}
