package config

import (
	"example/mhy-image-safe/internal/gemini"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"google.golang.org/genai"
)

type Config struct {
	Location    string
	Model       string
	Project     string
	Temperature float64
	Concurrent  int
	ImageDir    string
	Client      *genai.Client
	OutputDir   string
}

func Init() *Config {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}
	concurrent, _ := strconv.Atoi(os.Getenv("CONCURRENT"))
	temperature, _ := strconv.ParseFloat(os.Getenv("TEMPERATURE"), 64)

	cfg := &Config{
		Location:    os.Getenv("LOCATION"),
		Model:       os.Getenv("MODEL"),
		Project:     os.Getenv("PROJECT"),
		Temperature: temperature,
		Concurrent:  concurrent,
		ImageDir:    os.Getenv("IMAGE_DIR"),
		OutputDir:   "result.csv",
	}

	cfg.Client = gemini.SetupClient(cfg.Project, cfg.Location)
	return cfg
}
