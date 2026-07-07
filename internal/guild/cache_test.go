package guild

import (
	"fmt"
	"testing"

	"github.com/bwmarrin/discordgo"
	"github.com/iran-discord-services/stats/internal/persist"
)

func TestHumansOnline_OnlyConfirmedHumansWithClientPlatform(t *testing.T) {
	c := NewCache()
	guildID := "test-guild"

	c.BeginMemberSync(guildID, 1)
	c.RegisterMember(guildID, "you", false, 0)
	for i := range 49 {
		c.RegisterMember(guildID, fmt.Sprintf("bot-%d", i), true, 0)
	}
	c.MembersChunkReceived(guildID)

	stats := c.RegistryStats(guildID)
	if stats.Humans != 1 || stats.Bots != 49 {
		t.Fatalf("registry humans=%d bots=%d, want 1/49", stats.Humans, stats.Bots)
	}

	c.UpdatePresence(guildID, "you", discordgo.StatusInvisible, discordgo.ClientStatus{
		Desktop: discordgo.StatusOnline,
	})
	if got := c.HumansOnline(guildID); got != 0 {
		t.Fatalf("invisible human: want 0 got %d", got)
	}

	c.UpdatePresence(guildID, "you", discordgo.StatusOnline, discordgo.ClientStatus{
		Desktop: discordgo.StatusOnline,
	})
	if got := c.HumansOnline(guildID); got != 1 {
		t.Fatalf("online human: want 1 got %d", got)
	}
}

func TestHumansOnline_BeforeMemberRegistryLoaded(t *testing.T) {
	c := NewCache()
	c.BeginMemberSync("g", 2)
	c.RegisterMember("g", "you", false, 0)
	c.UpdatePresence("g", "you", discordgo.StatusOnline, discordgo.ClientStatus{
		Desktop: discordgo.StatusOnline,
	})
	if got := c.HumansOnline("g"); got != 0 {
		t.Fatalf("before registry loaded: want 0 got %d", got)
	}
}

func TestGlobalStatsHybrid_UsesSavedUntilLiveReady(t *testing.T) {
	c := NewCache()
	c.EnsureGuild("a")
	c.EnsureGuild("b")

	saved := map[string]persist.GuildSnapshot{
		"a": {GuildID: "a", HumansOnline: 10, MembersLoaded: true},
		"b": {GuildID: "b", HumansOnline: 20, MembersLoaded: true},
	}

	stats := c.GlobalStatsHybrid(saved)
	if stats.Guilds != 2 || stats.HumansOnline != 30 {
		t.Fatalf("hybrid before live = %+v, want 2 guilds / 30 online", stats)
	}

	c.BeginMemberSync("a", 1)
	c.RegisterMember("a", "live", false, 0)
	c.MembersChunkReceived("a")
	c.UpdatePresence("a", "live", discordgo.StatusOnline, discordgo.ClientStatus{
		Desktop: discordgo.StatusOnline,
	})

	stats = c.GlobalStatsHybrid(saved)
	if stats.GuildsReady != 1 {
		t.Fatalf("guilds ready = %d, want 1", stats.GuildsReady)
	}
	if stats.HumansOnline != 21 {
		t.Fatalf("hybrid after partial live = %+v, want 21 online", stats)
	}
}

func TestPresenceStatsHybrid_UsesSavedHumansWhileLiveReloads(t *testing.T) {
	c := NewCache()
	c.EnsureGuild("a")

	saved := map[string]persist.GuildSnapshot{
		"a": {GuildID: "a", Humans: 500, MembersLoaded: true},
	}

	stats := c.PresenceStatsHybrid(saved)
	if stats.TotalUsers != 500 {
		t.Fatalf("hybrid before reload = %+v, want 500 members", stats)
	}

	c.BeginMemberSync("a", 1)
	stats = c.PresenceStatsHybrid(saved)
	if stats.TotalUsers != 500 {
		t.Fatalf("hybrid during reload = %+v, want saved 500 until live ready", stats)
	}
}
