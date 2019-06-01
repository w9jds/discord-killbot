package main

import (
	"killbot/pkg/esi"
	"killbot/pkg/zkb"
	"log"
	"time"
)

func fetch() {
	log.Println("Concord Alerts have started processing!")

	for {
		bundle, error := zkbClient.GetRedisItem(queueID)
		if error != nil || bundle.ID == 0 {
			time.Sleep(5 * time.Second)
			continue
		}

		go process(bundle)
	}
}

func isCorpOrAlliance(CorporationID uint32, AllianceID uint32) bool {
	if alliance != 0 && AllianceID == alliance {
		return true
	}

	if CorporationID == corporation {
		return true
	}

	return false
}

func process(redis *zkb.RedisResponse) {
	killMail, _, error := esiClient.GetKillMail(redis.ID, redis.Zkb.Hash, false)
	if error != nil {
		log.Println("Error pulling killmail from esi: ", error)
		return
	}

	ids, finalBlowID, hostileCorpID, friendlies, isAttacker, isVictim := getUniqueIds(killMail)

	if isVictim == true || isAttacker == true {
		names, error := esiClient.GetNames(ids)
		if error != nil {
			log.Println("Error getting names from esi: ", error)
		}

		message := buildDiscordMessage(killMail, names, finalBlowID, hostileCorpID, redis.Zkb.TotalValue, friendlies, "", isAttacker, isVictim)

		sendKillMailMessage(killsChannel, message)
		return
	}

	if isInChain, _, mapName := checkChainForSystem(killMail.SystemID); isInChain {
		names, error := esiClient.GetNames(ids)
		if error != nil {
			log.Println("Error getting names from esi: ", error)
		}

		message := buildDiscordMessage(killMail, names, finalBlowID, hostileCorpID, redis.Zkb.TotalValue, friendlies, mapName, isAttacker, isVictim)

		sendKillMailMessage(intelChannel, message)
		return
	}

	// log.Println(fmt.Sprintf("System %d is not in any available chains, killmail skipped.", killMail.SystemID))
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
