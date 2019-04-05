package esi

import (
	"encoding/json"
	"fmt"
)

// KillMail that is recieved from eve online
type KillMail struct {
	ID        uint32     `json:"killmail_id,omitempty"`
	Time      string     `json:"killmail_time,omitempty"`
	SystemID  uint32     `json:"solar_system_id,omitempty"`
	Victim    victim     `json:"victim,omitempty"`
	Attackers []attacker `json:"attackers,omitempty"`
	WarID     uint32     `json:"war_id,omitempty"`
}

type victim struct {
	ID            uint32     `json:"character_id,omitempty"`
	AllianceID    uint32     `json:"alliance_id,omitempty"`
	CorporationID uint32     `json:"corporation_id,omitempty"`
	DamageTaken   uint64     `json:"damage_taken,omitempty"`
	Items         []KillItem `json:"items,omitempty"`
	ShipTypeID    uint32     `json:"ship_type_id,omitempty"`
}

type attacker struct {
	ID             uint32  `json:"character_id,omitempty"`
	AllianceID     uint32  `json:"alliance_id,omitempty"`
	CorporationID  uint32  `json:"corporation_id,omitempty"`
	DamageDone     uint64  `json:"damage_done,omitempty"`
	FinalBlow      bool    `json:"final_blow,omitempty"`
	SecurityStatus float32 `json:"security_status,omitempty"`
	ShipTypeID     uint32  `json:"ship_type_id,omitempty"`
	WeaponTypeID   uint32  `json:"weapon_type_id,omitempty"`
}

// KillItem is an item that is found on a killmail
type KillItem struct {
	ID                uint32 `json:"item_type_id,omitempty"`
	Flag              int16  `json:"flag,omitempty"`
	QuantityDropped   uint32 `json:"quantity_dropped,omitempty"`
	QuantityDestroyed uint32 `json:"quantity_destroyed,omitempty"`
	Singleton         int8   `json:"singleton,omitempty"`
}

// KillFitting the fitting built from the items on the victim's killmail
type KillFitting struct {
	SubSystemSlot map[uint32]*KillItem
	HighSlot      map[uint32]*KillItem
	MedSlot       map[uint32]*KillItem
	LoSlot        map[uint32]*KillItem
	RigSlot       map[uint32]*KillItem
	FighterBay    map[uint32]*KillItem
	ServiceSlot   map[uint32]*KillItem
	Cargo         map[uint32]*KillItem
	DroneBay      map[uint32]*KillItem
}

// GetKillMail retrieves a specific killmail from ESI
func (esi Client) GetKillMail(killID uint32, hash string, withFitting bool) (*KillMail, *KillFitting, error) {
	body, error := esi.get(fmt.Sprintf("/v1/killmails/%d/%s/", killID, hash))
	if error != nil {
		return nil, nil, error
	}

	var killmail KillMail
	error = json.Unmarshal(body, &killmail)
	if error != nil {
		return nil, nil, error
	}

	if withFitting == true {
		return &killmail, buildShipFitting(killmail), nil
	}

	return &killmail, nil, nil
}

func updateFittingItem(group map[uint32]*KillItem, item KillItem) {
	if current, ok := group[item.ID]; ok {
		current.QuantityDestroyed += item.QuantityDestroyed
		current.QuantityDropped += item.QuantityDropped
	} else {
		group[item.ID] = &item
	}
}

func buildShipFitting(killmail KillMail) *KillFitting {
	fit := &KillFitting{
		SubSystemSlot: map[uint32]*KillItem{},
		HighSlot:      map[uint32]*KillItem{},
		MedSlot:       map[uint32]*KillItem{},
		LoSlot:        map[uint32]*KillItem{},
		RigSlot:       map[uint32]*KillItem{},
		Cargo:         map[uint32]*KillItem{},
		DroneBay:      map[uint32]*KillItem{},
	}

	for _, item := range killmail.Victim.Items {
		if item.Flag == 5 {
			updateFittingItem(fit.Cargo, item)
		} else if item.Flag == 87 {
			updateFittingItem(fit.DroneBay, item)
		} else if item.Flag >= 27 && item.Flag <= 34 {
			updateFittingItem(fit.HighSlot, item)
		} else if item.Flag >= 19 && item.Flag <= 26 {
			updateFittingItem(fit.MedSlot, item)
		} else if item.Flag >= 11 && item.Flag <= 18 {
			updateFittingItem(fit.LoSlot, item)
		} else if item.Flag >= 92 && item.Flag <= 94 {
			updateFittingItem(fit.RigSlot, item)
		} else if item.Flag >= 125 && item.Flag <= 128 {
			updateFittingItem(fit.SubSystemSlot, item)
		}
	}

	return fit
}
