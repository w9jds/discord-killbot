package main

import (
	"database/sql"
	"fmt"
	"log"
)

func checkChainForSystem(ID uint32) (bool, string, string) {
	var mapID string
	var mapName string

	row := postgres.QueryRow(fmt.Sprintf(
		"SELECT systems.map_id, maps.name FROM systems JOIN maps ON maps.id = systems.map_id WHERE maps.owner_id = %d AND systems.id = %d LIMIT 1;",
		alliance,
		ID,
	))

	switch error := row.Scan(&mapID, &mapName); error {
	case sql.ErrNoRows:
		return false, mapID, mapName
	case nil:
		return true, mapID, mapName
	default:
		log.Fatal(error)
		return false, mapID, mapName
	}
}
