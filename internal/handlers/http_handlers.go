package handlers

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"lottery/internal/services"
)

const tenantCookieName = "lottery_tenant_name"
const tenantIDKey = "tenantID"

// HTTPHandler holds the dependencies for the HTTP handlers, like the lottery service.
type HTTPHandler struct {
	service   *services.LotteryService
	templates *template.Template
}

// NewHTTPHandler creates a new HTTPHandler.
func NewHTTPHandler(service *services.LotteryService, templates *template.Template) *HTTPHandler {
	return &HTTPHandler{
		service:   service,
		templates: templates,
	}
}

// TenantMiddleware identifies the tenant for each request.
func (h *HTTPHandler) TenantMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantName, err := c.Cookie(tenantCookieName)
		if err != nil {
			// If cookie is not set, use a default name based on IP
			tenantName = fmt.Sprintf("user-%s", c.ClientIP())
		}

		// Combine name and IP for a unique tenant ID
		tenantID := fmt.Sprintf("%s-%s", tenantName, c.ClientIP())
		c.Set(tenantIDKey, tenantID)

		// This call also updates the LastActivity timestamp for the session
		_ = h.service.GetPrizes(tenantID) // A simple way to ensure session exists and is active

		c.Next()
	}
}

// renderPage is a helper for two-step template rendering.
func (h *HTTPHandler) renderPage(c *gin.Context, pageData gin.H, contentTmpl string) {
	// Automatically add current tenant name to all page renders
	currentTenant, _ := c.Cookie(tenantCookieName)
	pageData["CurrentTenant"] = currentTenant

	buf := new(bytes.Buffer)
	err := h.templates.ExecuteTemplate(buf, contentTmpl, pageData)
	if err != nil {
		log.Printf("Error executing content template %s: %v", contentTmpl, err)
		c.String(http.StatusInternalServerError, "Template rendering error")
		return
	}

	pageData["PageContent"] = template.HTML(buf.String())

	err = h.templates.ExecuteTemplate(c.Writer, "layout.html", pageData)
	if err != nil {
		log.Printf("Error executing layout template: %v", err)
		c.String(http.StatusInternalServerError, "Template rendering error")
	}
}

// RegisterPublicRoutes registers routes that do not require tenant identification.
func (h *HTTPHandler) RegisterPublicRoutes(router *gin.Engine) {
	router.POST("/set-tenant", h.SetTenant)
	router.GET("/clear-tenant", h.ClearTenant) // New route
}

// RegisterTenantRoutes registers routes that require the tenant middleware.
func (h *HTTPHandler) RegisterTenantRoutes(router *gin.RouterGroup) {
	router.GET("/", h.ShowIndex)
	router.GET("/prizes", h.ShowPrizesPage)
	router.POST("/prizes", h.AddPrize)
	router.POST("/upload-prizes-csv", h.UploadPrizesCSV)
	router.GET("/participants", h.ShowParticipantsPage)
	router.POST("/participants", h.AddParticipant)
	router.POST("/upload-participants-csv", h.UploadParticipantsCSV)
	router.GET("/lottery", h.ShowLotteryPage)
	router.POST("/draw", h.PerformDraw)
	router.GET("/prizes/list", h.GetPrizeListPartial)
	router.GET("/export-results-csv", h.ExportResultsCSV)
}

// SetTenant handles setting the tenant name cookie.
func (h *HTTPHandler) SetTenant(c *gin.Context) {
	tenantName := c.PostForm("tenantName")
	if tenantName != "" {
		// Set cookie for a year
		c.SetCookie(tenantCookieName, tenantName, 3600*24*365, "/", "", false, true)
	}
	c.Redirect(http.StatusFound, "/")
}

