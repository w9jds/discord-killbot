package main

import (
	"database/sql"
	"fmt"
	"killbot/pkg/esi"
	"killbot/pkg/zkb"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/text/language"
	"golang.org/x/text/message"

	_ "github.com/GoogleCloudPlatform/cloudsql-proxy/proxy/dialers/postgres"
	"github.com/bwmarrin/discordgo"
)

var (
	httpClient *http.Client
	esiClient  *esi.Client
	zkbClient  *zkb.Client
	discord    *discordgo.Session
	postgres   *sql.DB

	queueID      string
	botToken     string
	killsChannel string
	intelChannel string
	alliance     uint32
	corporation  uint32

	psqlInfo string
)

func isCorpOrAlliance(CorporationID uint32, AllianceID uint32) bool {
	if alliance != 0 && AllianceID == alliance {
		return true
	}

	if CorporationID == corporation {
		return true
	}

	return false
}

func getIds(killMail *esi.KillMail) ([]uint32, uint32, uint32, []uint32, bool, bool) {
	var finalBlowID uint32
	var hostileCorpID uint32

	isAttacker := false
	isVictim := isCorpOrAlliance(killMail.Victim.CorporationID, killMail.Victim.AllianceID)

	unique := make(map[uint32]struct{})
	unique[killMail.Victim.ID] = struct{}{}
	unique[killMail.SystemID] = struct{}{}
	unique[killMail.Victim.ShipTypeID] = struct{}{}
	unique[killMail.Victim.CorporationID] = struct{}{}

	if killMail.Victim.AllianceID != 0 {
		unique[killMail.Victim.AllianceID] = struct{}{}
	}

	friendlies := []uint32{}
	for _, attacker := range killMail.Attackers {
		isFriendly := isCorpOrAlliance(attacker.CorporationID, attacker.AllianceID)

		if isFriendly == true {
			friendlies = append(friendlies, attacker.ID)

			if isAttacker == false {
				isAttacker = true
			}
		}

		if attacker.FinalBlow == true {
			finalBlowID = attacker.ID

			if attacker.CorporationID != 0 {
				hostileCorpID = attacker.CorporationID
			}
		}

		if _, ok := unique[attacker.ID]; !ok {
			unique[attacker.ID] = struct{}{}
		}
		if _, ok := unique[attacker.ShipTypeID]; !ok {
			unique[attacker.ShipTypeID] = struct{}{}
		}
		if _, ok := unique[attacker.CorporationID]; !ok {
			unique[attacker.CorporationID] = struct{}{}
		}

		if attacker.AllianceID != 0 {
			if _, ok := unique[attacker.AllianceID]; !ok {
				unique[attacker.AllianceID] = struct{}{}
			}
		}
	}

	ids := []uint32{}
	for key := range unique {
		ids = append(ids[:], key)
	}

	return ids, finalBlowID, hostileCorpID, friendlies, isAttacker, isVictim
}

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
					Name:   "Victim",
					Value:  fmt.Sprintf("[%s](https://zkillboard.com/character/%d)", names[killMail.Victim.ID].Name, killMail.Victim.ID),
					Inline: true,
				},
				&discordgo.MessageEmbedField{
					Name:   "Corporation",
					Value:  fmt.Sprintf("[%s](https://zkillboard.com/corporation/%d)", names[hostileCorpID].Name, hostileCorpID),
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

func processKillMail(redis *zkb.RedisResponse) {
	killMail, _, error := esiClient.GetKillMail(redis.ID, redis.Zkb.Hash, false)
	if error != nil {
		log.Println("Error pulling killmail from esi: ", error)
		return
	}

	ids, finalBlowID, hostileCorpID, friendlies, isAttacker, isVictim := getIds(killMail)

	if isVictim == true || isAttacker == true {
		names, error := esiClient.GetNames(ids)
		if error != nil {
			log.Println("Error getting names from esi: ", error)
		}

		message := buildDiscordMessage(killMail, names, finalBlowID, hostileCorpID, redis.Zkb.TotalValue, friendlies, "", isAttacker, isVictim)

		sendKillMailMessage(killsChannel, message)
		return
	}

	isInChain, mapName := checkChainForSystem(killMail.SystemID)

	if isInChain {
		names, error := esiClient.GetNames(ids)
		if error != nil {
			log.Println("Error getting names from esi: ", error)
		}

		message := buildDiscordMessage(killMail, names, finalBlowID, hostileCorpID, redis.Zkb.TotalValue, friendlies, mapName, isAttacker, isVictim)

		sendKillMailMessage(intelChannel, message)
		return
	}

	log.Println(fmt.Sprintf("System %d is not in any available chains, killmail skipped.", killMail.SystemID))
}

func checkChainForSystem(ID uint32) (bool, string) {
	var mapName string

	row := postgres.QueryRow(fmt.Sprintf(
		"SELECT maps.name FROM systems JOIN maps ON maps.id = systems.map_id WHERE maps.owner_id = %d AND systems.id = %d LIMIT 1;",
		alliance,
		ID,
	))

	switch error := row.Scan(&mapName); error {
	case sql.ErrNoRows:
		return false, mapName
	case nil:
		return true, mapName
	default:
		log.Fatal(error)
		return false, mapName
	}
}

func sendKillMailMessage(channelID string, message *discordgo.MessageSend) {
	_, error := discord.ChannelMessageSendComplex(channelID, message)
	if error != nil {
		log.Println(fmt.Sprintf("Message send to %s failed: ", channelID), error)
		time.Sleep(5 * time.Second)
		sendKillMailMessage(channelID, message)
	}
}

func setupEnv() bool {
	queueID = strings.Trim(os.Getenv("REDISQ_ID"), " ")
	if queueID == "" {
		log.Println("Environment variable 'REDISQ_ID' requires a value, if you want to continue progress where you left off in the kills queue")
	}

	botToken = strings.Trim(os.Getenv("BOT_TOKEN"), " ")
	if botToken == "" {
		log.Fatal("Environment variable 'BOT_TOKEN' requires a value to be able to push messages to discord")
	}

	killsChannel = strings.Trim(os.Getenv("KILLS_CHANNEL_ID"), " ")
	intelChannel = strings.Trim(os.Getenv("INTEL_CHANNEL_ID"), " ")
	if killsChannel == "" && intelChannel == "" {
		log.Fatal("Either the `KILLS_CHANNEL_ID` or the `INTEL_CHANNEL_ID` need to be set, or the bot will do nothing")
	}
	if killsChannel == "" {
		log.Println("Environment variable 'KILLS_CHANNEL_ID' requires a value if you want a stream of kills your alliance/corp are related to")
	}
	if intelChannel == "" {
		log.Println("Environment variable 'INTEL_CHANNEL_ID' requires a value if you want to get a stream of kills that you aren't related to, and are in chain")
	}

	host := strings.Trim(os.Getenv("POSTGRES_HOST"), " ")
	if host == "" {
		log.Fatal("Environment variable `POSTGRES_HOST` is required to connect to the systems database")
	}

	user := strings.Trim(os.Getenv("POSTGRES_USER"), " ")
	if user == "" {
		log.Fatal("Environment variable `POSTGRES_USER` is required to connect to the systems database")
	}

	dbname := strings.Trim(os.Getenv("POSTGRES_DB"), " ")
	if dbname == "" {
		log.Fatal("Environment variable `POSTGRES_DB` is required to connect to the systems database")
	}

	password := strings.Trim(os.Getenv("POSTGRES_PASSWORD"), " ")
	if password == "" {
		log.Fatal("Environment variable `POSTGRES_PASSWORD` is required to connect to the systems database")
	}

	psqlInfo = fmt.Sprintf("host=%s dbname=%s user=%s password=%s sslmode=disable",
		host, dbname, user, password)

	allianceID, error := strconv.ParseUint(os.Getenv("ALLIANCE_ID"), 10, 32)
	if error != nil {
		log.Fatal("Environment variable 'ALLIANCE_ID' wasn't set correctly (or at all)")
	}

	corporationID, error := strconv.ParseUint(os.Getenv("CORPORATION_ID"), 10, 32)
	if error != nil {
		log.Fatal("Environment Variable 'CORPORATION_ID' wasn't set correctly (or at all)")
	}

	alliance = uint32(allianceID)
	corporation = uint32(corporationID)

	return true
}

func main() {
	var error error
	isReady := setupEnv()

	if isReady == true {
		httpClient = &http.Client{}
		esiClient = esi.CreateClient(httpClient)
		zkbClient = zkb.CreateClient(httpClient)

		postgres, error = sql.Open("cloudsqlpostgres", psqlInfo)
		if error != nil {
			log.Fatal(error)
		}

		defer postgres.Close()

		discord, error = discordgo.New("Bot " + botToken)
		if error != nil {
			log.Fatal("discordgo: ", error)
		}

		error = discord.Open()
		if error != nil {
			log.Fatal("discordgo: ", error)
		}

		defer discord.Close()

		getKillmails()
	}

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc
}

func getKillmails() {

	log.Println("Concord Alerts have started processing!")

	for {
		bundle, error := zkbClient.GetRedisItem(queueID)
		if error != nil || bundle.ID == 0 {
			time.Sleep(5 * time.Second)
			continue
		}

		go processKillMail(bundle)
	}
}
