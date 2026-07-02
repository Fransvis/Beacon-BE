package api

import (
	"context"
	"log"
	"net/http"
	"scam-directory/internal/models"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"
)

// experiencedRateLimiter tracks IP+scamID votes with a 24h window.
var experiencedRateLimiter = struct {
	mu    sync.Mutex
	votes map[string]time.Time
}{
	votes: make(map[string]time.Time),
}

func canVote(ip, scamID string) bool {
	key := ip + "|" + scamID
	ttl := 24 * time.Hour
	experiencedRateLimiter.mu.Lock()
	defer experiencedRateLimiter.mu.Unlock()
	if t, ok := experiencedRateLimiter.votes[key]; ok && time.Since(t) < ttl {
		return false
	}
	experiencedRateLimiter.votes[key] = time.Now()
	return true
}

type Handler struct {
	scamRepo    ScamRepository
	commentRepo CommentRepository
}

type ScamRepository interface {
	CreateScam(ctx context.Context, scam *models.Scam) error
	GetScamByID(ctx context.Context, id uuid.UUID) (*models.Scam, error)
	SearchScams(ctx context.Context, query string, offset, limit int) ([]models.Scam, int, error)
	FindSimilarScams(ctx context.Context, scamID uuid.UUID, limit int) ([]models.Scam, error)
	AddScamReport(ctx context.Context, report *models.ScamReport) error
	IncrementReportCount(ctx context.Context, id uuid.UUID) error
	LookupByIdentifier(ctx context.Context, identifier string) ([]models.Scam, error)
	GetScamStatistics(ctx context.Context) (map[string]interface{}, error)
	GetDailySummary(ctx context.Context) (map[string]interface{}, error)
	GetScamTypes(ctx context.Context) ([]models.ScamType, error)
	AddContactMethod(ctx context.Context, scamID uuid.UUID, cm *models.ContactMethod) error
	AddTransferMethod(ctx context.Context, scamID uuid.UUID, tm *models.MoneyTransferMethod) error
	AddLocation(ctx context.Context, scamID uuid.UUID, loc *models.Location) error
	AddKeyword(ctx context.Context, scamID uuid.UUID, keyword string) error
	RecordExperience(ctx context.Context, scamID uuid.UUID, userID *string, ipHash string) error
	GetMyActivity(ctx context.Context, userID string) (reported []models.Scam, experienced []models.Scam, err error)
}

type CommentRepository interface {
	Create(ctx context.Context, comment *models.Comment) error
	GetByScamID(ctx context.Context, scamID uuid.UUID) ([]models.Comment, error)
	CountByScamID(ctx context.Context, scamID uuid.UUID) (int, error)
}

func NewHandler(scamRepo ScamRepository, commentRepo CommentRepository) *Handler {
	return &Handler{
		scamRepo:    scamRepo,
		commentRepo: commentRepo,
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

	var reporterID *string
	if uid, ok := c.Get("user_id"); ok {
		s := uid.(string)
		reporterID = &s
	}

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
		ReporterID:         reporterID,
		MoneyDirection:     req.MoneyDirection,
		ScammerNames:       req.ScammerNames,
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
	if scam.ScammerNames == nil {
		scam.ScammerNames = []string{}
	}

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
// @Param offset query int false "Offset" default(0)
// @Param limit query int false "Limit results" default(20)
// @Success 200 {object} PaginatedScams
// @Router /scams [get]
func (h *Handler) SearchScams(c *gin.Context) {
	query := c.Query("query")

	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if offset < 0 {
		offset = 0
	}

	limit, err := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if err != nil || limit <= 0 || limit > 100 {
		limit = 20
	}

	log.Printf("Searching scams with query: %s, offset: %d, limit: %d", query, offset, limit)

	scams, total, err := h.scamRepo.SearchScams(c, query, offset, limit)
	if err != nil {
		log.Printf("Error searching scams: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search scams"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"scams":   scams,
		"total":   total,
		"hasMore": offset+len(scams) < total,
	})
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

	candidates, _, err := h.scamRepo.SearchScams(c, searchQuery, 0, 5)
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

	report := &models.ScamReport{
		ScamID:        id,
		ReporterEmail: req.ReporterEmail,
		Description:   req.Description,
		LossAmount:    req.LossAmount,
		DateOccurred:  dateOccurred,
		City:          req.City,
		Province:      req.Province,
		Country:       req.Country,
		Status:        "PENDING",
	}

	if err := h.scamRepo.AddScamReport(c, report); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add report"})
		return
	}

	c.JSON(http.StatusCreated, report)
}

// LookupScam finds scams associated with a given contact identifier (phone, email, URL).
func (h *Handler) LookupScam(c *gin.Context) {
	identifier := c.Param("identifier")
	if identifier == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "identifier is required"})
		return
	}

	scams, err := h.scamRepo.LookupByIdentifier(c, identifier)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to lookup identifier"})
		return
	}

	c.JSON(http.StatusOK, scams)
}