// ClearTenant clears the user's session and cookie, then redirects to home.
func (h *HTTPHandler) ClearTenant(c *gin.Context) {
	// This handler is on a public route, so it needs to construct the tenantID itself
	// before clearing the cookie.
	tenantName, err := c.Cookie(tenantCookieName)
	if err == nil && tenantName != "" {
		// If the cookie exists, construct the tenantID and clear the session data.
		tenantID := fmt.Sprintf("%s-%s", tenantName, c.ClientIP())
		h.service.ClearSession(tenantID)
	}

	// Clear the cookie by setting its max age to -1
	c.SetCookie(tenantCookieName, "", -1, "/", "", false, true)

	c.Redirect(http.StatusFound, "/")
}

// ShowIndex handles the request for the home page.
func (h *HTTPHandler) ShowIndex(c *gin.Context) {
	h.renderPage(c, gin.H{"title": "首頁"}, "index.html")
}

// ShowPrizesPage handles the request for the prize setting page.
func (h *HTTPHandler) ShowPrizesPage(c *gin.Context) {
	tenantID := c.GetString(tenantIDKey)
	data := gin.H{
		"title":  "獎項設定",
		"Prizes": h.service.GetPrizes(tenantID),
	}
	h.renderPage(c, data, "prize_setting.html")
}

// AddPrize handles the form submission for adding a new prize.
func (h *HTTPHandler) AddPrize(c *gin.Context) {
	tenantID := c.GetString(tenantIDKey)
	prizeName := c.PostForm("prizeName")
	itemName := c.PostForm("itemName")
	quantityStr := c.PostForm("quantity")
	drawFromAllStr := c.PostForm("drawFromAll")

	quantity, err := strconv.Atoi(quantityStr)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid quantity")
		return
	}
	drawAllFlag := drawFromAllStr == "true"

	h.service.AddPrize(tenantID, prizeName, itemName, quantity, drawAllFlag)

	data := gin.H{"Prizes": h.service.GetPrizes(tenantID)}
	if err := h.templates.ExecuteTemplate(c.Writer, "prize_list_container.html", data); err != nil {
		log.Printf("Error executing template: %v", err)
	}
}

// UploadPrizesCSV handles the CSV upload for prizes.
func (h *HTTPHandler) UploadPrizesCSV(c *gin.Context) {
	tenantID := c.GetString(tenantIDKey)
	file, _, err := c.Request.FormFile("prizeCSV")
	if err != nil {
		c.String(http.StatusBadRequest, "Error retrieving file: %v", err)
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			c.String(http.StatusInternalServerError, "Error reading CSV: %v", err)
			return
		}
		if len(record) != 4 {
			log.Printf("Skipping malformed CSV record: %v", record)
			continue
		}
		prizeName, itemName := record[0], record[1]
		quantity, _ := strconv.Atoi(record[2])
		drawFromAll, _ := strconv.ParseBool(record[3])
		h.service.AddPrize(tenantID, prizeName, itemName, quantity, drawFromAll)
	}

	data := gin.H{"Prizes": h.service.GetPrizes(tenantID)}
	if err := h.templates.ExecuteTemplate(c.Writer, "prize_list_container.html", data); err != nil {
		log.Printf("Error executing template: %v", err)
	}
}

// ShowParticipantsPage handles the request for the participant setting page.
func (h *HTTPHandler) ShowParticipantsPage(c *gin.Context) {
	tenantID := c.GetString(tenantIDKey)
	data := gin.H{
		"title":        "參與者設定",
		"Participants": h.service.GetParticipants(tenantID),
	}
	h.renderPage(c, data, "participant_setting.html")
}

// AddParticipant handles the form submission for adding a new participant.
func (h *HTTPHandler) AddParticipant(c *gin.Context) {
	tenantID := c.GetString(tenantIDKey)
	participantID := c.PostForm("participantID")
	participantName := c.PostForm("participantName")

	if participantID == "" || participantName == "" {
		c.String(http.StatusBadRequest, "Participant ID and Name cannot be empty")
		return
	}

	h.service.AddParticipant(tenantID, participantID, participantName)

	data := gin.H{"Participants": h.service.GetParticipants(tenantID)}
	if err := h.templates.ExecuteTemplate(c.Writer, "participant_list_container.html", data); err != nil {
		log.Printf("Error executing template: %v", err)
	}
}

