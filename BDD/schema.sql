CREATE TABLE IF NOT EXISTS driver(
    driver_number INT PRIMARY KEY,
    first_name VARCHAR(50) NOT NULL,
    last_name VARCHAR(50) NOT NULL,
    name_acronym VARCHAR(3) NOT NULL,
    team_name VARCHAR(50) NOT NULL,
    country_code CHAR(2) NOT NULL
);

CREATE TABLE IF NOT EXISTS session(
    session_key INT PRIMARY KEY,
    session_name VARCHAR(50) NOT NULL,
    session_type VARCHAR(50) NOT NULL,
    location VARCHAR(50) NOT NULL,
    country_name VARCHAR(50) NOT NULL,
    year INT NOT NULL,
    circuit_short_name VARCHAR(50) NOT NULL,
    date_start DATE NOT NULL
);

CREATE TABLE IF NOT EXISTS position(
    driver_number INT PRIMARY KEY,
    session_key INT NOT NULL 
    position INT NOT NULL,
    date DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS lap(
    driver_number INT NOT NULL,
    session_key INT NOT NULL,
    lap_number INT NOT NULL,
    lap_duration TIME NOT NULL,
    duration_sector_1 TIME NOT NULL,
    duration_sector_2 TIME NOT NULL,
    duration_sector_3 TIME NOT NULL,
    st_speed INT NOT NULL,
    date_start DATETIME NOT NULL,
);