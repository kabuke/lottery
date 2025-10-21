package main

import (
	"embed"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"lottery/internal/handlers"
	"lottery/internal/services"
)

//go:embed all:templates
var templateFS embed.FS

//go:embed all:assets
var assetsFS embed.FS

func main() {
	// 1. Initialize the Lottery Service
	lotteryService := services.NewLotteryService()

	// 2. Load HTML templates from the embedded filesystem.
	templates, err := template.ParseFS(templateFS, "templates/*.html")
	if err != nil {
		log.Fatalf("Failed to parse templates: %v", err)
	}

	// 3. Initialize the HTTP Handler
	httpHandler := handlers.NewHTTPHandler(lotteryService, templates)

	// 4. Set up the Gin router
	r := gin.Default()

	// 5. Serve static files from the embedded filesystem.
	assetsSubFS, err := fs.Sub(assetsFS, "assets")
	if err != nil {
		log.Fatalf("Failed to create assets sub-filesystem: %v", err)
	}
	r.StaticFS("/assets", http.FS(assetsSubFS))

	// 6. Register public routes (before middleware)
	httpHandler.RegisterPublicRoutes(r)

	// 7. Group routes that require tenant identification and apply middleware
	tenantRoutes := r.Group("/")
	tenantRoutes.Use(httpHandler.TenantMiddleware())
	httpHandler.RegisterTenantRoutes(tenantRoutes)

	// 8. Start the background janitor to clean up inactive sessions
	go func() {
		for {
			time.Sleep(10 * time.Minute) // Run every 10 minutes
			lotteryService.CleanUpInactiveSessions()
			log.Println("Performed cleanup of inactive sessions.")
		}
	}()

	// 9. Run the server
	log.Println("Server starting on http://localhost:8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}