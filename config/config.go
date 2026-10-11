package config

import (
	"github.com/joho/godotenv"
	"log"
	"os"
)

type Config struct {
	DbUrl string
}

func NewConfig() Config {
	// Load env variables for use
	if err := godotenv.Load(); err != nil {
		log.Println("error loading system environment variables")
	}
	// Fetch database connection string+
	dbUrl := fetchEnvVariable("DBURL")

	return Config{
		DbUrl: dbUrl,
	}
}

func fetchEnvVariable(varb string) string {
	if varb == "" {
		return ""
	}
	return os.Getenv(varb)
}
