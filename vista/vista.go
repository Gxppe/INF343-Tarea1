package vista

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

func VerCorredores() {
	resp, err := http.Get("http://localhost:8080/api/corredor")
	if err != nil {
		log.Fatal("Error al obtener corredores:", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("Error al leer respuesta:", err)
	}
	var corredores []map[string]interface{}
	if err := json.Unmarshal(body, &corredores); err != nil {
		log.Fatal("Error al deserializar respuesta:", err)
	}
	fmt.Println("---------------------------------------------------------------------------")
	fmt.Println("| #  | Nombre        | Apellido      | N Piloto | Equipo           | Pais |")
	fmt.Println("---------------------------------------------------------------------------")
	for i, corredor := range corredores {
		fmt.Printf("| %-2d | %-13s | %-13s | %-8d | %-16s | %-4s |\n",
			i+1,
			corredor["first_name"],
			corredor["last_name"],
			int(corredor["driver_number"].(float64)),
			corredor["team_name"],
			corredor["country_code"],
		)
	}
	fmt.Println("---------------------------------------------------------------------------")
	
}

func boolAfirmativo(v interface{}) string {
	if b, ok := v.(bool); ok && b {
		return "Sí"
	}
	return "No"
}

func VerDetalleCorredor() {
	fmt.Println("Ingrese el numero de piloto:")
	var numero int
	fmt.Scanln(&numero)

	resp, err := http.Get(fmt.Sprintf("http://localhost:8080/api/corredor/detalle/%d", numero))
	if err != nil {
		log.Fatal("Error al obtener detalles del corredor:", err)
	}
	defer resp.Body.Close()

	var detalle map[string]interface{}
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &detalle); err != nil {
		log.Fatal("Error al deserializar respuesta:", err)
	}

	// Primero revisar si el corredor existe
	if detalle["driver_id"] == nil {
		fmt.Println("El piloto no existe")
		return
	}

	fmt.Println("-----------------------------------------------------------------------------------")
	fmt.Println("| #  | Carrera | Pos Final | Vuelta rápida | Velocidad max | Menor tiempo vuelta  |")
	fmt.Println("-----------------------------------------------------------------------------------")
	for i, c := range detalle["race_results"].([]interface{}) {
		carrera := c.(map[string]interface{})
		fmt.Printf("| %-2d | %-7s | %-9.0f | %-13s | %-13.0f | %-20.3f |\n",
			i+1,
			carrera["race"].(string),             // string
			carrera["position"].(float64),         // float64
			boolAfirmativo(carrera["fastest_lap"]), // string ("Sí"/"No")
			carrera["max_speed"].(float64),         // float64
			carrera["best_lap_duration"].(float64), // float64
		)
	}
	fmt.Println("-----------------------------------------------------------------------------------")	
	
	performance := detalle["performance_summary"].(map[string]interface{})

	fmt.Println("-------------------------------------------------")
	fmt.Println("| Resumen del desempeno del piloto              |")
	fmt.Println("-------------------------------------------------")
	fmt.Printf("| %-30s | %-12v |\n", "Carreras ganadas", int(performance["wins"].(float64)))
	fmt.Printf("| %-30s | %-12v |\n", "Veces en el top 3", int(performance["top_3_finishes"].(float64)))
	fmt.Printf("| %-30s | %-7.0f km/h |\n", "Velocidad maxima alcanzada", performance["max_speed"].(float64))
	fmt.Println("-------------------------------------------------")


}

func formatearFecha(iso string) string {
	t, err := time.Parse(time.RFC3339, iso)
	if err != nil {
		return iso
	}
	return t.Format("02-01-2006")
}

