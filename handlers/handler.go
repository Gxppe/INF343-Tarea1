
import (
	"database/sql"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Getter races
func Get_Races(db *sql.DB) ([]map[string]interface{}, error) {
	rows, err := db.Query(`SELECT * FROM session WHERE session_name = 'Race';`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runners []map[string]interface{}
	for rows.Next() {
		var session_key int
		var country_name string
		var date_start time.Time
		var year int
		var circuit_short_name string
		err := rows.Scan(&session_key, &country_name, &date_start, &year, &circuit_short_name)
		if err != nil {
			

// Getter for all drivers
func Get_Drivers(db *sql.DB) ([]map[string]interface{}, error) {
	rows, err := db.Query(`SELECT * FROM session WHERE session_name = 'Race';`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runners []map[string]interface{}
	for rows.Next() {
		var first_name string
		var last_name string
		var team_name string
		var country_code string

		err := rows.Scan(&first_name, &last_name, &team_name, &country_code)
		if err != nil {
			return nil, err
		}
		runner := map[string]interface{}{
			"first_name":   first_name,
			"last_name":    last_name,
			"team_name":    team_name,
			"country_code": country_code,
		}
		runners = append(runners, runner)
	}
	return runners, nil
}

// Getter for driver details
func Get_DriverDetails(db *sql.DB, driverID int) map[string]interface{} {
	// Consulta para obtener el resumen de rendimiento del piloto
	var wins, top3Finishes, maxSpeed int
	var raceResults []map[string]interface{}

	// Obtener el resumen de rendimiento
	row := db.QueryRow(`SELECT wins, top_3_finishes, max_speed FROM performance_summary WHERE driver_id = ?`, driverID)
	row.Scan(&wins, &top3Finishes, &maxSpeed)

	performanceSummary := map[string]interface{}{
		"wins":           wins,
		"top_3_finishes": top3Finishes,
		"max_speed":      maxSpeed,
	}

	// Obtener los resultados de las carreras del piloto
	rows, _ := db.Query(`SELECT session_key, circuit_short_name, race, position, fastest_lap, max_speed, best_lap_duration
		FROM race_results WHERE driver_id = ?`, driverID)
	defer rows.Close()

	for rows.Next() {
		var sessionKey int
		var circuitShortName, race string
		var position int
		var fastestLap bool
		var lapMaxSpeed float64
		var bestLapDuration float64

		// Escanear los resultados de cada carrera
		rows.Scan(&sessionKey, &circuitShortName, &race, &position, &fastestLap, &lapMaxSpeed, &bestLapDuration)

		// Almacenar los resultados de la carrera
		raceResult := map[string]interface{}{
			"session_key":        sessionKey,
			"circuit_short_name": circuitShortName,
			"race":               race,
			"position":           position,
			"fastest_lap":        fastestLap,
			"max_speed":          lapMaxSpeed,
			"best_lap_duration":  bestLapDuration,
		}
		raceResults = append(raceResults, raceResult)
	}

	// Devolver los detalles completos del piloto
	return map[string]interface{}{
		"driver_id":           driverID,
		"performance_summary": performanceSummary,
		"race_results":        raceResults,
	}
}

// Getter for session details
func Get_SessionDetails(db *sql.DB, sessionID int) map[string]interface{} {
	var sessionKey, year int
	var countryName, circuitShortName, race string
	var dateStart time.Time

	// Consulta para obtener la información de la sesión
	row := db.QueryRow(`SELECT session_key, country_name, date_start, year, circuit_short_name, race
		FROM session WHERE session_key = ?`, sessionID)
	row.Scan(&sessionKey, &countryName, &dateStart, &year, &circuitShortName, &race)

	// Obtener los resultados de la carrera
	var results []map[string]interface{}
	rows, _ := db.Query(`SELECT position, driver, team, country FROM results WHERE session_key = ?`, sessionID)
	defer rows.Close()

	for rows.Next() {
		var position int
		var driver, team, country string

		// Escanear los resultados de cada piloto
		rows.Scan(&position, &driver, &team, &country)

		// Almacenar los resultados de la carrera
		result := map[string]interface{}{
			"position": position,
			"driver":   driver,
			"team":     team,
			"country":  country,
		}
		results = append(results, result)
	}

	// Obtener la vuelta más rápida y la velocidad máxima alcanzada
	var fastestLapDriver, maxSpeedDriver string
	var totalTime, sector1, sector2, sector3, maxSpeed float64

	// Obtener los datos de la vuelta más rápida
	row = db.QueryRow(`SELECT driver, total_time, sector_1, sector_2, sector_3 FROM fastest_lap WHERE session_key = ?`, sessionID)
	row.Scan(&fastestLapDriver, &totalTime, &sector1, &sector2, &sector3)

	// Obtener los datos de la velocidad máxima
	row = db.QueryRow(`SELECT driver, speed_kmh FROM max_speed WHERE session_key = ?`, sessionID)
	row.Scan(&maxSpeedDriver, &maxSpeed)

	// Devolver todos los detalles de la sesión
	return map[string]interface{}{
		"session_key":        sessionKey,
		"country_name":       countryName,
		"date_start":         dateStart,
		"year":               year,
		"circuit_short_name": circuitShortName,
		"race":               race,
		"results":            results,
		"fastest_lap": map[string]interface{}{
			"driver":     fastestLapDriver,
			"total_time": totalTime,
			"sector_1":   sector1,
			"sector_2":   sector2,
			"sector_3":   sector3,
		},
		"max_speed": map[string]interface{}{
			"driver":    maxSpeedDriver,
			"speed_kmh": maxSpeed,
		},
	}
}

// Getter for season summary
func Get_SeasonSummary(db *sql.DB, season int) map[string]interface{} {
	// Obtener los 3 pilotos con más victorias
	var top3Winners []map[string]interface{}
	rows, _ := db.Query(`SELECT driver, team, country, COUNT(*) AS wins
		FROM results WHERE season = ? GROUP BY driver ORDER BY wins DESC LIMIT 3`, season)
	defer rows.Close()

	for rows.Next() {
		var driver, team, country string
		var wins int
		rows.Scan(&driver, &team, &country, &wins)

		top3Winners = append(top3Winners, map[string]interface{}{
			"driver":  driver,
			"team":    team,
			"country": country,
			"wins":    wins,
		})
	}

	// Obtener los 3 pilotos con más vueltas rápidas
	var top3FastestLaps []map[string]interface{}
	rows, _ = db.Query(`SELECT driver, team, country, COUNT(*) AS fastest_laps
		FROM laps WHERE season = ? AND fastest_lap = 1 GROUP BY driver ORDER BY fastest_laps DESC LIMIT 3`, season)
	defer rows.Close()

	for rows.Next() {
		var driver, team, country string
		var fastestLaps int
		rows.Scan(&driver, &team, &country, &fastestLaps)

		top3FastestLaps = append(top3FastestLaps, map[string]interface{}{
			"driver":       driver,
			"team":         team,
			"country":      country,
			"fastest_laps": fastestLaps,
		})
	}

	// Obtener los 3 pilotos con más pole positions
	var top3PolePositions []map[string]interface{}
	rows, _ = db.Query(`SELECT driver, team, country, COUNT(*) AS poles
		FROM results WHERE season = ? AND pole_position = 1 GROUP BY driver ORDER BY poles DESC LIMIT 3`, season)
	defer rows.Close()

	for rows.Next() {
		var driver, team, country string
		var poles int
		rows.Scan(&driver, &team, &country, &poles)

		top3PolePositions = append(top3PolePositions, map[string]interface{}{
			"driver":  driver,
			"team":    team,
			"country": country,
			"poles":   poles,
		})
	}

	// Crear el resumen de la temporada
	seasonSummary := map[string]interface{}{
		"season":               season,
		"top_3_winners":        top3Winners,
		"top_3_fastest_laps":   top3FastestLaps,
		"top_3_pole_positions": top3PolePositions,
	}

	// Devolver el resumen de la temporada
	return seasonSummary
}
