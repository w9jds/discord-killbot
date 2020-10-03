package discord

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	esi "github.com/w9jds/go.esi"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// MessageDetails all the details associated with the message being built
type MessageDetails struct {
	Names         map[uint32]esi.NameRef
	KillMail      *esi.KillMail
	FinalBlowID   uint32
	HostileCorpID uint32
	TotalValue    float64
	Friendlies    []uint32
	MapName       string
	IsAttacker    bool
	IsVictim      bool
}

func (details *MessageDetails) getMessageColor() int {
	color := 6710886

	if details.IsAttacker == true {
		color = 8103679
	}

	if details.IsVictim == true {
		color = 16711680
	}

	if details.IsVictim == true && details.IsAttacker == true {
		color = 6570404
	}

	return color
}

func (details *MessageDetails) getMessageTitle() string {
	var title string
	victimID := details.KillMail.Victim.ID
	victimCorpID := details.KillMail.Victim.CorporationID

	if !details.IsVictim && !details.IsAttacker {
		hostileCorpID := details.HostileCorpID
		shipID := details.KillMail.Victim.ShipTypeID

		title = fmt.Sprintf(
			"%s lost a %s to %s",
			details.Names[victimCorpID].Name,
			details.Names[shipID].Name,
			details.Names[hostileCorpID].Name,
		)
	}

	title = fmt.Sprintf(
		"%s killed %s (%s)",
		details.Names[details.FinalBlowID].Name,
		details.Names[victimID].Name,
		details.Names[victimCorpID].Name,
	)

	if len(title) > 256 {
		title = title[:253] + "..."
	}

	return title
}

func (details *MessageDetails) buildEmbedFields() []*discordgo.MessageEmbedField {
	printer := message.NewPrinter(language.English)

	return []*discordgo.MessageEmbedField{
		&discordgo.MessageEmbedField{
			Name: "Killing Blow",
			Value: fmt.Sprintf(
				"[%s](https://zkillboard.com/character/%d)",
				details.Names[details.FinalBlowID].Name,
				details.FinalBlowID,
			),
			Inline: false,
		},
		&discordgo.MessageEmbedField{
			Name: "Corporation",
			Value: fmt.Sprintf(
				"[%s](https://zkillboard.com/corporation/%d)",
				details.Names[details.HostileCorpID].Name,
				details.HostileCorpID,
			),
			Inline: true,
		},
		&discordgo.MessageEmbedField{
			Name: "Victim",
			Value: fmt.Sprintf(
				"[%s](https://zkillboard.com/character/%d)",
				details.Names[details.KillMail.Victim.ID].Name,
				details.KillMail.Victim.ID,
			),
			Inline: false,
		},
		&discordgo.MessageEmbedField{
			Name: "Corporation",
			Value: fmt.Sprintf(
				"[%s](https://zkillboard.com/corporation/%d)",
				details.Names[details.KillMail.Victim.CorporationID].Name,
				details.KillMail.Victim.CorporationID,
			),
			Inline: true,
		},
		&discordgo.MessageEmbedField{
			Name: "Ship",
			Value: fmt.Sprintf(
				"[%s](https://zkillboard.com/ship/%d)",
				details.Names[details.KillMail.Victim.ShipTypeID].Name,
				details.KillMail.Victim.ShipTypeID,
			),
			Inline: true,
		},
		&discordgo.MessageEmbedField{
			Name: "System",
			Value: fmt.Sprintf(
				"[%s](https://zkillboard.com/system/%d)",
				details.Names[details.KillMail.SystemID].Name,
				details.KillMail.SystemID,
			),
			Inline: false,
		},
		&discordgo.MessageEmbedField{
			Name: "Pilots Involved",
			Value: fmt.Sprintf(
				"%d",
				len(details.KillMail.Attackers),
			),
			Inline: false,
		},
		&discordgo.MessageEmbedField{
			Name: "Value",
			Value: printer.Sprintf(
				"%.2f ISK",
				details.TotalValue,
			),
			Inline: true,
		},
	}
}

func (details *MessageDetails) buildMapLocation() *discordgo.MessageEmbedField {
	if details.MapName != "" {
		return &discordgo.MessageEmbedField{
			Name:   "Map Name",
			Value:  details.MapName,
			Inline: false,
		}
	}

	return nil
}

func (details *MessageDetails) buildInvolvedPiolets() *discordgo.MessageEmbedField {
	if len(details.Friendlies) > 0 {
		var members []string
		var participants string

		for _, id := range details.Friendlies {
			members = append(members, fmt.Sprintf(
				"[%s](https://zkillboard.com/character/%d/)",
				details.Names[id].Name,
				id,
			))
		}

		for {
			participants = strings.Join(members, ", ")

			if len(participants) <= 1024 {
				break
			}

			members = members[:len(members)-1]
		}

		return &discordgo.MessageEmbedField{
			Name:   "Friendly Pilots Involved",
			Value:  participants,
			Inline: false,
		}
	}

	return nil
}
