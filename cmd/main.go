package main

import (
	"encoding/json"
	"fmt"
	"killbot/pkg/discord"
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

	"github.com/gorilla/websocket"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

var (
	httpClient     *http.Client
	discordAPI     *discord.Discord
	esiAPI         *esi.Esi
	redisQueue     *zkb.Redisq
	webhook        string
	allianceEnv    int32
	corporationEnv int32
)

func isCorpOrAlliance(AllianceID int32, CorporationID int32) bool {

	if allianceEnv != 0 && AllianceID == allianceEnv {
		return true
	}

	if CorporationID == corporationEnv {
		return true
	}

	return false
}

func getIds(killMail *esi.KillMail) ([]int32, int32, []int32, bool, bool) {
	var finalBlowID int32

	isAttacker := false
	isVictim := isCorpOrAlliance(killMail.Victim.AllianceID, killMail.Victim.CorporationID)

	unique := make(map[int32]struct{})
	unique[killMail.Victim.ID] = struct{}{}
	unique[killMail.SolarSystemID] = struct{}{}
	unique[killMail.Victim.ShipTypeID] = struct{}{}
	unique[killMail.Victim.CorporationID] = struct{}{}

	if killMail.Victim.AllianceID != 0 {
		unique[killMail.Victim.AllianceID] = struct{}{}
	}

	friendlies := []int32{}
	for _, attacker := range killMail.Attackers {
		isFriendly := isCorpOrAlliance(attacker.AllianceID, attacker.CorporationID)

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

	ids := []int32{}
	for key := range unique {
		ids = append(ids[:], key)
	}

	return ids, finalBlowID, friendlies, isAttacker, isVictim
}

func getKillMailNames(killMail *esi.KillMail, ids []int32) map[int32]esi.NameRef {
	references := map[int32]esi.NameRef{}

	buffer, error := json.Marshal(ids)
	if error != nil {
		log.Println("Unable serialize list of ids: ", error)
	}

	names, error := esiAPI.UniverseNames(buffer)
	if error != nil {
		log.Println("Unable to get names from killmail: ", error)
	}

	for _, ref := range names {
		references[ref.ID] = ref
	}

	return references
}

func buildAttachment(killMail *esi.KillMail, names map[int32]esi.NameRef, finalBlowID int32, totalValue float64, friendlies []int32, isAttacker bool, isVictim bool) discord.Attachment {
	var embeds []discord.Embed
	var fields []discord.Field

	printer := message.NewPrinter(language.English)

	embed := discord.Embed{}
	embed.Color = 6710886

	if isAttacker == true {
		embed.Color = 8103679
	}

	if isVictim == true {
		embed.Color = 16711680
	}

	if isVictim == true && isAttacker == true {
		embed.Color = 6570404
	}

	embed.Title = fmt.Sprintf("%s killed %s (%s)", names[finalBlowID].Name, names[killMail.Victim.ID].Name, names[killMail.Victim.CorporationID].Name)
	embed.URL = fmt.Sprintf("https://zkillboard.com/kill/%d", killMail.KillmailID)
	embed.Timestamp = killMail.KillmailTime
	embed.Thumbnail = discord.Image{
		URL: fmt.Sprintf("https://imageserver.eveonline.com/Render/%d_128.png", killMail.Victim.ShipTypeID),
	}

	fields = append(fields, discord.Field{
		Name:   "Ship",
		Value:  fmt.Sprintf("[%s](https://zkillboard.com/ship/%d)", names[killMail.Victim.ShipTypeID].Name, killMail.Victim.ShipTypeID),
		Inline: true,
	})

	fields = append(fields, discord.Field{
		Name:   "System",
		Value:  fmt.Sprintf("[%s](https://zkillboard.com/system/%d)", names[killMail.SolarSystemID].Name, killMail.SolarSystemID),
		Inline: true,
	})

	fields = append(fields, discord.Field{
		Name:   "Pilots Involved",
		Value:  fmt.Sprintf("%d", len(killMail.Attackers)),
		Inline: true,
	})

	fields = append(fields, discord.Field{
		Name:   "Value",
		Value:  printer.Sprintf("%.2f ISK", totalValue),
		Inline: true,
	})

	if len(friendlies) > 0 {
		var members []string

		for _, id := range friendlies {
			members = append(members, fmt.Sprintf("[%s](https://zkillboard.com/character/%d/)", names[id].Name, id))
		}

		fields = append(fields, discord.Field{
			Name:   "Friendly Pilots Involved",
			Value:  strings.Join(members, ", "),
			Inline: false,
		})
	}

	embed.Fields = fields
	attachment := discord.Attachment{Username: "Concord Alerts", Embeds: append(embeds, embed)}

	if totalValue > 3000000000 && len(friendlies) > 3 {
		attachment.Content = "WOAH! Look at that kill @everyone"
	}

	return attachment
}

func processKillMail(zkill zkb.Zkb) {
	killMail, error := esiAPI.Killmail(zkill.Href)
	if error != nil {
		log.Println("Error pulling killmail from esi: ", error)
		return
	}

	ids, finalBlowID, friendlies, isAttacker, isVictim := getIds(killMail)
	// log.Println(fmt.Sprintf("Processing Kill %d", killMail.KillmailID))

	if isVictim == true || isAttacker == true {
		names := getKillMailNames(killMail, ids)
		attachment := buildAttachment(killMail, names, finalBlowID, zkill.TotalValue, friendlies, isAttacker, isVictim)

		buffer, error := json.Marshal(attachment)
		if error != nil {
			log.Println("Unable serialize the discord attachment: ", error)
			return
		}

		discordAPI.PushWebhook(webhook, buffer)
	}
}

func setupEnv() bool {
	webhook = os.Getenv("WEBHOOK")

	alliance, error := strconv.ParseInt(os.Getenv("ALLIANCE_ID"), 10, 32)
	if error != nil {
		log.Fatal("Environment variable 'ALLIANCE_ID' wasn't set correctly (or at all)")
	}

	corporation, error := strconv.ParseInt(os.Getenv("CORPORATION_ID"), 10, 32)
	if error != nil {
		log.Fatal("Environment Variable 'CORPORATION_ID' is required!")
	}

	allianceEnv = int32(alliance)
	corporationEnv = int32(corporation)

	// exporter, error := stackdriver.NewExporter(stackdriver.Options{ProjectID: os.Getenv("PROJECT_ID")})
	// if error != nil {
	// 	log.Fatal(error)
	// }

	// view.RegisterExporter(exporter)

	// error = view.Register(ochttp.ClientLatencyView, ochttp.ClientResponseBytesView)
	// if error != nil {
	// 	log.Fatal(error)
	// }

	// trace.RegisterExporter(exporter)

	return true
}

func main() {
	isReady := setupEnv()

	if isReady == true {
		httpClient = &http.Client{
			// Transport: &ochttp.Transport{
			// 	Propagation: &propagation.HTTPFormat{},
			// },
		}
		esiAPI = esi.NewAPI(httpClient)
		discordAPI = discord.NewAPI(httpClient)
		redisQueue = zkb.NewAPI(httpClient)

		getKillmails()
	}

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc
}

func getKillmails() {

	log.Println("Concord Alerts have started processing!")

	for {
		bundle, error := redisQueue.GetItem("chingy-killbot-discord")
		if error != nil || bundle.KillID == 0 {
			time.Sleep(5 * time.Second)
			continue
		}

		go processKillMail(bundle.ZKB)
	}
}

func connectWebsocket() {
	streamSub := []byte("{\"action\":\"sub\",\"channel\":\"killstream\"}")

	socket, _, error := websocket.DefaultDialer.Dial("wss://zkillboard.com:2096", nil)
	if error != nil {
		log.Println("Error opening websocket to zkillboard: ", error)
	}

	log.Println("Concord Alerts have started processing!")

	defer socket.Close()

	done := make(chan struct{})

	socket.WriteMessage(websocket.TextMessage, streamSub)

	zkillChannel := make(chan []byte)
	go readZkillMessage(socket, zkillChannel, done)

	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {

		select {

		case <-done:
			return

		case tick := <-ticker.C:
			log.Println(fmt.Sprintf("%s: Keep Alive Subscription", tick.String()))

			error := socket.WriteMessage(websocket.TextMessage, []byte(""))
			if error != nil {
				log.Println("KeepAlive error: ", error)
				return
			}

		case message := <-zkillChannel:
			var bundle zkb.Bundle
			error = json.Unmarshal(message, &bundle)
			if error != nil {
				log.Println(message)
				break
			}

			go processKillMail(bundle.ZKB)
		}

	}
}

func readZkillMessage(socket *websocket.Conn, zkillChannel chan []byte, done chan struct{}) {
	for {
		defer close(done)

		_, message, error := socket.ReadMessage()
		if error != nil {
			log.Println("Error receiving message: ", error)
			return
		}

		zkillChannel <- message
	}
}
