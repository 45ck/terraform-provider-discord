package discord

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
