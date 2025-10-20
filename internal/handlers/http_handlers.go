package handlers

import (
	"bytes"
	"encoding/csv"
	"html/template"
	"io"
	"net/http"
	"strconv"

	"lottery/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/logger"
)

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

// renderPage is a helper to perform a two-step template rendering.
// It first executes the content template into a buffer, then executes the main
// layout template, passing the rendered content as a variable.
func (h *HTTPHandler) renderPage(c *gin.Context, pageData gin.H, contentTmpl string) {
	// Step 1: Render the specific page content into a buffer.
	buf := new(bytes.Buffer)
	err := h.templates.ExecuteTemplate(buf, contentTmpl, pageData)
	if err != nil {
		logger.Infof("Error executing content template %s: %v", contentTmpl, err)
		c.String(http.StatusInternalServerError, "Template rendering error")
		return
	}

	// Step 2: Add the rendered content to the main data map and render the layout.
	pageData["PageContent"] = template.HTML(buf.String())

	err = h.templates.ExecuteTemplate(c.Writer, "layout.html", pageData)
	if err != nil {
		logger.Infof("Error executing layout template: %v", err)
		c.String(http.StatusInternalServerError, "Template rendering error")
	}
}

// RegisterRoutes registers all the application routes.
func (h *HTTPHandler) RegisterRoutes(router *gin.Engine) {
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

// ShowIndex handles the request for the home page.
func (h *HTTPHandler) ShowIndex(c *gin.Context) {
	h.renderPage(c, gin.H{"title": "首頁"}, "index.html")
}

// ShowPrizesPage handles the request for the prize setting page.
func (h *HTTPHandler) ShowPrizesPage(c *gin.Context) {
	data := gin.H{
		"title":  "獎項設定",
		"Prizes": h.service.Prizes,
	}
	h.renderPage(c, data, "prize_setting.html")
}

// AddPrize handles the form submission for adding a new prize.
func (h *HTTPHandler) AddPrize(c *gin.Context) {
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

	h.service.AddPrize(prizeName, itemName, quantity, drawAllFlag)

	// Return the updated prize list partial for HTMX
	data := gin.H{
		"Prizes": h.service.Prizes,
	}
	if err := h.templates.ExecuteTemplate(c.Writer, "prize_list_container.html", data); err != nil {
		logger.Infof("Error executing template: %v", err)
		c.String(http.StatusInternalServerError, "Template error")
	}
}

// UploadPrizesCSV handles the CSV upload for prizes.
func (h *HTTPHandler) UploadPrizesCSV(c *gin.Context) {
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
			h.service.ClearPrize()
			c.String(http.StatusInternalServerError, "Error reading CSV: %v", err)
			return
		}

		// logger.Infof("record: %+v", record)

		if len(record) != 4 {
			logger.Infof("Skipping malformed CSV record: %v", record)
			continue // Skip malformed rows
		}

		prizeName := record[0]
		itemName := record[1]
		quantity, err := strconv.Atoi(record[2])
		if err != nil {
			logger.Infof("Skipping CSV record with invalid quantity: %v", record)
			continue // Skip rows with invalid quantity
		}
		drawFromAll, err := strconv.ParseBool(record[3])
		if err != nil {
			logger.Infof("Skipping CSV record with invalid drawFromAll: %v", record)
			continue // Skip rows with invalid drawFromAll
		}

		h.service.AddPrize(prizeName, itemName, quantity, drawFromAll)
	}

	// Return the updated prize list partial for HTMX
	data := gin.H{
		"Prizes": h.service.Prizes,
	}
	if err := h.templates.ExecuteTemplate(c.Writer, "prize_list_container.html", data); err != nil {
		logger.Infof("Error executing template: %v", err)
		c.String(http.StatusInternalServerError, "Template error")
	}
}

// ShowParticipantsPage handles the request for the participant setting page.
func (h *HTTPHandler) ShowParticipantsPage(c *gin.Context) {
	data := gin.H{
		"title":        "參與者設定",
		"Participants": h.service.Participants,
	}
	h.renderPage(c, data, "participant_setting.html")
}