// UploadParticipantsCSV handles the CSV upload for participants.
func (h *HTTPHandler) UploadParticipantsCSV(c *gin.Context) {
	tenantID := c.GetString(tenantIDKey)
	file, _, err := c.Request.FormFile("participantCSV")
	if err != nil {
		c.String(http.StatusBadRequest, "Error retrieving file: %v", err)
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			c.String(http.StatusInternalServerError, "Error reading CSV: %v", err)
			return
		}
		if len(record) != 2 {
			log.Printf("Skipping malformed participant CSV record: %v", record)
			continue
		}
		h.service.AddParticipant(tenantID, record[0], record[1])
	}

	data := gin.H{"Participants": h.service.GetParticipants(tenantID)}
	if err := h.templates.ExecuteTemplate(c.Writer, "participant_list_container.html", data); err != nil {
		log.Printf("Error executing template: %v", err)
	}
}

// ShowLotteryPage handles the request for the main lottery drawing page.
func (h *HTTPHandler) ShowLotteryPage(c *gin.Context) {
	tenantID := c.GetString(tenantIDKey)
	data := gin.H{
		"title":          "抽獎介面",
		"Prizes":         h.service.GetPrizes(tenantID),
		"Participants":   h.service.GetParticipants(tenantID),
		"LotteryResults": h.service.GetLotteryResults(tenantID),
	}
	h.renderPage(c, data, "lottery_interface.html")
}

// PerformDraw handles the request to draw a winner for a prize.
func (h *HTTPHandler) PerformDraw(c *gin.Context) {
	tenantID := c.GetString(tenantIDKey)
	prizeName := c.PostForm("prizeName")
	if prizeName == "" {
		c.String(http.StatusBadRequest, "Please select a prize.")
		return
	}

	result, err := h.service.Draw(tenantID, prizeName)
	if err != nil {
		c.String(http.StatusOK, "<p>%s</p>", err.Error())
		return
	}

	data := gin.H{
		"Result": result,
		"Prizes": h.service.GetPrizes(tenantID),
	}

	if err := h.templates.ExecuteTemplate(c.Writer, "lottery_draw_response.html", data); err != nil {
		log.Printf("Error executing template: %v", err)
	}
}

// GetPrizeListPartial returns the HTML partial for the prize list body.
func (h *HTTPHandler) GetPrizeListPartial(c *gin.Context) {
	tenantID := c.GetString(tenantIDKey)
	if err := h.templates.ExecuteTemplate(c.Writer, "prize_list_table_body.html", h.service.GetPrizes(tenantID)); err != nil {
		log.Printf("Error executing template: %v", err)
	}
}

// ExportResultsCSV handles the request to download the lottery results as a CSV file.
func (h *HTTPHandler) ExportResultsCSV(c *gin.Context) {
	tenantID := c.GetString(tenantIDKey)
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment;filename=lottery_results.csv")

	c.Writer.Write([]byte("\xef\xbb\xbf"))
	w := csv.NewWriter(c.Writer)

	if err := w.Write([]string{"獎項名稱", "員工編號", "員工姓名", "獎品名稱"}); err != nil {
		log.Printf("Error writing CSV header: %v", err)
		return
	}

	for _, result := range h.service.GetLotteryResults(tenantID) {
		row := []string{result.PrizeName, result.WinnerID, result.WinnerName, result.PrizeItem}
		if err := w.Write(row); err != nil {
			log.Printf("Error writing CSV row: %v", err)
			return
		}
	}
	w.Flush()

	if err := w.Error(); err != nil {
		log.Printf("Error flushing CSV writer: %v", err)
	}
}