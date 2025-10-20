package main

import (
	"html/template"
	"log"

	"github.com/gin-gonic/gin"
	"lottery/internal/handlers"
	"lottery/internal/services"
)

func main() {
	// 1. Initialize the Lottery Service
	lotteryService := services.NewLotteryService()

	// 2. Load all HTML templates into a single template set.
	// The template names will be their file names.
	templates, err := template.ParseGlob("internal/templates/*.html")
	if err != nil {
		log.Fatalf("Failed to parse templates: %v", err)
	}

	// 3. Initialize the HTTP Handler
	httpHandler := handlers.NewHTTPHandler(lotteryService, templates)

	// 4. Set up the Gin router
	r := gin.Default()

	// 5. Register routes
	httpHandler.RegisterRoutes(r)

	// Serve static files
	r.Static("/assets", "./web/assets")

	// 6. Run the server
	log.Println("Server starting on http://localhost:8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}
