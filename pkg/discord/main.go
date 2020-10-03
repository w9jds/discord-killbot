package discord

import (
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
)

// Killbot is a bot configured to push kills to specific channels
type Killbot struct {
	discord      *discordgo.Session
	killsChannel string
	intelChannel string
}

// CreateDiscordBot builds a new KillBot with alliance/corporations configured
func CreateDiscordBot(botToken string, killChannelID string, intelChannelID string) *Killbot {
	var discord, err = discordgo.New("Bot " + botToken)
	if err != nil {
		log.Fatal("discordgo: ", err)
	}

	err = discord.Open()
	if err != nil {
		log.Fatal("discordgo: ", err)
	}

	defer discord.Close()

	return &Killbot{
		discord:      discord,
		killsChannel: killChannelID,
		intelChannel: intelChannelID,
	}
}

func (bot *Killbot) buildDiscordMessage(details *MessageDetails) *discordgo.MessageSend {
	message := &discordgo.MessageSend{
		Embed: &discordgo.MessageEmbed{
			Title:     details.getMessageTitle(),
			URL:       fmt.Sprintf("https://zkillboard.com/kill/%d", details.KillMail.ID),
			Timestamp: details.KillMail.Time,
			Color:     details.getMessageColor(),
			Thumbnail: &discordgo.MessageEmbedThumbnail{
				URL: fmt.Sprintf(
					"https://imageserver.eveonline.com/Render/%d_128.png",
					details.KillMail.Victim.ShipTypeID,
				),
			},
			Fields: details.buildEmbedFields(),
		},
	}

	if details.TotalValue > 3000000000 && len(details.Friendlies) > 3 {
		message.Content = "WOAH! Look at that kill @everyone"
	}

	return message
}

// SendKillMessage pushes a discord message to the kills channel
func (killbot *Killbot) SendKillMessage(message *discordgo.MessageSend) {
	killbot.sendMessage(killbot.killsChannel, message)
}

// SendIntelMessage pushes a discord message to the intel channel
func (killbot *Killbot) SendIntelMessage(message *discordgo.MessageSend) {
	killbot.sendMessage(killbot.intelChannel, message)
}

func (killbot *Killbot) sendMessage(channelID string, message *discordgo.MessageSend) {
	_, err := killbot.discord.ChannelMessageSendComplex(channelID, message)

	if err != nil {
		errMessage := fmt.Sprintf("Message send to %s failed: ", killbot.intelChannel)
		log.Println(errMessage, err)

		time.Sleep(5 * time.Second)
		killbot.sendMessage(channelID, message)
	}
}
