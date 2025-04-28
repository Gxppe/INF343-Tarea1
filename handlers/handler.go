package handler

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Getter races
func Get_Races(db *sql.DB) ([]map[string]interface{}, error) {
	// Especifica explícitamente las columnas que necesitas
	rows, err := db.Query(`
        SELECT 
            session_key, 
            country_name, 
            date_start, 
            year, 
            circuit_short_name 
        FROM session 
        WHERE session_name = 'Race'
    `)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var races []map[string]interface{}
	for rows.Next() {
		var session_key int
		var country_name string
		var date_start time.Time
		var year int
		var circuit_short_name string

		// Verifica el orden de las columnas al escanear
		err := rows.Scan(
			&session_key,
			&country_name,
			&date_start,
			&year,
			&circuit_short_name,
		)
		if err != nil {
			// No ignores los errores, esto es importante para debuggear
			return nil, fmt.Errorf("error scanning row: %v", err)
		}

		race := map[string]interface{}{
			"session_key":        session_key,
			"country_name":       country_name,
			"date_start":         date_start.Format(time.RFC3339), // Formatea la fecha
			"year":               year,
			"circuit_short_name": circuit_short_name,
		}
		races = append(races, race)
	}

	// Verifica errores después de iterar
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return races, nil
}

// Getter for all drivers
func Get_Drivers(db *sql.DB) ([]map[string]interface{}, error) {
	// Especifica explícitamente las columnas que necesitas
	rows, err := db.Query(`
        SELECT 
            driver_number,
            first_name, 
            last_name, 
            team_name, 
            country_code 
        FROM driver
    `)
	if err != nil {
		return nil, fmt.Errorf("error querying drivers: %v", err)
	}
	defer rows.Close()

	var drivers []map[string]interface{}
	for rows.Next() {
		var (
			driverNumber                               int
			firstName, lastName, teamName, countryCode string
		)

		// Escanea todas las columnas en el orden correcto
		err := rows.Scan(
			&driverNumber,
			&firstName,
			&lastName,
			&teamName,
			&countryCode,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning driver row: %v", err)
		}

		driver := map[string]interface{}{
			"driver_number": driverNumber,
			"first_name":    firstName,
			"last_name":     lastName,
			"team_name":     teamName,
			"country_code":  countryCode,
		}
		drivers = append(drivers, driver)
	}

	// Verifica errores después de iterar
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error after row iteration: %v", err)
	}

	return drivers, nil
}

func GetDriverDetails(db *sql.DB, driverID int) (map[string]interface{}, error) {
	// Validar ID del conductor
	if driverID <= 0 {
		return nil, fmt.Errorf("ID de conductor inválido")
	}

	// Verificar que el conductor existe
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM driver WHERE driver_number = ?)", driverID).Scan(&exists)
	if err != nil || !exists {
		return nil, fmt.Errorf("conductor no encontrado")
	}

	// Obtener estadísticas de rendimiento
	stats, err := getPerformanceStats(db, driverID)
	if err != nil {
		return nil, fmt.Errorf("error obteniendo estadísticas: %v", err)
	}

	// Obtener resultados de carreras
	raceResults, err := getRaceResults(db, driverID)
	if err != nil {
		return nil, fmt.Errorf("error obteniendo resultados de carrera: %v", err)
	}

	// Construir respuesta final
	response := map[string]interface{}{
		"driver_id":           driverID,
		"performance_summary": stats,
		"race_results":        raceResults,
	}

	return response, nil
}

