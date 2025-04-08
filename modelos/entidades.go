package modelos

type Driver struct {
    DriverNumber int `json:"driver_number"`
    FirstName    string `json:"first_name"`
    LastName     string `json:"last_name"`
    NameAcronym  string `json:"name_acronym"`
    TeamName     string `json:"team_name"`
    CountryCode  string `json:"country_code"`
}

type Session struct {
    SessionKey       int    `json:"session_key"`
    SessionName      string `json:"session_name"`
    SessionType      string `json:"session_type"`
    Location         string `json:"location"`
    CountryName      string `json:"country_name"`
    Year             int    `json:"year"`
    CircuitShortName string `json:"circuit_short_name"`
    DateStart        string `json:"date_start"`
}

type Position struct {
	DriverNumber	int 	`json:"driver_number"`
	SessionKey 		int 	`json:"session_key"`
	Position 		int 	`json:"position"`
	Date 			string 	`json:"date"`
}

type Laps struct {
	DriverNumber 	int 	`json:"driver_number"`
	SessionKey 		int 	`json:"session_key"`
	LapNumber 		int 	`json:"lap_number"`
	LapDuration 	string 	`json:"lap_duration"`
	DurationSector1 string 	`json:"duration_sector_1"`
	DurationSector2 string 	`json:"duration_sector_2"`
	DurationSector3 string 	`json:"duration_sector_3"`
	StSpeed 		float64 `json:"st_speed"`
	DateStart		string 	`json:"date_start"`
}