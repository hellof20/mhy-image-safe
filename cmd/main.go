package main

import (
	"log"

	"example/mhy-image-safe/internal/config"
	"example/mhy-image-safe/internal/service"
)

func main() {
	cfg := config.Init()

	analyzer := service.NewImageAnalyzer(cfg.Client, cfg.Model)
	processor := service.NewImageProcessor(analyzer, cfg.Concurrent)

	if err := processor.ProcessImages(cfg.ImageDir); err != nil {
		log.Fatal(err)
	}
	log.Println("BatchInvoke completed successfully.")
}
