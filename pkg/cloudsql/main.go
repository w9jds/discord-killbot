package cloudsql

import (
	"database/sql"
	"fmt"
	"log"

	// Used to connect to gcp postgres
	_ "github.com/GoogleCloudPlatform/cloudsql-proxy/proxy/dialers/postgres"
)

// CloudSQL is an instance of a connection to GCP postgres
type CloudSQL struct {
	postgres *sql.DB
}

// ConnectCloudSQL connect to GCP Cloud SQL instance
func ConnectCloudSQL(dsn string) *CloudSQL {
	var postgres, err = sql.Open("cloudsqlpostgres", dsn)
	if err != nil {
		log.Fatal(err)
	}

	defer postgres.Close()

	return &CloudSQL{
		postgres: postgres,
	}
}

func (cloudSql *CloudSQL) checkChainForSystem(ID uint32) (bool, string, string) {
	var mapID string
	var mapName string

	row := cloudSql.postgres.QueryRow(fmt.Sprintf(
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
