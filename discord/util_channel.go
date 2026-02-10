package discord

import (
	"context"
	"github.com/andersfylling/disgord"
	"github.com/bwmarrin/discordgo"
)

func getTextChannelType(channelType uint) (string, bool) {
	switch channelType {
	case 0:
		return "text", true
	case 2:
		return "voice", true
	case 4:
		return "category", true
	case 5:
		return "news", true
	case 6:
		return "store", true
	case 10:
		return "announcement_thread", true
	case 11:
		return "public_thread", true
	case 12:
		return "private_thread", true
	case 13:
		return "stage", true
	case 15:
		return "forum", true
	case 16:
		return "media", true
	}

	return "text", false
}

func getDiscordChannelType(name string) (uint, bool) {
	switch name {
	case "text":
		return 0, true
	case "voice":
		return 2, true
	case "category":
		return 4, true
	case "news":
		return 5, true
	case "store":
		return 6, true
	case "announcement_thread":
		return 10, true
	case "public_thread":
		return 11, true
	case "private_thread":
		return 12, true
	case "stage":
		return 13, true
	case "forum":
		return 15, true
	case "media":
		return 16, true
	}

	return 0, false
}

type Channel struct {
	ServerId  string
	ChannelId string
	Channel   *discordgo.Channel
}

func findChannelById(array []*disgord.Channel, id disgord.Snowflake) *disgord.Channel {
	for _, element := range array {
		if element.ID == id {
			return element
		}
	}

	return nil
}

func arePermissionsSynced(from *disgord.Channel, to *disgord.Channel) bool {
	for _, p1 := range from.PermissionOverwrites {
		cont := false
		for _, p2 := range to.PermissionOverwrites {
			if p1.ID == p2.ID && p1.Type == p2.Type && p1.Allow == p2.Allow && p1.Deny == p2.Deny {
				cont = true
				break
			}
		}
		if !cont {
			return false
		}
	}

	for _, p1 := range to.PermissionOverwrites {
		cont := false
		for _, p2 := range from.PermissionOverwrites {
			if p1.ID == p2.ID && p1.Type == p2.Type && p1.Allow == p2.Allow && p1.Deny == p2.Deny {
				cont = true
				break
			}
		}
		if !cont {
			return false
		}
	}

	return true
}

func syncChannelPermissions(c *disgord.Client, ctx context.Context, from *disgord.Channel, to *disgord.Channel) error {
	for _, p := range to.PermissionOverwrites {
		if err := c.DeleteChannelPermission(ctx, to.ID, p.ID); err != nil {
			return err
		}
	}

	for _, p := range from.PermissionOverwrites {
		params := &disgord.UpdateChannelPermissionsParams{
			Allow: p.Allow,
			Deny:  p.Deny,
			Type:  p.Type,
		}
		if err := c.UpdateChannelPermissions(ctx, to.ID, p.ID, params); err != nil {
			return err
		}
	}

	return nil
}
