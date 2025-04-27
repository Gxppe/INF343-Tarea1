package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	//"os"
	"io"
	"log"
	_ "github.com/mattn/go-sqlite3"
	//"github.com/gin-gonic/gin"
	"github.com/INF343-Tarea1/entidades"
	"github.com/INF343-Tarea1/vista"
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

func check(num float64) interface{} {
	if num == 0 {
		return nil
	}
	return num
}

func rellenarBD() {
	db, err := conectarBD()
	if err != nil {
		log.Fatal("Error conectando a la base de datos:", err)
	}
	defer db.Close()

	// Drivers
	sessionDrivers := map[int][]int{
		9574: {1, 2, 3, 4, 10, 11, 14, 16, 18, 20, 22, 23, 24, 27, 31, 44, 55, 63, 77, 81},
		9636: {30, 50, 43},
	}

	for sessionKey, drivers := range sessionDrivers {
		// Obtener todos los drivers de la API
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
		fmt.Println("Drivers correctamente insertados para la sesión:", sessionKey)
	}

	// Rellenar tabla de sesiones (o carreras)
	fmt.Println("Obteniendo sesiones de la API...")
	sesiones, err := obtenerJSON[entidades.Session]("https://api.openf1.org/v1/sessions?session_name=Race&year=2024")
	if err != nil {
		log.Fatal("Error obteniendo sesiones:", err)
	}
	fmt.Println("✓ Sesiones obtenidas de la API, total:", len(sesiones))
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

	// Obtiene las sesiones
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

	// Rellenar tabla de posiciones (positions)
	for _, key := range claves {
		posiciones, err := obtenerJSON[entidades.Position](
			fmt.Sprintf("https://api.openf1.org/v1/position?session_key=%d", key))
		if err != nil {
			log.Println("Error posiciones:", err)
			continue
		}
		for _, p := range posiciones {
			if p.DriverNumber == 61 {
				continue
			}
			_, err := db.Exec(`
                INSERT OR IGNORE INTO position (driver_number, session_key, position, date)
                VALUES (?, ?, ?, ?)`,
				p.DriverNumber, p.SessionKey, p.Position, p.Date)
			if err != nil {
				log.Println("Error insertando posición:", err)
			}
		}
	}

	// Rellenar tabla de vueltas (laps)
	for _, key := range claves {
		vueltas, err := obtenerJSON[entidades.Lap](
			fmt.Sprintf("https://api.openf1.org/v1/laps?session_key=%d", key))
		if err != nil {
			log.Println("Error vueltas:", err)
			continue
		}
		for _, l := range vueltas {
			if l.DriverNumber == 61 {
				continue
			}
			// Revisar si hay valores nulos
			check(float64(l.LapNumber))
			check(l.LapDuration)
			check(l.DurationSector1)
			check(l.DurationSector2)
			check(l.DurationSector3)
			check(l.StSpeed)

			_, err := db.Exec(`
				INSERT OR IGNORE INTO lap (
					driver_number, session_key, lap_number, lap_duration,
					duration_sector_1, duration_sector_2, duration_sector_3,
					st_speed, date_start
				) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				l.DriverNumber,
				l.SessionKey,
				l.LapNumber,
				check(l.LapDuration),
				check(l.DurationSector1),
				check(l.DurationSector2),
				check(l.DurationSector3),
				check(l.StSpeed),
				l.DateStart)
			if err != nil {
				log.Println("Error insertando vuelta:", err)
			}
		}
	}

	fmt.Println("✓ Base de datos rellenada correctamente.")
}

func main(){
	rellenarBD()
	for {
		fmt.Println("\n Menu")
		fmt.Println("1. Ver corredores")
		fmt.Println("2. Ver detalle de corredor")
		fmt.Println("3. Ver carreras")
		fmt.Println("4. Ver detalle de carrera")
		fmt.Println("5. Resumen de temporada")
		fmt.Println("6. Salir")
		fmt.Println("Seleccione una opción:")
		var opcion int
		fmt.Scanln(&opcion)

		switch opcion {
		case 1:
			vista.VerCorredores()
		case 2:
			vista.VerDetalleCorredor()
		case 3:
			vista.VerCarreras()
		case 4:
			vista.VerDetalleCarrera()
		case 5:
			vista.ResumenTemporada()
		case 6:
			fmt.Println("Saliendo...")
			return
		default:
			fmt.Println("Opción no válida. Intente nuevamente.")
		}
	}
}