package main


import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	//"os"
	"io"
	"log"
	//"sort"
	
    _ "github.com/mattn/go-sqlite3"
    //"github.com/gin-gonic/gin"
	"github.com/INF343-Tarea1/entidades"
)

func conectarBD() (*sql.DB, error) {
    return sql.Open("sqlite3", "./proxy.db")
}

func obtenerJSON[T any](url string) ([]T, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data []T
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	return data, nil
}

func insertarDriver(db *sql.DB, d entidades.Driver) error {
    _, err := db.Exec(`
        INSERT OR IGNORE INTO driver (
            driver_number, first_name, last_name, name_acronym, team_name, country_code
        ) VALUES (?, ?, ?, ?, ?, ?)`,
        d.DriverNumber, d.FirstName, d.LastName, d.NameAcronym, d.TeamName, d.CountryCode)
    return err
}

func insertarSession(db *sql.DB, s entidades.Session) error {
	_, err := db.Exec(`
		INSERT OR IGNORE INTO session (
			session_key, session_name, session_type, location, country_name, year, circuit_short_name, date_start
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		s.SessionKey, s.SessionName, s.SessionType, s.Location, s.CountryName, s.Year, s.CircuitShortName, s.DateStart)
	return err
}

func insertarPosition(db *sql.DB, p entidades.Position) error {
	_, err := db.Exec(`
		INSERT OR IGNORE INTO position (
			driver_number, session_key, position, date
		) VALUES (?, ?, ?, ?)`,
		p.DriverNumber, p.SessionKey, p.Position, p.Date)
	return err
}

func insertarLap(db *sql.DB, l entidades.Lap) error {
	_, err := db.Exec(`
		INSERT OR IGNORE INTO lap (
			driver_number, session_key, lap_number, lap_duration, duration_sector_1, duration_sector_2, duration_sector_3, st_speed, date_start
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		l.DriverNumber, l.SessionKey, l.LapNumber, l.LapDuration, l.DurationSector1, l.DurationSector2, l.DurationSector3, l.StSpeed, l.DateStart)
	return err
}


func main() {
    db, err := conectarBD()
    if err != nil {
        log.Fatal("Error conectando a la base de datos:", err)
    }
    defer db.Close()

    // 🔹 Rellenar tabla de drivers
    sessionDrivers := map[int][]int{
        9574: {1, 2, 3, 4, 10, 11, 14, 16, 18, 20, 22, 23, 24, 27, 31, 44, 55, 63, 77, 81},
        9636: {30, 50, 43},
    }

    for sessionKey, drivers := range sessionDrivers {
        todos, err := obtenerJSON[entidades.Driver](
            fmt.Sprintf("https://api.openf1.org/v1/drivers?session_key=%d", sessionKey))
        if err != nil {
            log.Println("Error al obtener drivers:", err)
            continue
        }

        // Crear un set de drivers válidos
        valid := map[int]bool{}
        for _, num := range drivers {
            valid[num] = true
        }

        for _, d := range todos {
            if valid[d.DriverNumber] {
                if err := insertarDriver(db, d); err != nil {
                    log.Println("Error insertando driver:", err)
                }
            }
        }
    }

    // 🔹 Rellenar tabla de carreras
    sesiones, err := obtenerJSON[entidades.Session](
        "https://api.openf1.org/v1/sessions?session_name=Race&year=2024")
    if err != nil {
        log.Fatal("Error obteniendo sesiones:", err)
    }
    for _, s := range sesiones {
        _, err := db.Exec(`
            INSERT OR IGNORE INTO session (
                session_key, session_name, session_type, location,
                country_name, year, circuit_short_name, date_start
            ) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
            s.SessionKey, s.SessionName, s.SessionType, s.Location,
            s.CountryName, s.Year, s.CircuitShortName, s.DateStart)
        if err != nil {
            log.Println("Error insertando sesión:", err)
        }
    }

    // 🔹 Obtener todas las session_key desde BD
    rows, err := db.Query("SELECT session_key FROM session")
    if err != nil {
        log.Fatal(err)
    }

    var claves []int
    for rows.Next() {
        var clave int
        rows.Scan(&clave)
        claves = append(claves, clave)
    }

    // 🔹 Rellenar tabla de posiciones
    for _, key := range claves {
        posiciones, err := obtenerJSON[entidades.Position](
            fmt.Sprintf("https://api.openf1.org/v1/position?session_key=%d", key))
        if err != nil {
            log.Println("Error posiciones:", err)
            continue
        }
        for _, p := range posiciones {
            _, err := db.Exec(`
                INSERT OR IGNORE INTO position (driver_number, session_key, position, date)
                VALUES (?, ?, ?, ?)`,
                p.DriverNumber, p.SessionKey, p.Position, p.Date)
            if err != nil {
                log.Println("Error insertando posición:", err)
            }
        }
    }

    // 🔹 Rellenar tabla de vueltas (laps)
    for _, key := range claves {
        vueltas, err := obtenerJSON[entidades.Lap](
            fmt.Sprintf("https://api.openf1.org/v1/laps?session_key=%d", key))
        if err != nil {
            log.Println("Error vueltas:", err)
            continue
        }

        for _, l := range vueltas {
			_, err := db.Exec(`
				INSERT OR IGNORE INTO lap (
					driver_number, session_key, lap_number, lap_duration,
					duration_sector_1, duration_sector_2, duration_sector_3,
					st_speed, date_start
				) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				l.DriverNumber, l.SessionKey, l.LapNumber, l.LapDuration,
				l.DurationSector1, l.DurationSector2, l.DurationSector3,
				l.StSpeed, l.DateStart)
			if err != nil {
				log.Println("Error insertando vuelta:", err)
			}
		}
    }

    fmt.Println("✓ Base de datos rellenada correctamente.")
}