// AddParticipant handles the form submission for adding a new participant.
func (h *HTTPHandler) AddParticipant(c *gin.Context) {
	participantID := c.PostForm("participantID")
	participantName := c.PostForm("participantName")

	if participantID == "" || participantName == "" {
		c.String(http.StatusBadRequest, "Participant ID and Name cannot be empty")
		return
	}

	h.service.AddParticipant(participantID, participantName)

	// Return the updated participant list partial for HTMX
	data := gin.H{
		"Participants": h.service.Participants,
	}
	if err := h.templates.ExecuteTemplate(c.Writer, "participant_list_container.html", data); err != nil {
		logger.Infof("Error executing template: %v", err)
		c.String(http.StatusInternalServerError, "Template error")
	}
}

// UploadParticipantsCSV handles the CSV upload for participants.
func (h *HTTPHandler) UploadParticipantsCSV(c *gin.Context) {
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
			h.service.ClearParticipant()
			c.String(http.StatusInternalServerError, "Error reading CSV: %v", err)
			return
		}

		if len(record) != 2 {
			logger.Infof("Skipping malformed participant CSV record: %v", record)
			continue // Skip malformed rows
		}

		h.service.AddParticipant(record[0], record[1])
	}

	// Return the updated participant list partial for HTMX
	data := gin.H{
		"Participants": h.service.Participants,
	}
	if err := h.templates.ExecuteTemplate(c.Writer, "participant_list_container.html", data); err != nil {
		logger.Infof("Error executing template: %v", err)
		c.String(http.StatusInternalServerError, "Template error")
	}
}

// ShowLotteryPage handles the request for the main lottery drawing page.
func (h *HTTPHandler) ShowLotteryPage(c *gin.Context) {
	data := gin.H{
		"title":          "抽獎介面",
		"Prizes":         h.service.Prizes,
		"Participants":   h.service.Participants,
		"LotteryResults": h.service.LotteryResults,
	}
	h.renderPage(c, data, "lottery_interface.html")
}

// PerformDraw handles the request to draw a winner for a prize.
func (h *HTTPHandler) PerformDraw(c *gin.Context) {
	prizeName := c.PostForm("prizeName")
	if prizeName == "" {
		c.String(http.StatusBadRequest, "Please select a prize.")
		return
	}

	result, err := h.service.Draw(prizeName)
	if err != nil {
		c.String(http.StatusOK, "<p>%s</p>", err.Error()) // Return error as simple paragraph
		return
	}

	// Pass both the result and the updated prizes list to the template
	data := gin.H{
		"Result": result,
		"Prizes": h.service.Prizes,
	}

	if err := h.templates.ExecuteTemplate(c.Writer, "lottery_draw_response.html", data); err != nil {
		logger.Infof("Error executing template: %v", err)
		c.String(http.StatusInternalServerError, "Template error")
	}
}

// GetPrizeListPartial returns the HTML partial for the prize list body.
func (h *HTTPHandler) GetPrizeListPartial(c *gin.Context) {
	if err := h.templates.ExecuteTemplate(c.Writer, "prize_list_table_body.html", h.service.Prizes); err != nil {
		logger.Infof("Error executing template: %v", err)
		c.String(http.StatusInternalServerError, "Template error")
	}
}

// ExportResultsCSV handles the request to download the lottery results as a CSV file.
func (h *HTTPHandler) ExportResultsCSV(c *gin.Context) {
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment;filename=lottery_results.csv")

	// Add BOM to ensure UTF-8 compatibility in Excel
	c.Writer.Write([]byte("\xef\xbb\xbf"))

	w := csv.NewWriter(c.Writer)

	// Write header
	if err := w.Write([]string{"獎項名稱", "員工編號", "員工姓名", "獎品名稱"}); err != nil {
		logger.Infof("Error writing CSV header: %v", err)
		c.String(http.StatusInternalServerError, "Error writing CSV")
		return
	}

	// Write data
	for _, result := range h.service.LotteryResults {
		row := []string{result.PrizeName, result.WinnerID, result.WinnerName, result.PrizeItem}
		if err := w.Write(row); err != nil {
			logger.Infof("Error writing CSV row: %v", err)
			c.String(http.StatusInternalServerError, "Error writing CSV")
			return
		}
	}

	w.Flush()

	if err := w.Error(); err != nil {
		logger.Infof("Error flushing CSV writer: %v", err)
		c.String(http.StatusInternalServerError, "Error writing CSV")
	}
}
