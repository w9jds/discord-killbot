package zkill

import (
	"killbot/pkg/discord"
	"net/http"

	esi "github.com/w9jds/go.esi"
	zkb "github.com/w9jds/zkb"

	"log"
	"time"
)

// KillProcessor is an instance of a redis processor for kills
type KillProcessor struct {
	zkbClient  *zkb.Client
	esiClient  *esi.Client
	AllianceID uint32
	CorpID     uint32
	QueueID    string
}

// CreateKillProcessor creates a new KillProcessor
func CreateKillProcessor(allianceID uint32, corpID uint32, queueID string) *KillProcessor {
	var httpClient = &http.Client{}

	return &KillProcessor{
		zkbClient:  zkb.CreateClient(httpClient),
		esiClient:  esi.CreateClient(httpClient),
		AllianceID: allianceID,
		CorpID:     corpID,
		QueueID:    queueID,
	}
}

// StartWatch has a processor start watching for new kill
func (processor *KillProcessor) StartWatch() {
	log.Println("Concord Alerts have started processing!")

	for {
		bundle, error := processor.zkbClient.GetRedisItem(processor.QueueID)
		if error != nil || bundle.ID == 0 {
			time.Sleep(5 * time.Second)
			continue
		}

		go processor.process(bundle)
	}
}

func (processor *KillProcessor) isCorpOrAlliance(CorporationID uint32, AllianceID uint32) bool {
	if processor.AllianceID != 0 && AllianceID == processor.AllianceID {
		return true
	}

	if CorporationID == processor.CorpID {
		return true
	}

	return false
}

func (processor *KillProcessor) getEsiKillmail(id uint32, hash string) *esi.KillMail {
	killMail, _, err := processor.esiClient.GetKillMail(id, hash, false)

	if err != nil {
		log.Println("error pulling killmail from esi: ", err)
		return nil
	}

	return killMail
}

func (processor *KillProcessor) processAffiliatedKill(killMail *esi.KillMail) {
	names, error := processor.esiClient.GetNames(ids)
	if error != nil {
		log.Println("Error getting names from esi: ", error)
	}

	details := &discord.MessageDetails{
		Names         names,
		KillMail      killMail,
		FinalBlowID   finalBlowID,
		HostileCorpID hostileCorpID,
		TotalValue    totalValue,
		Friendlies    Friendlies,
		MapName       "",
		IsAttacker    isAttacker,
		IsVictim      isVictim,
	}

	message := discord

	sendKillMailMessage(message)
	return
}

func (processor *KillProcessor) processChainKill(ids []uint32, killMail *esi.KillMail) {
	names, error := processor.esiClient.GetNames(ids)
	if error != nil {
		log.Println("Error getting names from esi: ", error)
	}

	message := buildDiscordMessage(killMail, names, finalBlowID, hostileCorpID, redis.Zkb.TotalValue, friendlies, mapName, isAttacker, isVictim)

	sendKillMailMessage(message)
	return
}

func (processor *KillProcessor) process(redis *zkb.RedisResponse) {
	if killMail := processor.getEsiKillmail(redis.ID, redis.Zkb.Hash); killMail != nil {
		ids, finalBlowID, hostileCorpID, friendlies, isAttacker, isVictim := getUniqueIds(killMail)



		if isVictim == true || isAttacker == true {
			processor.processAffiliatedKill()
		}

		if isInChain, _, mapName := checkChainForSystem(killMail.SystemID); isInChain {
			processor.processChainKill()
		}
	}
}

func getUniqueIds(killMail *esi.KillMail) ([]uint32, uint32, uint32, []uint32, bool, bool) {
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
