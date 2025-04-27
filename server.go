package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strconv"

	handler "github.com/INF343-Tarea1/handlers"
	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
)

// Conectar a la base de datos
func connectDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", "./proxy.db")
	if err != nil {
		return nil, err
	}
	// Verificar que la conexión realmente funciona
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}

func main() {
	// Crear el router de Gin
	r := gin.Default()

	// Conectar a la base de datos
	db, err := connectDB()
	if err != nil {
		fmt.Println("Error al conectar a la base de datos:", err)
		return
	}
	defer db.Close()
	fmt.Println("Conexión a la base de datos establecida correctamente")

	// Endpoint para obtener todos los corredores
	r.GET("/api/corredor", func(c *gin.Context) {
		// Obtener lista de corredores desde la base de datos
		drivers, err := handler.Get_Drivers(db)
		if err != nil {
			// Registrar el error para debugging
			log.Printf("Error al obtener corredores: %v", err)

			// Retornar error al cliente
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "internal_server_error",
				"message": "No se pudo obtener la lista de corredores",
			})
			return
		}

		// Si no hay corredores, retornar array vacío
		if len(drivers) == 0 {
			c.JSON(http.StatusOK, []interface{}{})
			return
		}

		// Retornar lista de corredores
		c.JSON(http.StatusOK, drivers)
	})

	// Endpoint para obtener detalles de un corredor específico
	r.GET("/api/corredor/detalle/:id", func(c *gin.Context) {
		driverIDStr := c.Param("id")
		driverID, err := strconv.Atoi(driverIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "invalid_driver_id",
				"message": "El ID debe ser un número entero",
			})
			return
		}

		driverDetails, err := handler.GetDriverDetails(db, driverID)
		if err != nil {
			if err.Error() == "driver not found" {
				c.JSON(http.StatusNotFound, gin.H{
					"error":   "driver_not_found",
					"message": fmt.Sprintf("No se encontró el conductor con ID %d", driverID),
				})
			} else {
				log.Printf("Error getting driver details: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{
					"error":   "internal_server_error",
					"message": "Error al obtener detalles del conductor",
				})
			}
			return
		}

		c.JSON(http.StatusOK, driverDetails)
	})
	// Endpoint para obtener todas las carreras
	r.GET("/api/carrera", func(c *gin.Context) {
		races, err := handler.Get_Races(db)
		if err != nil {
			c.JSON(500, gin.H{"error": "Error al obtener las carreras"})
			return
		}
		c.JSON(200, races)
	})
	// Endpoint para obtener detalles de una carrera específica
	r.GET("/api/carrera/detalle/:id", func(c *gin.Context) {
		raceID, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ID de carrera inválido"})
			return
		}

		raceDetails, err := handler.GetRaceDetails(db, raceID)
		if err != nil {
			log.Printf("Error obteniendo detalles de carrera: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al obtener detalles de la carrera"})
			return
		}

		c.JSON(http.StatusOK, raceDetails)
	})

	r.GET("/api/temporada/resumen", func(c *gin.Context) {
		if err != nil {
			c.JSON(400, gin.H{"error": "season inválido"})
			return
		}
		resp, err := handler.GetSeasonSummary(db)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, resp)
	})

	// Iniciar el servidor en el puerto 8080
	r.Run(":8080")
}