func VerCarreras() {
	resp, err := http.Get("http://localhost:8080/api/carrera")
	if err != nil {
		log.Fatal("Error al obtener carreras:", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("Error al leer respuesta:", err)
	}
	var carreras []map[string]interface{}
	if err := json.Unmarshal(body, &carreras); err != nil {
		log.Fatal("Error al deserializar respuesta:", err)
	}
	fmt.Println("-----------------------------------------------------------------------------------------")
	fmt.Println("| #  | ID carrera | País           | Fecha       | Año  | Circuito           |")
	fmt.Println("-----------------------------------------------------------------------------------------")
	
	for i, c := range carreras {
		fecha := formatearFecha(c["date_start"].(string))
		fmt.Printf("| %-2d | %-10v | %-14v | %-10s | %-4v | %-18v |\n",
			i+1,
			c["session_key"],
			c["country_name"],
			fecha,
			c["year"],
			c["circuit_short_name"],
		)
	}
	fmt.Println("-----------------------------------------------------------------------------------------")
	

}

func VerDetalleCarrera() {
	fmt.Println("Ingrese el ID de la carrera:")
	var id int
	fmt.Scanln(&id)

	resp, err := http.Get(fmt.Sprintf("http://localhost:8080/api/carrera/detalle/%d", id))
	if err != nil {
		log.Fatal("Error al obtener detalles de la carrera:", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Println("Error:", resp.Status)
		return
	}

	var detalle map[string]interface{}
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &detalle); err != nil {
		log.Fatal("Error al deserializar respuesta:", err)
	}

	// Primero revisar si la carrera existe
	if detalle["race_id"] == nil {
		fmt.Println("La carrera no existe")
		return
	}

	fmt.Println("\n| Resultados                                               |")
	fmt.Println("------------------------------------------------------------")
	fmt.Println("| Posición | Piloto              | Equipo        | País     |")
	fmt.Println("------------------------------------------------------------")
	for _, r := range detalle["results"].([]interface{}) {
		d := r.(map[string]interface{})
		fmt.Printf("| %-8v | %-18v | %-13v | %-8v |\n",
			d["position"], d["driver"], d["team"], d["country"])
	}

	v := detalle["fastest_lap"].(map[string]interface{})
	fmt.Println("\n| Vuelta más rápida                                        |")
	fmt.Println("------------------------------------------------------------")
	fmt.Println("| Piloto         | Tiempo Total | Sector 1 | Sector 2 | Sector 3 |")
	fmt.Println("------------------------------------------------------------")
	fmt.Printf("| %-14v | %-12.3f | %-8.3f | %-8.3f | %-8.3f |\n",
		v["driver"], v["total_time"], v["sector_1"], v["sector_2"], v["sector_3"])

	s := detalle["max_speed"].(map[string]interface{})
	fmt.Println("\n| Velocidad máxima alcanzada                               |")
	fmt.Println("------------------------------------------------------------")
	fmt.Println("| Piloto         | Velocidad (km/h)                        |")
	fmt.Println("------------------------------------------------------------")
	fmt.Printf("| %-14v | %-29.1f |\n", s["driver"], s["speed_kmh"])
}

func ResumenTemporada() {
	resp, err := http.Get("http://localhost:8080/api/temporada/resumen/")
	if err != nil {
		log.Fatal("Error al obtener el resumen de temporada:", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Println("Error:", resp.Status)
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("Error al leer la respuesta:", err)
	}

	var resumen map[string]interface{}
	if err := json.Unmarshal(body, &resumen); err != nil {
		log.Fatal("Error al deserializar resumen:", err)
	}

	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Println(" Top 3 Pilotos con más Victorias - Temporada 2024")
	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Printf("| %-9s | %-20s | %-20s | %-5s | %-10s |\n", "Posición", "Piloto", "Equipo", "País", "Victorias")
	fmt.Println("--------------------------------------------------------------------------------")
	for _, item := range resumen["top_3_winners"].([]interface{}) {
		p := item.(map[string]interface{})
		fmt.Printf("| %-9.0f | %-20s | %-20s | %-5s | %-10.0f |\n",
			p["position"], p["driver"], p["team"], p["country"], p["wins"])
	}
	fmt.Println("--------------------------------------------------------------------------------")

	// Top 3 Pilotos con más Vueltas Rápidas
	fmt.Println()
	fmt.Println("-------------------------------------------------------------------------------------")
	fmt.Println(" Top 3 Pilotos con más Vueltas Rápidas - Temporada 2024")
	fmt.Println("-------------------------------------------------------------------------------------")
	fmt.Printf("| %-9s | %-20s | %-20s | %-5s | %-15s |\n", "Posición", "Piloto", "Equipo", "País", "Vueltas Rápidas")
	fmt.Println("-------------------------------------------------------------------------------------")
	for _, item := range resumen["top_3_fastest_laps"].([]interface{}) {
		p := item.(map[string]interface{})
		fmt.Printf("| %-9.0f | %-20s | %-20s | %-5s | %-15.0f |\n",
			p["position"], p["driver"], p["team"], p["country"], p["fastest_laps"])
	}
	fmt.Println("-------------------------------------------------------------------------------------")

	// Top 3 Pilotos con más Pole Positions
	fmt.Println()
	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Println(" Top 3 Pilotos con más Pole Positions - Temporada 2024")
	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Printf("| %-9s | %-20s | %-20s | %-5s | %-10s |\n", "Posición", "Piloto", "Equipo", "País", "Poles")
	fmt.Println("--------------------------------------------------------------------------------")
	for _, item := range resumen["top_3_pole_positions"].([]interface{}) {
		p := item.(map[string]interface{})
		fmt.Printf("| %-9.0f | %-20s | %-20s | %-5s | %-10.0f |\n",
			p["position"], p["driver"], p["team"], p["country"], p["poles"])
	}
	fmt.Println("--------------------------------------------------------------------------------")
}

