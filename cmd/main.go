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

	_ "github.com/lib/pq"
	"golang.org/x/text/language"
	"golang.org/x/text/message"

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

func getIds(killMail *esi.KillMail) ([]uint32, uint32, []uint32, bool, bool) {
	var finalBlowID uint32

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

	return ids, finalBlowID, friendlies, isAttacker, isVictim
}

func buildDiscordMessage(killMail *esi.KillMail, names map[uint32]esi.NameRef, finalBlowID uint32, totalValue float64, friendlies []uint32, isAttacker bool, isVictim bool) *discordgo.MessageSend {
	printer := message.NewPrinter(language.English)

	message := &discordgo.MessageSend{
		Embed: &discordgo.MessageEmbed{
			Title:     fmt.Sprintf("%s killed %s (%s)", names[finalBlowID].Name, names[killMail.Victim.ID].Name, names[killMail.Victim.CorporationID].Name),
			URL:       fmt.Sprintf("https://zkillboard.com/kill/%d", killMail.ID),
			Timestamp: killMail.Time,
			Color:     6710886,
			Thumbnail: &discordgo.MessageEmbedThumbnail{
				URL: fmt.Sprintf("https://imageserver.eveonline.com/Render/%d_128.png", killMail.Victim.ShipTypeID),
			},
			Fields: []*discordgo.MessageEmbedField{
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

	if len(friendlies) > 0 {
		var members []string

		for _, id := range friendlies {
			members = append(members, fmt.Sprintf("[%s](https://zkillboard.com/character/%d/)", names[id].Name, id))
		}

		message.Embed.Fields = append(message.Embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Friendly Pilots Involved",
			Value:  strings.Join(members, ", "),
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

	ids, finalBlowID, friendlies, isAttacker, isVictim := getIds(killMail)

	if isVictim == true || isAttacker == true {
		names, error := esiClient.GetNames(ids)
		if error != nil {
			log.Println("Error getting names from esi: ", error)
		}

		message := buildDiscordMessage(killMail, names, finalBlowID, redis.Zkb.TotalValue, friendlies, isAttacker, isVictim)

		sendKillMailMessage(killsChannel, message)
		return
	}

	query := fmt.Sprintf(
		"SELECT systems.id, systems.identifier, systems.name as system_name, systems.map_id, maps.name as map_name, systems.status FROM systems JOIN maps ON maps.id = systems.map_id WHERE maps.owner_id = %d AND systems.id = %d;",
		alliance,
		killMail.SystemID,
	)

	row := postgres.QueryRow(query)
	switch error := row.Scan(); error {
	case sql.ErrNoRows:
		log.Println(fmt.Sprintf("System %d not on any available chains", killMail.SystemID))
	case nil:
		names, error := esiClient.GetNames(ids)
		if error != nil {
			log.Println("Error getting names from esi: ", error)
		}

		message := buildDiscordMessage(killMail, names, finalBlowID, redis.Zkb.TotalValue, friendlies, isAttacker, isVictim)

		sendKillMailMessage(intelChannel, message)
		return
	default:
		panic(error)
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

	port := strings.Trim(os.Getenv("POSTGRES_PORT"), " ")
	if port == "" {
		log.Fatal("Environment variable `POSTGRES_PORT` is required to connect to the systems database")
	}

	dbname := strings.Trim(os.Getenv("POSTGRES_DB"), " ")
	if dbname == "" {
		log.Fatal("Environment variable `POSTGRES_DB` is required to connect to the systems database")
	}

	password := strings.Trim(os.Getenv("POSTGRES_PASSWORD"), " ")
	if password == "" {
		log.Fatal("Environment variable `POSTGRES_PASSWORD` is required to connect to the systems database")
	}

	psqlInfo = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

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

		log.Println(alliance)
		log.Println(corporation)
		log.Println(psqlInfo)
		log.Println(botToken)
		log.Println(killsChannel)
		log.Println(intelChannel)

		postgres, error = sql.Open("postgres", psqlInfo)
		if error != nil {
			panic(error)
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