func getPerformanceStats(db *sql.DB, driverID int) (map[string]interface{}, error) {
	stats := make(map[string]interface{})
	var wins, podiums int
	var maxSpeed float64

	// Calcular victorias y podios
	err := db.QueryRow(`
		WITH LatestVictories AS (
    SELECT 
        p.driver_number,
        p.session_key,
        p.position,
        p.date,
        ROW_NUMBER() OVER (PARTITION BY p.session_key ORDER BY p.date DESC) AS rn
    FROM position p
    WHERE p.position = 1
),
DriverLatestPositions AS (
    SELECT 
        p.driver_number,
        p.session_key,
        p.position,
        ROW_NUMBER() OVER (PARTITION BY p.driver_number, p.session_key ORDER BY p.date DESC) AS rn
    FROM position p
    WHERE p.driver_number = 1  -- Corredor específico
),
DriverTop3 AS (
    SELECT 
        p.driver_number,
        COUNT(*) AS top3_count
    FROM position p
    WHERE p.driver_number = ?  -- Corredor específico
    AND p.position <= 3
    AND (p.driver_number, p.session_key, p.date) IN (
        SELECT driver_number, session_key, MAX(date)
        FROM position
        GROUP BY driver_number, session_key
    )
    GROUP BY p.driver_number
)
SELECT 

    COUNT(lv.session_key) AS wins,
    COALESCE(dt.top3_count, 0) AS top3_count
FROM driver d
LEFT JOIN LatestVictories lv 
    ON d.driver_number = lv.driver_number 
    AND lv.rn = 1
LEFT JOIN DriverTop3 dt
    ON d.driver_number = dt.driver_number
WHERE d.driver_number = 1  -- Mismo corredor que en los CTEs
GROUP BY d.driver_number, d.first_name, d.last_name, d.team_name, d.country_code, dt.top3_count;`, driverID).Scan(&wins, &podiums)
	if err != nil {
		return nil, err
	}

	// Obtener velocidad máxima
	err = db.QueryRow(`
		SELECT MAX(st_speed) FROM lap 
		WHERE driver_number = ?`, driverID).Scan(&maxSpeed)
	if err != nil {
		return nil, err
	}

	stats["wins"] = wins
	stats["top_3_finishes"] = podiums
	stats["max_speed"] = maxSpeed

	return stats, nil
}

func getRaceResults(db *sql.DB, driverID int) ([]map[string]interface{}, error) {
	query := `
WITH FinalRacePositions AS (
    SELECT 
        p.driver_number,
        p.session_key,
        p.position,
        p.date,
        ROW_NUMBER() OVER (PARTITION BY p.session_key, p.driver_number ORDER BY p.date DESC) as pos_rank
    FROM position p
    JOIN session s ON p.session_key = s.session_key
    WHERE p.driver_number = ?
    AND s.session_type = 'Race'
)
SELECT 
    frp.session_key,
    s.circuit_short_name,
    s.session_name,
    frp.position as final_position,
    (
        SELECT CAST(l.lap_duration AS REAL)
        FROM lap l 
        WHERE l.session_key = frp.session_key 
        AND l.driver_number = frp.driver_number
        ORDER BY l.lap_duration ASC
        LIMIT 1
    ) as best_lap_duration,
    (
        SELECT MAX(l.st_speed) 
        FROM lap l 
        WHERE l.session_key = frp.session_key 
        AND l.driver_number = frp.driver_number
    ) as max_speed,
    EXISTS(
        SELECT 1 
        FROM lap l 
        WHERE l.session_key = frp.session_key 
        AND l.driver_number = frp.driver_number
        AND l.lap_duration = (
            SELECT MIN(l2.lap_duration) 
            FROM lap l2 
            WHERE l2.session_key = frp.session_key
        )
    ) as had_fastest_lap
FROM FinalRacePositions frp
JOIN session s ON frp.session_key = s.session_key
WHERE frp.pos_rank = 1  -- Solo la última posición registrada en cada carrera
ORDER BY frp.session_key DESC;`

	rows, err := db.Query(query, driverID)
	if err != nil {
		return nil, fmt.Errorf("error executing query: %v", err)
	}
	defer rows.Close()

	var results []map[string]interface{}
	seenSessions := make(map[int]bool)

	for rows.Next() {
		var (
			sessionKey  int
			circuitName string
			raceName    string
			position    int
			bestLapDur  sql.NullFloat64
			maxSpeed    sql.NullFloat64
			fastestLap  bool
		)

		err := rows.Scan(
			&sessionKey,
			&circuitName,
			&raceName,
			&position,
			&bestLapDur,
			&maxSpeed,
			&fastestLap,
		)
		if err != nil {
			log.Printf("Error scanning race result: %v", err)
			continue
		}

		if seenSessions[sessionKey] {
			continue
		}
		seenSessions[sessionKey] = true

		// Manejo de valores NULL
		lapDuration := 0.0
		if bestLapDur.Valid {
			lapDuration = bestLapDur.Float64
		}

		speed := 0.0
		if maxSpeed.Valid {
			speed = maxSpeed.Float64
		}

		result := map[string]interface{}{
			"session_key":        sessionKey,
			"circuit_short_name": circuitName,
			"race":               raceName,
			"position":           position,
			"fastest_lap":        fastestLap,
			"max_speed":          speed,
			"best_lap_duration":  lapDuration,
		}

		results = append(results, result)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error processing rows: %v", err)
	}

	return results, nil
}

