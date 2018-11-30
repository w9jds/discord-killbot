package esi

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

// NameRef ESI Name Reference Response
type NameRef struct {
	Category string `json:"category"`
	ID       int32  `json:"id"`
	Name     string `json:"name"`
}

// KillMail ESI KillMail Response
type KillMail struct {
	Victim        Victim     `json:"victim"`
	Attackers     []Attacker `json:"attackers"`
	KillmailID    int32      `json:"killmail_id"`
	KillmailTime  string     `json:"killmail_time"`
	MoonID        int32      `json:"moon_id"`
	SolarSystemID int32      `json:"solar_system_id"`
	WarID         int32      `json:"war_id"`
}

// Character character base structure
type Character struct {
	ID            int32 `json:"character_id"`
	AllianceID    int32 `json:"alliance_id"`
	CorporationID int32 `json:"corporation_id"`
}

// Attacker ESI structure for killmail attacker
type Attacker struct {
	// *Character
	ID             int32   `json:"character_id"`
	AllianceID     int32   `json:"alliance_id"`
	CorporationID  int32   `json:"corporation_id"`
	DamageDone     int32   `json:"damage_done"`
	FactionID      int32   `json:"faction_id"`
	FinalBlow      bool    `json:"final_blow"`
	SecurityStatus float64 `json:"security_status"`
	ShipTypeID     int32   `json:"ship_type_id"`
	WeaponTypeID   int32   `json:"weapon_type_id"`
}

// Victim ESI structure for killmail victim
type Victim struct {
	// *Character
	ID            int32 `json:"character_id"`
	AllianceID    int32 `json:"alliance_id"`
	CorporationID int32 `json:"corporation_id"`
	DamageTaken   int32 `json:"damage_taken"`
	FactionID     int32 `json:"faction_id"`
	ShipTypeID    int32 `json:"ship_type_id"`
}

// Esi base client for communicating with EVE Online's Api
type Esi struct {
	client *http.Client
}

// NewAPI returns a new instance of Esi with the passed in http client
func NewAPI(httpClient *http.Client) *Esi {
	return &Esi{client: httpClient}
}

// UniverseNames returns a list of names from the universe for passed Ids
func (esi *Esi) UniverseNames(buffer []byte) ([]NameRef, error) {
	request, error := http.NewRequest("POST", "https://esi.evetech.net/v2/universe/names/", bytes.NewBuffer(buffer))
	if error != nil {
		log.Println("Error creating new request: ", error)
	}

	request.Header.Add("User-Agent", "Killbot - Chingy Chonga/Jeremy Shore - w9jds@live.com")
	request.Header.Add("Accept", "application/json")

	response, error := esi.client.Do(request)
	if error != nil {
		log.Println("Unable to get kill mail names: ", error)
		return nil, error
	}

	body, error := ioutil.ReadAll(response.Body)
	if error != nil {
		log.Println("Invalid response: ", error)
		return nil, error
	}

	var names []NameRef
	error = json.Unmarshal(body, &names)
	if error != nil {
		log.Println("Unable to parse names: ", error)
		return nil, error
	}

	return names, nil
}

// Killmail receives the killmail from the uri passed in
// func (esi *Esi) Killmail(uri string) (*KillMail, error) {
// 	request, error := http.NewRequest("GET", uri, nil)
// 	request.Header.Add("User-Agent", "Killbot - Chingy Chonga/Jeremy Shore - w9jds@live.com")
// 	request.Header.Add("Accept", "application/json")

// 	response, error := esi.client.Do(request)
// 	if error != nil {
// 		log.Println("Unable to get Killmail: ", error)
// 		return nil, error
// 	}

// 	body, error := ioutil.ReadAll(response.Body)
// 	if error != nil {
// 		log.Println("Invalid response: ", error)
// 		return nil, error
// 	}

// 	var killMail *KillMail
// 	error = json.Unmarshal(body, &killMail)
// 	if error != nil {
// 		log.Println("Error received from esi: ", string(body))
// 		return nil, error
// 	}

// 	return killMail, nil
// }

// Killmail receives the killmail from the uri passed in
func (esi *Esi) Killmail(uri string) (*KillMail, error) {
	var killMail *KillMail

	success := make(chan interface{})
	failed := make(chan error)

	go esi.request("GET", uri, nil, success, failed, &killMail)

	for {
		select {
		case <-success:
			return killMail, nil
		case error := <-failed:
			return nil, error
		}
	}
}

func (esi *Esi) request(method string, uri string, data io.Reader, success chan interface{}, failed chan error, contents interface{}) {
	for i := 3; i >= 0; i-- {
		request, error := http.NewRequest(method, uri, data)
		request.Header.Add("User-Agent", "Killbot - Chingy Chonga/Jeremy Shore - w9jds@live.com")
		request.Header.Add("Accept", "application/json")

		response, error := esi.client.Do(request)
		if error != nil {

			log.Println("Esi request failed: ", error)
			time.Sleep(5 * time.Second)

			if i == 0 {
				failed <- error
			}

			continue
		}

		body, error := ioutil.ReadAll(response.Body)
		if error != nil {
			log.Println("Invalid response: ", error)
			time.Sleep(5 * time.Second)

			if i == 0 {
				failed <- error
			}

			continue
		}

		error = json.Unmarshal(body, contents)
		if error != nil {
			log.Println("Error received from esi: ", string(body))
			time.Sleep(5 * time.Second)

			if i == 0 {
				failed <- error
			}

			continue
		}

		success <- contents
		break
	}
}
