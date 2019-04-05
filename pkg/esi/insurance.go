package esi

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
)

type insurance struct {
	Levels []coverageLevel `json:"levels,omitempty"`
	TypeID uint32          `json:"type_id,omitempty"`
}

type coverageLevel struct {
	Cost   float64 `json:"cost,omitempty"`
	Name   string  `json:"name,omitempty"`
	Payout float64 `json:"payout,omitempty"`
}

// Coverage houses all the levels of insurance for a specific item
type Coverage struct {
	Basic    coverageLevel
	Standard coverageLevel
	Bronze   coverageLevel
	Silver   coverageLevel
	Gold     coverageLevel
	Platinum coverageLevel
}

// GetShipInsurance gets all insurance values and filters out anything that isn't for the specified ShipID
func (esi Client) GetShipInsurance(shipID uint32) (*Coverage, error) {
	body, error := esi.get(fmt.Sprintf("/v1/insurance/prices/"))
	if error != nil {
		return nil, error
	}

	var ships []insurance
	error = json.Unmarshal(body, &ships)
	if error != nil {
		return nil, error
	}

	for _, insurance := range ships {
		if insurance.TypeID == shipID {
			return buildCoverage(insurance), nil
		}
	}

	return nil, errors.New("Insurance for specified shipID was not found")
}

func buildCoverage(insurance insurance) *Coverage {
	coverage := &Coverage{}

	for _, level := range insurance.Levels {
		field := reflect.ValueOf(coverage).Elem().FieldByName(level.Name)
		field.Set(reflect.ValueOf(level))
	}

	return coverage
}