// TODO:arreglar CONSUMO DE WINS Y mostrar en pantalla las victorias
func GetRaceDetails(db *sql.DB, raceID int) (map[string]interface{}, error) {
	// 1. Obtener información básica de la carrera
	raceQuery := `
        SELECT 
            session_key,
            circuit_short_name,
            location as country_name,
            date_start,
            strftime('%Y', date_start) as year
        FROM session
        WHERE session_key = ? AND session_type = 'Race'`

	var (
		sessionKey  int
		circuitName string
		countryName string
		dateStart   time.Time
		year        string
	)

	err := db.QueryRow(raceQuery, raceID).Scan(
		&sessionKey,
		&circuitName,
		&countryName,
		&dateStart,
		&year,
	)
	if err != nil {
		return nil, fmt.Errorf("error obteniendo datos de la carrera: %v", err)
	}

	// 2. Obtener primeros 3 puestos
	top3Query := `
	WITH FinalPositions AS (
		SELECT 
			p.driver_number,
			p.position,
			p.date,
			ROW_NUMBER() OVER (PARTITION BY p.driver_number ORDER BY p.date DESC) as pos_rank
		FROM position p
		WHERE p.session_key = ?
	)
	SELECT 
		fp.position,
		d.first_name || ' ' || d.last_name as driver_name,
		d.team_name,
		d.country_code
	FROM FinalPositions fp
	JOIN driver d ON fp.driver_number = d.driver_number
	WHERE fp.pos_rank = 1  -- Solo la última posición registrada de cada piloto
	AND fp.position <= 3   -- Solo posiciones del podio (1°, 2°, 3°)
	ORDER BY fp.position ASC
	LIMIT 3;
    `
	rows, err := db.Query(top3Query, raceID)
	if err != nil {
		return nil, fmt.Errorf("error obteniendo top 3: %v", err)
	}
	defer rows.Close()

	var results []map[string]interface{}
	seenDrivers := make(map[string]bool)

	for rows.Next() {
		var (
			position   int
			driverName string
			teamName   string
			country    string
		)
		if err := rows.Scan(&position, &driverName, &teamName, &country); err != nil {
			log.Printf("Error scanning top 3 result: %v", err)
			continue
		}
		if !seenDrivers[driverName] {
			results = append(results, map[string]interface{}{
				"position": position,
				"driver":   driverName,
				"team":     teamName,
				"country":  country,
			})
			seenDrivers[driverName] = true
		}
	}

	// 3. Obtener último puesto
	lastQuery := `SELECT p.position, d.first_name, d.last_name
FROM position p
JOIN driver d ON p.driver_number = d.driver_number
WHERE p.session_key = ?
AND p.position = (
    SELECT MAX(position) 
    FROM position 
    WHERE session_key = ?
);`
	var (
		lastPos     int
		lastDriver  string
		lastTeam    string
		lastCountry string
	)
	err = db.QueryRow(lastQuery, raceID, raceID).Scan(
		&lastPos, &lastDriver, &lastTeam, &lastCountry,
	)
	if err != nil {
		return nil, fmt.Errorf("error obteniendo último puesto: %v", err)
	}
	if !seenDrivers[lastDriver] {
		results = append(results, map[string]interface{}{
			"position": "Ultimo",
			"driver":   lastDriver,
			"team":     lastTeam,
			"country":  lastCountry,
		})
		seenDrivers[lastDriver] = true
	}

	// 4. Obtener el piloto con la mayor velocidad máxima
	maxSpeedQuery := `
        SELECT 
            d.first_name || ' ' || d.last_name as driver_name,
            MAX(l.st_speed) as max_speed
        FROM lap l
        JOIN driver d ON l.driver_number = d.driver_number
        WHERE l.session_key = ?
        GROUP BY l.driver_number
        ORDER BY max_speed DESC
        LIMIT 1`
	var (
		speedDriver string
		maxSpeed    float64
	)
	err = db.QueryRow(maxSpeedQuery, raceID).Scan(&speedDriver, &maxSpeed)
	if err != nil {
		return nil, fmt.Errorf("error obteniendo velocidad máxima: %v", err)
	}
	maxSpeedData := map[string]interface{}{
		"driver":    speedDriver,
		"speed_kmh": maxSpeed,
	}

	// 5. Obtener la vuelta más rápida (parseando strings de duración)
	fastestLapQuery := `
        SELECT 
            d.first_name || ' ' || d.last_name as driver_name,
            l.lap_duration as raw_total,
            l.duration_sector_1 as raw_s1,
            l.duration_sector_2 as raw_s2,
            l.duration_sector_3 as raw_s3
        FROM lap l
        JOIN driver d ON l.driver_number = d.driver_number
        WHERE l.session_key = ?
          AND l.lap_duration = (
              SELECT MIN(l2.lap_duration) 
              FROM lap l2 
              WHERE l2.session_key = ?
          )
        LIMIT 1;`
	var (
		fastDriver      string
		rawTotal, rawS1 string
		rawS2, rawS3    string
	)
	err = db.QueryRow(fastestLapQuery, raceID, raceID).Scan(
		&fastDriver, &rawTotal, &rawS1, &rawS2, &rawS3,
	)
	if err != nil {
		return nil, fmt.Errorf("error obteniendo vuelta más rápida: %v", err)
	}

	// helper: "MM:SS.SSSSSS" → segundos float64
	parseDur := func(raw string) (float64, error) {
		if strings.Contains(raw, ":") {
			// Caso formato MM:SS.SSS
			parts := strings.Split(raw, ":")
			if len(parts) != 2 {
				return 0, fmt.Errorf("formato inválido: %s", raw)
			}
			min, err := strconv.ParseFloat(parts[0], 64)
			if err != nil {
				return 0, err
			}
			sec, err := strconv.ParseFloat(parts[1], 64)
			if err != nil {
				return 0, err
			}
			return min*60 + sec, nil
		} else {
			// Caso sólo segundos
			sec, err := strconv.ParseFloat(raw, 64)
			if err != nil {
				return 0, err
			}
			return sec, nil
		}
	}

	totalSecs, err := parseDur(rawTotal)
	if err != nil {
		return nil, fmt.Errorf("parseDur total: %v", err)
	}
	s1, err := parseDur(rawS1)
	if err != nil {
		return nil, fmt.Errorf("parseDur s1: %v", err)
	}
	s2, err := parseDur(rawS2)
	if err != nil {
		return nil, fmt.Errorf("parseDur s2: %v", err)
	}
	s3, err := parseDur(rawS3)
	if err != nil {
		return nil, fmt.Errorf("parseDur s3: %v", err)
	}

	fastestLap := map[string]interface{}{
		"driver":     fastDriver,
		"total_time": totalSecs,
		"sector_1":   s1,
		"sector_2":   s2,
		"sector_3":   s3,
	}

	// 6. Construir respuesta final
	response := map[string]interface{}{
		"race_id":            sessionKey,
		"country_name":       countryName,
		"date_start":         dateStart.Format(time.RFC3339Nano),
		"year":               year,
		"circuit_short_name": circuitName,
		"results":            results,
		"fastest_lap":        fastestLap,
		"max_speed":          maxSpeedData,
	}

	return response, nil
}

