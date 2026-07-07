package guild

import (
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/iran-discord-services/stats/internal/persist"
)

var onlineStatuses = map[discordgo.Status]struct{}{
	discordgo.StatusOnline:       {},
	discordgo.StatusIdle:         {},
	discordgo.StatusDoNotDisturb: {},
}

const (
	FlagMemberRole uint8 = 1 << iota
	FlagBotRole
	FlagServiceRole
)

type guildState struct {
	humans        map[string]struct{}
	bots          map[string]struct{}
	status        map[string]discordgo.Status
	roleFlags     map[string]uint8
	membersLoaded bool
	pendingChunks int
}

type Cache struct {
	mu     sync.RWMutex
	guilds map[string]*guildState
}

func NewCache() *Cache {
	return &Cache{guilds: make(map[string]*guildState)}
}

func (c *Cache) EnsureGuild(guildID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.guilds[guildID]; !ok {
		c.guilds[guildID] = newGuildState()
	}
}

func newGuildState() *guildState {
	return &guildState{
		humans:    make(map[string]struct{}),
		bots:      make(map[string]struct{}),
		status:    make(map[string]discordgo.Status),
		roleFlags: make(map[string]uint8),
	}
}

func (c *Cache) RemoveGuild(guildID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.guilds, guildID)
}

func (c *Cache) BeginMemberSync(guildID string, chunkCount int) {
	c.EnsureGuild(guildID)
	c.mu.Lock()
	defer c.mu.Unlock()
	state := c.guilds[guildID]
	state.humans = make(map[string]struct{})
	state.bots = make(map[string]struct{})
	state.status = make(map[string]discordgo.Status)
	state.roleFlags = make(map[string]uint8)
	state.membersLoaded = false
	if chunkCount <= 0 {
		state.pendingChunks = 1
		return
	}
	state.pendingChunks = chunkCount
}

func (c *Cache) RegisterMember(guildID, userID string, isBot bool, roleFlags uint8) {
	c.EnsureGuild(guildID)
	c.mu.Lock()
	defer c.mu.Unlock()
	state := c.guilds[guildID]
	delete(state.status, userID)
	if roleFlags != 0 {
		state.roleFlags[userID] = roleFlags
	} else {
		delete(state.roleFlags, userID)
	}
	if isBot {
		delete(state.humans, userID)
		state.bots[userID] = struct{}{}
		return
	}
	delete(state.bots, userID)
	state.humans[userID] = struct{}{}
}

func (c *Cache) MembersChunkReceived(guildID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	state, ok := c.guilds[guildID]
	if !ok {
		return
	}
	if state.pendingChunks > 0 {
		state.pendingChunks--
	}
	if state.pendingChunks <= 0 {
		state.membersLoaded = true
		state.pendingChunks = 0
	}
}

func (c *Cache) MembersLoaded(guildID string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	state, ok := c.guilds[guildID]
	return ok && state.membersLoaded
}

func (c *Cache) IsBot(guildID, userID string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	state, ok := c.guilds[guildID]
	if !ok {
		return false
	}
	_, ok = state.bots[userID]
	return ok
}

func (c *Cache) IsHuman(guildID, userID string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	state, ok := c.guilds[guildID]
	if !ok {
		return false
	}
	_, ok = state.humans[userID]
	return ok
}

func (c *Cache) RemoveMember(guildID, userID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	state, ok := c.guilds[guildID]
	if !ok {
		return
	}
	delete(state.humans, userID)
	delete(state.bots, userID)
	delete(state.status, userID)
	delete(state.roleFlags, userID)
}

func humanPresenceSignal(status discordgo.Status, client discordgo.ClientStatus) bool {
	if status == discordgo.StatusOffline || status == discordgo.StatusInvisible {
		return false
	}
	if _, ok := onlineStatuses[status]; !ok {
		return false
	}
	return clientPlatformOnline(client)
}

func clientPlatformOnline(client discordgo.ClientStatus) bool {
	for _, platformStatus := range []discordgo.Status{client.Desktop, client.Mobile, client.Web} {
		if _, ok := onlineStatuses[platformStatus]; ok {
			return true
		}
	}
	return false
}

func (c *Cache) UpdatePresence(guildID, userID string, status discordgo.Status, client discordgo.ClientStatus) {
	c.mu.Lock()
	defer c.mu.Unlock()
	state, ok := c.guilds[guildID]
	if !ok {
		return
	}

	_, isHuman := state.humans[userID]
	_, isTrackedBot := state.bots[userID]
	hasBotRole := state.roleFlags[userID]&FlagBotRole != 0
	if !isHuman && !isTrackedBot && !hasBotRole {
		return
	}

	if isHuman {
		if !humanPresenceSignal(status, client) {
			delete(state.status, userID)
			return
		}
	} else if _, online := onlineStatuses[status]; !online {
		delete(state.status, userID)
		return
	}
	state.status[userID] = status
}

func (c *Cache) HumansOnline(guildID string) int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	state, ok := c.guilds[guildID]
	if !ok || !state.membersLoaded {
		return 0
	}

	count := 0
	for userID := range state.humans {
		status, tracked := state.status[userID]
		if !tracked {
			continue
		}
		if _, online := onlineStatuses[status]; !online {
			continue
		}
		count++
	}
	return count
}

