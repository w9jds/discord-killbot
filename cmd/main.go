package main

import (
	"database/sql"
	"esi"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"zkb"

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

	dsn string
)

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

	dsn = fmt.Sprintf("host=%s dbname=%s user=%s password=%s sslmode=disable", host, dbname, user, password)

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

		// ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		// defer cancel()

		// client, error := authenticatedClient(ctx)
		// if error != nil {
		// 	log.Fatal(error)
		// }

		// proxy.Init(client, nil, nil)

		postgres, error = sql.Open("cloudsqlpostgres", dsn)
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

		fetch()
	}

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc
}
