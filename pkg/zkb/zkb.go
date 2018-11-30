package zkb

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
)

// Package container from Redisq
type Package struct {
	Package Bundle `json:"package"`
}

// Bundle structure for cache data
type Bundle struct {
	KillID int32 `json:"killID"`
	ZKB    Zkb   `json:"zkb"`
}

// Zkb structure with zkillboard information
type Zkb struct {
	LocationID  float64 `json:"locationID"`
	FittedValue float64 `json:"fittedValue"`
	TotalValue  float64 `json:"totalValue"`
	Href        string  `json:"href"`
}

// Redisq base client for hitting the zkill redis queue
type Redisq struct {
	client *http.Client
}

// NewAPI returns a new instance of Redisq with the passed in http client
func NewAPI(httpClient *http.Client) *Redisq {
	return &Redisq{client: httpClient}
}

// GetItem pulls the next item in the zkill redis queue
func (redisq *Redisq) GetItem(queueID string) (*Bundle, error) {
	request, error := http.NewRequest("GET", "https://redisq.zkillboard.com/listen.php?ttw=0&queueID=chingy-killbot-integration", nil)
	if error != nil {
		log.Println("Error creating new request: ", error)
		return nil, error
	}

	response, error := redisq.client.Do(request)
	if error != nil {
		log.Println("Error pulling kills from redisq: ", error)
		return nil, error
	}

	body, error := ioutil.ReadAll(response.Body)
	if error != nil {
		log.Println("Invalid response from redisq: ", error)
		return nil, error
	}

	var bundle Package
	error = json.Unmarshal(body, &bundle)
	if error != nil {
		log.Println("Error parsing bundle response")
		return nil, error
	}

	return &bundle.Package, nil
}