//TODO: ARREGLAR LOS "POLE" Y QUE NO CONSIDERE TODOS LOS AÑOS

func GetSeasonSummary(db *sql.DB) (map[string]interface{}, error) {
	season := 2024

	// 1) Top 3 winners
	winnersQ := `
WITH LatestVictories AS (
    SELECT 
        p.driver_number,
        p.session_key,
        p.position,
        p.date,
        ROW_NUMBER() OVER (PARTITION BY p.session_key ORDER BY p.date DESC) AS rn
    FROM position p
    WHERE p.position = 1
)
SELECT 
    d.first_name || ' ' || d.last_name AS driver,
    d.team_name AS team,
    d.country_code AS country,
    COUNT(lv.session_key) AS wins
FROM LatestVictories lv
JOIN driver d ON lv.driver_number = d.driver_number
WHERE lv.rn = 1  -- Solo seleccionamos la carrera con la fecha más alta (última)
GROUP BY lv.driver_number
ORDER BY wins DESC
LIMIT 3
    `
	rows, err := db.Query(winnersQ, season)
	if err != nil {
		return nil, fmt.Errorf("query winners: %v", err)
	}
	defer rows.Close()

	topWinners := make([]map[string]interface{}, 0, 3)
	pos := 1
	for rows.Next() {
		var driver, team, country string
		var wins int
		if err := rows.Scan(&driver, &team, &country, &wins); err != nil {
			return nil, fmt.Errorf("scan winners: %v", err)
		}
		topWinners = append(topWinners, map[string]interface{}{
			"position": pos,
			"driver":   driver,
			"team":     team,
			"country":  country,
			"wins":     wins,
		})
		pos++
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// 2) Top 3 fastest laps
	fastestQ := `
WITH FastestLapsPerSession AS (
    SELECT 
        l.session_key,
        l.driver_number,
        l.lap_number,
        l.lap_duration,
        ROW_NUMBER() OVER (PARTITION BY l.session_key ORDER BY l.lap_duration ASC) AS fastest_rank
    FROM lap l
    JOIN session s ON l.session_key = s.session_key
    WHERE s.year = ?
),
DriverFastestLaps AS (
    SELECT 
        driver_number,
        COUNT(*) AS fastest_laps
    FROM FastestLapsPerSession
    WHERE fastest_rank = 1
    GROUP BY driver_number
)
SELECT 
    d.first_name || ' ' || d.last_name AS driver,
    d.team_name AS team,
    d.country_code AS country,
    COALESCE(dfl.fastest_laps, 0) AS fastest_laps
FROM driver d
LEFT JOIN DriverFastestLaps dfl ON d.driver_number = dfl.driver_number
ORDER BY fastest_laps DESC
LIMIT 3;
    `
	rowsF, err := db.Query(fastestQ, season)
	if err != nil {
		return nil, fmt.Errorf("query fastest laps: %v", err)
	}
	defer rowsF.Close()

	topFastest := make([]map[string]interface{}, 0, 3)
	pos = 1
	for rowsF.Next() {
		var driver, team, country string
		var flaps int
		if err := rowsF.Scan(&driver, &team, &country, &flaps); err != nil {
			return nil, fmt.Errorf("scan fastest laps: %v", err)
		}
		topFastest = append(topFastest, map[string]interface{}{
			"position":     pos,
			"driver":       driver,
			"team":         team,
			"country":      country,
			"fastest_laps": flaps,
		})
		pos++
	}
	if err := rowsF.Err(); err != nil {
		return nil, err
	}

	// 3) Top 3 pole positions (basado en el mejor tiempo en la vuelta 1)
	poleQ := `
WITH LatestVictories AS (
    SELECT 
        p.driver_number,
        p.session_key,
        p.position,
        p.date,
        ROW_NUMBER() OVER (PARTITION BY p.session_key ORDER BY p.date ASC) AS rn  -- Ordenar por fecha más baja
    FROM position p
    WHERE p.position = 1
)
SELECT 
    d.first_name || ' ' || d.last_name AS driver,
    d.team_name AS team,
    d.country_code AS country,
    COUNT(lv.session_key) AS wins
FROM LatestVictories lv
JOIN driver d ON lv.driver_number = d.driver_number
WHERE lv.rn = 1  -- Solo seleccionamos la carrera con la fecha más baja
GROUP BY lv.driver_number
ORDER BY wins DESC
LIMIT 3

    `
	rowsP, err := db.Query(poleQ, season)
	if err != nil {
		return nil, fmt.Errorf("query poles: %v", err)
	}
	defer rowsP.Close()

	topPoles := make([]map[string]interface{}, 0, 3)
	pos = 1
	for rowsP.Next() {
		var driver, team, country string
		var poles int
		if err := rowsP.Scan(&driver, &team, &country, &poles); err != nil {
			return nil, fmt.Errorf("scan poles: %v", err)
		}
		topPoles = append(topPoles, map[string]interface{}{
			"position": pos,
			"driver":   driver,
			"team":     team,
			"country":  country,
			"poles":    poles,
		})
		pos++
	}
	if err := rowsP.Err(); err != nil {
		return nil, err
	}

	// Armar JSON
	return map[string]interface{}{
		"season":               season,
		"top_3_winners":        topWinners,
		"top_3_fastest_laps":   topFastest,
		"top_3_pole_positions": topPoles,
	}, nil
}
