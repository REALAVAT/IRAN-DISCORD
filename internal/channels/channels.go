package channels

import (
	"github.com/bwmarrin/discordgo"
)

const viewChannel int64 = 1024

var textTypes = map[discordgo.ChannelType]struct{}{
	discordgo.ChannelType(0):  {},
	discordgo.ChannelType(5):  {},
	discordgo.ChannelType(15): {},
	discordgo.ChannelType(16): {},
}

var voiceTypes = map[discordgo.ChannelType]struct{}{
	discordgo.ChannelType(2):  {},
	discordgo.ChannelType(13): {},
}

func isCountable(t discordgo.ChannelType) bool {
	if _, ok := textTypes[t]; ok {
		return true
	}
	_, ok := voiceTypes[t]
	return ok
}

func applyMemberOverwrites(perms int64, overwrites []*discordgo.PermissionOverwrite, memberRoleIDs map[string]struct{}) int64 {
	for _, ow := range overwrites {
		if ow == nil || ow.Type != discordgo.PermissionOverwriteTypeRole {
			continue
		}
		if _, ok := memberRoleIDs[ow.ID]; !ok {
			continue
		}
		perms = (perms &^ ow.Deny) | ow.Allow
	}
	return perms
}

func memberBasePermissions(roles []*discordgo.Role, guildID, memberRoleID string) int64 {
	var perms int64
	for _, role := range roles {
		if role == nil {
			continue
		}
		if role.ID == guildID || role.ID == memberRoleID {
			perms |= role.Permissions
		}
	}
	return perms
}

func memberCanView(
	channel *discordgo.Channel,
	byID map[string]*discordgo.Channel,
	basePerms int64,
	memberRoleIDs map[string]struct{},
) bool {
	perms := basePerms

	var parents []*discordgo.Channel
	parentID := channel.ParentID
	for parentID != "" {
		parent, ok := byID[parentID]
		if !ok {
			break
		}
		parents = append([]*discordgo.Channel{parent}, parents...)
		parentID = parent.ParentID
	}

	for _, parent := range parents {
		perms = applyMemberOverwrites(perms, parent.PermissionOverwrites, memberRoleIDs)
	}

	perms = applyMemberOverwrites(perms, channel.PermissionOverwrites, memberRoleIDs)
	return perms&viewChannel == viewChannel
}

func CountMemberVisible(chans []*discordgo.Channel, roles []*discordgo.Role, guildID, memberRoleID string) int {
	byID := make(map[string]*discordgo.Channel, len(chans))
	for _, ch := range chans {
		if ch != nil {
			byID[ch.ID] = ch
		}
	}

	memberRoleIDs := map[string]struct{}{guildID: {}, memberRoleID: {}}
	basePerms := memberBasePermissions(roles, guildID, memberRoleID)

	total := 0
	for _, ch := range chans {
		if ch == nil || !isCountable(ch.Type) {
			continue
		}
		if !memberCanView(ch, byID, basePerms, memberRoleIDs) {
			continue
		}
		total++
	}
	return total
}
