package api

import (
	"context"
	"log"
	"net/http"
	"scam-directory/internal/models"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
)

type Handler struct {
	scamRepo ScamRepository
}

type ScamRepository interface {
	CreateScam(ctx context.Context, scam *models.Scam) error
	GetScamByID(ctx context.Context, id uuid.UUID) (*models.Scam, error)
	SearchScams(ctx context.Context, query string, limit int) ([]models.Scam, error)
	FindSimilarScams(ctx context.Context, scamID uuid.UUID, limit int) ([]models.Scam, error)
	AddScamReport(ctx context.Context, report *models.ScamReport) error
	GetScamStatistics(ctx context.Context) (map[string]interface{}, error)
}

func NewHandler(scamRepo ScamRepository) *Handler {
	return &Handler{
		scamRepo: scamRepo,
	}
}

// CreateScam godoc
// @Summary Create a new scam report
// @Description Create a new scam report with details
// @Tags scams
// @Accept json
// @Produce json
// @Param scam body CreateScamRequest true "Scam details"
// @Success 201 {object} Scam
// @Router /scams [post]
func (h *Handler) CreateScam(c *gin.Context) {
	var req CreateScamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("Invalid request body: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Generate new UUID and log it
	id, err := uuid.NewV4()
	if err != nil {
		log.Printf("Failed to generate ID: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate ID"})
		return
	}

	now := time.Now()

	// Create scam object
	verificationStatus := "UNVERIFIED"
	statusActive := models.StatusActive
	riskMedium := models.RiskMedium

	scam := &models.Scam{
		ID:                 id,
		Title:              &req.Title,
		Description:        &req.Description,
		Type:               &req.Type,
		ReportCount:        1,
		DateFirstReported:  &now,
		DateLastReported:   &now,
		Status:             &statusActive,
		EstimatedLosses:    req.EstimatedLosses,
		RiskLevel:          &riskMedium,
		CreatedAt:          &now,
		UpdatedAt:          &now,
		VerificationStatus: &verificationStatus,
	}

	// Log the scam creation attempt
	log.Printf("Creating scam: %+v", scam)

	// Initialize empty slices for related data
	if len(req.Locations) > 0 {
		scam.Locations = req.Locations
		if len(req.Locations) == 1 {
			tempLocation := req.Locations[0].City + ", " + req.Locations[0].Country
			scam.PrimaryLocation = &tempLocation
		}
	} else {
		scam.Locations = []models.Location{}
	}

	if len(req.ContactMethods) > 0 {
		scam.ContactMethods = req.ContactMethods
	} else {
		scam.ContactMethods = []models.ContactMethod{}
	}

	if len(req.TransferMethods) > 0 {
		scam.TransferMethods = req.TransferMethods
	} else {
		scam.TransferMethods = []models.MoneyTransferMethod{}
	}

	// Initialize other slices
	scam.Demographics = []models.VictimDemographic{}
	scam.Evidence = []models.Evidence{}
	scam.Keywords = []string{}
	scam.RelatedScamIDs = []uuid.UUID{}

	if err := h.scamRepo.CreateScam(c, scam); err != nil {
		log.Printf("Failed to create scam: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create scam: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, scam)
}

// GetScam godoc
// @Summary Get a scam by ID
// @Description Get detailed information about a specific scam
// @Tags scams
// @Accept json
// @Produce json
// @Param id path string true "Scam ID"
// @Success 200 {object} Scam
// @Router /scams/{id} [get]
func (h *Handler) GetScam(c *gin.Context) {
	id, err := uuid.FromString(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid scam ID"})
		return
	}

	log.Printf("Fetching scam with ID: %s", id)

	scam, err := h.GetScamByID(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Scam not found"})
		return
	}

	c.JSON(http.StatusOK, scam)
}

func (h *Handler) GetScamByID(c *gin.Context, id uuid.UUID) (*models.Scam, error) {
	log.Printf("Fetching scam with ID: %s", id)

	scam, err := h.scamRepo.GetScamByID(c, id)
	if err != nil {
		log.Printf("Error fetching scam: %v", err)
		return nil, err
	}

	return scam, nil
}

// SearchScams godoc
// @Summary Search for scams
// @Description Search for scams using various criteria
// @Tags scams
// @Accept json
// @Produce json
// @Param query query string false "Search query"
// @Param limit query int false "Limit results" default(50)
// @Success 200 {array} Scam
// @Router /scams/search [get]
func (h *Handler) SearchScams(c *gin.Context) {
	query := c.Query("query")
	limitStr := c.DefaultQuery("limit", "50")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 200 {
		limit = 50
	}
	log.Printf("Searching scams with query: %s, limit: %d", query, limit)

	scams, err := h.scamRepo.SearchScams(c, query, limit)
	if err != nil {
		log.Printf("Error searching scams: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search scams"})
		return
	}

	c.JSON(http.StatusOK, scams)
}

// CheckDuplicates checks whether a scam being submitted closely matches existing entries.
func (h *Handler) CheckDuplicates(c *gin.Context) {
	var req struct {
		Title          string `json:"title"`
		Description    string `json:"description"`
		ContactMethods []struct {
			Type  string `json:"type"`
			Value string `json:"value"`
		} `json:"contactMethods"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Build a search query from the submitted data — combine title + first contact value
	searchQuery := req.Title
	if len(req.ContactMethods) > 0 && req.ContactMethods[0].Value != "" {
		searchQuery = req.ContactMethods[0].Value
	}

	candidates, err := h.scamRepo.SearchScams(c, searchQuery, 5)
	if err != nil {
		log.Printf("Error checking duplicates: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check duplicates"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"hasDuplicates": len(candidates) > 0,
		"candidates":    candidates,
	})
}

// FindSimilarScams godoc
// @Summary Find similar scams
// @Description Find scams similar to a given scam ID
// @Tags scams
// @Accept json
// @Produce json
// @Param id path string true "Scam ID"
// @Success 200 {array} Scam
// @Router /scams/{id}/similar [get]
func (h *Handler) FindSimilarScams(c *gin.Context) {
	id, err := uuid.FromString(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid scam ID"})
		return
	}

	scams, err := h.scamRepo.FindSimilarScams(c, id, 10)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find similar scams"})
		return
	}

	c.JSON(http.StatusOK, scams)
}

// ReportScam godoc
// @Summary Report a new instance of a scam
// @Description Add a new report to an existing scam
// @Tags scams
// @Accept json
// @Produce json
// @Param id path string true "Scam ID"
// @Param report body ScamReportRequest true "Report details"
// @Success 201 {object} ScamReport
// @Router /scams/{id}/report [post]
func (h *Handler) ReportScam(c *gin.Context) {
	id, err := uuid.FromString(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid scam ID"})
		return
	}

	var req ScamReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	dateOccurred := time.Now()
	if req.DateOccurred != nil {
		dateOccurred = *req.DateOccurred
	}

	location := models.Location{}
	if req.Location != nil {
		location = *req.Location
	}

	report := &models.ScamReport{
		ScamID:        id,
		ReporterEmail: req.ReporterEmail,
		Description:   req.Description,
		LossAmount:    req.LossAmount,
		DateOccurred:  dateOccurred,
		Location:      location,
		Status:        "PENDING",
	}

	if err := h.scamRepo.AddScamReport(c, report); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add report"})
		return
	}

	c.JSON(http.StatusCreated, report)
}

// GetStatistics godoc
// @Summary Get scam statistics
// @Description Get overall statistics about scams
// @Tags statistics
// @Accept json
// @Produce json
// @Success 200 {object} Statistics
// @Router /statistics [get]
func (h *Handler) GetStatistics(c *gin.Context) {
	stats, err := h.scamRepo.GetScamStatistics(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get statistics"})
		return
	}

	c.JSON(http.StatusOK, stats)
}
