package main

import (
	"fmt"
	"killbot/pkg/esi"
	"log"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

func buildDiscordMessage(killMail *esi.KillMail, names map[uint32]esi.NameRef, finalBlowID uint32, hostileCorpID uint32, totalValue float64, friendlies []uint32, mapName string, isAttacker bool, isVictim bool) *discordgo.MessageSend {
	var title string
	printer := message.NewPrinter(language.English)

	if !isVictim && !isAttacker {
		title = fmt.Sprintf("%s lost a %s to %s", names[killMail.Victim.CorporationID].Name, names[killMail.Victim.ShipTypeID].Name, names[hostileCorpID].Name)
	} else {
		title = fmt.Sprintf("%s killed %s (%s)", names[finalBlowID].Name, names[killMail.Victim.ID].Name, names[killMail.Victim.CorporationID].Name)
	}

	if len(title) > 256 {
		title = title[:253] + "..."
	}

	message := &discordgo.MessageSend{
		Embed: &discordgo.MessageEmbed{
			Title:     title,
			URL:       fmt.Sprintf("https://zkillboard.com/kill/%d", killMail.ID),
			Timestamp: killMail.Time,
			Color:     6710886,
			Thumbnail: &discordgo.MessageEmbedThumbnail{
				URL: fmt.Sprintf("https://imageserver.eveonline.com/Render/%d_128.png", killMail.Victim.ShipTypeID),
			},
			Fields: []*discordgo.MessageEmbedField{
				&discordgo.MessageEmbedField{
					Name:   "Killing Blow",
					Value:  fmt.Sprintf("[%s](https://zkillboard.com/character/%d)", names[finalBlowID].Name, finalBlowID),
					Inline: true,
				},
				&discordgo.MessageEmbedField{
					Name:   "Corporation",
					Value:  fmt.Sprintf("[%s](https://zkillboard.com/corporation/%d)", names[hostileCorpID].Name, hostileCorpID),
					Inline: true,
				},
				&discordgo.MessageEmbedField{
					Name:   "Victim",
					Value:  fmt.Sprintf("[%s](https://zkillboard.com/character/%d)", names[killMail.Victim.ID].Name, killMail.Victim.ID),
					Inline: true,
				},
				&discordgo.MessageEmbedField{
					Name:   "Corporation",
					Value:  fmt.Sprintf("[%s](https://zkillboard.com/corporation/%d)", names[killMail.Victim.CorporationID].Name, killMail.Victim.CorporationID),
					Inline: true,
				},
				&discordgo.MessageEmbedField{
					Name:   "Ship",
					Value:  fmt.Sprintf("[%s](https://zkillboard.com/ship/%d)", names[killMail.Victim.ShipTypeID].Name, killMail.Victim.ShipTypeID),
					Inline: true,
				},
				&discordgo.MessageEmbedField{
					Name:   "System",
					Value:  fmt.Sprintf("[%s](https://zkillboard.com/system/%d)", names[killMail.SystemID].Name, killMail.SystemID),
					Inline: true,
				},
				&discordgo.MessageEmbedField{
					Name:   "Pilots Involved",
					Value:  fmt.Sprintf("%d", len(killMail.Attackers)),
					Inline: true,
				},
				&discordgo.MessageEmbedField{
					Name:   "Value",
					Value:  printer.Sprintf("%.2f ISK", totalValue),
					Inline: true,
				},
			},
		},
	}

	if isAttacker == true {
		message.Embed.Color = 8103679
	}

	if isVictim == true {
		message.Embed.Color = 16711680
	}

	if isVictim == true && isAttacker == true {
		message.Embed.Color = 6570404
	}

	if mapName != "" {
		message.Embed.Fields = append(message.Embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Map Name",
			Value:  mapName,
			Inline: false,
		})
	}

	if len(friendlies) > 0 {
		var members []string
		var participants string

		for _, id := range friendlies {
			members = append(members, fmt.Sprintf("[%s](https://zkillboard.com/character/%d/)", names[id].Name, id))
		}

		for {
			participants = strings.Join(members, ", ")

			if len(participants) <= 1024 {
				break
			}

			members = members[:len(members)-1]
		}

		message.Embed.Fields = append(message.Embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Friendly Pilots Involved",
			Value:  participants,
			Inline: false,
		})
	}

	if totalValue > 3000000000 && len(friendlies) > 3 {
		message.Content = "WOAH! Look at that kill @everyone"
	}

	return message
}

func sendKillMailMessage(channelID string, message *discordgo.MessageSend) {
	_, error := discord.ChannelMessageSendComplex(channelID, message)
	if error != nil {
		log.Println(fmt.Sprintf("Message send to %s failed: ", channelID), error)
		time.Sleep(5 * time.Second)
		sendKillMailMessage(channelID, message)
	}
}