// ExperiencedScam increments the report count without requiring a full report.
// Rate-limited to one vote per IP per scam per 24 hours.
func (h *Handler) ExperiencedScam(c *gin.Context) {
	id, err := uuid.FromString(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid scam ID"})
		return
	}

	ip := c.ClientIP()
	if !canVote(ip, id.String()) {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "Already counted"})
		return
	}

	if err := h.scamRepo.IncrementReportCount(c, id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update count"})
		return
	}

	var userID *string
	if uid, ok := c.Get("user_id"); ok {
		s := uid.(string)
		userID = &s
	}
	_ = h.scamRepo.RecordExperience(c, id, userID, ip)

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// GetMyActivity returns scams the authenticated user reported or experienced.
func (h *Handler) GetMyActivity(c *gin.Context) {
	userID, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}

	reported, experienced, err := h.scamRepo.GetMyActivity(c, userID.(string))
	if err != nil {
		log.Printf("GetMyActivity error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch activity"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"reported":    reported,
		"experienced": experienced,
	})
}

// GetScamTypes returns all rows from the scam_types lookup table.
func (h *Handler) GetScamTypes(c *gin.Context) {
	types, err := h.scamRepo.GetScamTypes(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get scam types"})
		return
	}
	c.JSON(http.StatusOK, types)
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

func (h *Handler) GetDailySummary(c *gin.Context) {
	summary, err := h.scamRepo.GetDailySummary(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get daily summary"})
		return
	}

	c.JSON(http.StatusOK, summary)
}

// GetComments returns all comments for a scam.
func (h *Handler) GetComments(c *gin.Context) {
	id, err := uuid.FromString(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid scam ID"})
		return
	}

	comments, err := h.commentRepo.GetByScamID(c, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get comments"})
		return
	}

	c.JSON(http.StatusOK, comments)
}

// CreateComment adds a comment to a scam. Authentication is optional.
func (h *Handler) CreateComment(c *gin.Context) {
	scamID, err := uuid.FromString(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid scam ID"})
		return
	}

	var req struct {
		Content     string `json:"content"      binding:"required,max=2000"`
		AuthorName  string `json:"authorName"   binding:"max=100"`
		IsAnonymous bool   `json:"isAnonymous"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	comment := &models.Comment{
		ScamID:      scamID.String(),
		Content:     req.Content,
		IsAnonymous: req.IsAnonymous,
	}

	if userID, ok := c.Get("user_id"); ok {
		uid := userID.(string)
		comment.UserID = &uid
		comment.IsAnonymous = false
	}

	if comment.IsAnonymous {
		name := "Anonymous"
		if req.AuthorName != "" {
			name = req.AuthorName
		}
		comment.AuthorName = &name
	}

	if err := h.commentRepo.Create(c, comment); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create comment"})
		return
	}

	c.JSON(http.StatusCreated, comment)
}

type AddContactMethodRequest struct {
	Type  string `json:"type"  binding:"required,max=50"`
	Value string `json:"value" binding:"required,max=255"`
}

type AddTransferMethodRequest struct {
	Type        string `json:"type"        binding:"required,max=50"`
	Description string `json:"description" binding:"max=500"`
}

type AddLocationRequest struct {
	City     string `json:"city"     binding:"max=100"`
	Province string `json:"province" binding:"max=100"`
	Country  string `json:"country"  binding:"required,max=100"`
}

type AddKeywordRequest struct {
	Keyword string `json:"keyword" binding:"required,max=100"`
}

func (h *Handler) AddContactMethod(c *gin.Context) {
	scamID, err := uuid.FromString(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid scam ID"})
		return
	}

	var req AddContactMethodRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cm := &models.ContactMethod{Type: req.Type, Value: req.Value, IsValid: true}
	if err := h.scamRepo.AddContactMethod(c, scamID, cm); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add contact method"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"ok": true})
}

func (h *Handler) AddTransferMethod(c *gin.Context) {
	scamID, err := uuid.FromString(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid scam ID"})
		return
	}

	var req AddTransferMethodRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tm := &models.MoneyTransferMethod{Type: req.Type, Description: req.Description}
	if err := h.scamRepo.AddTransferMethod(c, scamID, tm); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add transfer method"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"ok": true})
}

func (h *Handler) AddLocation(c *gin.Context) {
	scamID, err := uuid.FromString(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid scam ID"})
		return
	}

	var req AddLocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	loc := &models.Location{City: req.City, Province: req.Province, Country: req.Country, ReportCount: 1}
	if err := h.scamRepo.AddLocation(c, scamID, loc); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add location"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"ok": true})
}

func (h *Handler) AddKeyword(c *gin.Context) {
	scamID, err := uuid.FromString(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid scam ID"})
		return
	}

	var req AddKeywordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.scamRepo.AddKeyword(c, scamID, req.Keyword); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add keyword"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"ok": true})
}