func (c *Cache) GuildIDs() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	ids := make([]string, 0, len(c.guilds))
	for id := range c.guilds {
		ids = append(ids, id)
	}
	return ids
}

type RegistryStats struct {
	GuildID       string
	Humans        int
	Bots          int
	OnlineHumans  int
	MembersLoaded bool
}

func (c *Cache) RegistryStats(guildID string) RegistryStats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	state, ok := c.guilds[guildID]
	if !ok {
		return RegistryStats{GuildID: guildID}
	}

	online := 0
	for userID := range state.humans {
		if status, tracked := state.status[userID]; tracked {
			if _, ok := onlineStatuses[status]; ok {
				online++
			}
		}
	}

	return RegistryStats{
		GuildID:       guildID,
		Humans:        len(state.humans),
		Bots:          len(state.bots),
		OnlineHumans:  online,
		MembersLoaded: state.membersLoaded,
	}
}

type MainGuildTallies struct {
	RealUsers  int
	BotsTotal  int
	BotsOnline int
	Ready      bool
}

func (c *Cache) TalliesFor(guildID string) MainGuildTallies {
	c.mu.RLock()
	defer c.mu.RUnlock()
	state, ok := c.guilds[guildID]
	if !ok || !state.membersLoaded {
		return MainGuildTallies{}
	}

	tallies := MainGuildTallies{Ready: true}
	for userID, flags := range state.roleFlags {
		isServerBot := flags&(FlagBotRole|FlagServiceRole) != 0
		if isServerBot {
			tallies.BotsTotal++
		}
		if flags&FlagMemberRole != 0 && !isServerBot {
			tallies.RealUsers++
		}

		if flags&FlagBotRole == 0 {
			continue
		}
		if status, tracked := state.status[userID]; tracked {
			if _, online := onlineStatuses[status]; online {
				tallies.BotsOnline++
			}
		}
	}
	return tallies
}

type GlobalStats struct {
	Guilds       int
	GuildsReady  int
	HumansOnline int
}

func (c *Cache) GlobalStats() GlobalStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := GlobalStats{Guilds: len(c.guilds)}
	for _, state := range c.guilds {
		if !state.membersLoaded {
			continue
		}
		stats.GuildsReady++
		for userID := range state.humans {
			status, tracked := state.status[userID]
			if !tracked {
				continue
			}
			if _, online := onlineStatuses[status]; online {
				stats.HumansOnline++
			}
		}
	}
	return stats
}

func (c *Cache) GlobalStatsHybrid(saved map[string]persist.GuildSnapshot) GlobalStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := GlobalStats{Guilds: len(c.guilds)}
	if stats.Guilds == 0 {
		stats.Guilds = len(saved)
	}

	seen := make(map[string]struct{}, len(c.guilds))
	for guildID, state := range c.guilds {
		seen[guildID] = struct{}{}
		if state.membersLoaded {
			stats.GuildsReady++
			for userID := range state.humans {
				status, tracked := state.status[userID]
				if !tracked {
					continue
				}
				if _, online := onlineStatuses[status]; online {
					stats.HumansOnline++
				}
			}
			continue
		}
		if snap, ok := saved[guildID]; ok && snap.MembersLoaded {
			stats.HumansOnline += snap.HumansOnline
		}
	}

	for guildID, snap := range saved {
		if _, ok := seen[guildID]; ok {
			continue
		}
		if snap.MembersLoaded {
			stats.GuildsReady++
			stats.HumansOnline += snap.HumansOnline
		}
	}
	return stats
}

func (c *Cache) Snapshot(guildID string) persist.GuildSnapshot {
	stats := c.RegistryStats(guildID)
	return persist.GuildSnapshot{
		GuildID:       guildID,
		Humans:        stats.Humans,
		Bots:          stats.Bots,
		HumansOnline:  stats.OnlineHumans,
		MembersLoaded: stats.MembersLoaded,
		UpdatedAt:     time.Now().UTC(),
	}
}

func (c *Cache) AllSnapshots() []persist.GuildSnapshot {
	ids := c.GuildIDs()
	out := make([]persist.GuildSnapshot, 0, len(ids))
	for _, id := range ids {
		out = append(out, c.Snapshot(id))
	}
	return out
}

type PresenceStats struct {
	Servers    int
	TotalUsers int
}

func (c *Cache) PresenceStats() PresenceStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := PresenceStats{Servers: len(c.guilds)}
	for _, state := range c.guilds {
		if !state.membersLoaded {
			continue
		}
		stats.TotalUsers += len(state.humans)
	}
	return stats
}

func (c *Cache) PresenceStatsHybrid(saved map[string]persist.GuildSnapshot) PresenceStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := PresenceStats{Servers: len(c.guilds)}
	if stats.Servers == 0 {
		stats.Servers = len(saved)
	}

	seen := make(map[string]struct{}, len(c.guilds))
	for guildID, state := range c.guilds {
		seen[guildID] = struct{}{}
		if state.membersLoaded {
			stats.TotalUsers += len(state.humans)
			continue
		}
		if snap, ok := saved[guildID]; ok && snap.Humans > 0 {
			stats.TotalUsers += snap.Humans
		}
	}

	for guildID, snap := range saved {
		if _, ok := seen[guildID]; ok {
			continue
		}
		if snap.Humans > 0 {
			stats.TotalUsers += snap.Humans
		}
	}
	return stats
}
