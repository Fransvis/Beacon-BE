package models

import (
	"time"

	"github.com/gofrs/uuid"
)

type RiskLevel string

const (
	RiskLow    RiskLevel = "LOW"
	RiskMedium RiskLevel = "MEDIUM"
	RiskHigh   RiskLevel = "HIGH"
)

type ScamStatus string

const (
	StatusActive             ScamStatus = "ACTIVE"
	StatusResolved           ScamStatus = "RESOLVED"
	StatusUnderInvestigation ScamStatus = "UNDER_INVESTIGATION"
)

type ContactMethod struct {
	Type    string `json:"type"`    // e.g., "phone", "email", "social_media", "website"
	Value   string `json:"value"`   // the actual contact info
	IsValid bool   `json:"isValid"` // whether this contact is still active/valid
}

type MoneyTransferMethod struct {
	Type        string `json:"type"`        // e.g., "bank_transfer", "crypto", "gift_cards"
	Description string `json:"description"` // additional details
}

type VictimDemographic struct {
	AgeRange   string `json:"ageRange"`   // e.g., "18-25", "26-35", etc.
	Location   string `json:"location"`   // general location of victims
	Occupation string `json:"occupation"` // if targeting specific professions
	Count      int    `json:"count"`      // number of victims in this demographic
}

type Evidence struct {
	ID          uuid.UUID `json:"id"`
	Type        string    `json:"type"` // e.g., "screenshot", "audio", "document"
	URL         string    `json:"url"`  // URL to the stored evidence
	UploadedAt  time.Time `json:"uploadedAt"`
	Description string    `json:"description"`
}

type Location struct {
	City        string `json:"city"`
	Province    string `json:"province"`
	Country     string `json:"country"`
	ReportCount int    `json:"reportCount"`
	Coordinates *struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	} `json:"coordinates,omitempty"`
}

type Scam struct {
	ID                uuid.UUID   `json:"id" db:"id"`
	Title             *string     `json:"title,omitempty" db:"title"`
	Description       *string     `json:"description,omitempty" db:"description"`
	Type              *string     `json:"type,omitempty" db:"type"`
	ReportCount       int         `json:"reportCount" db:"report_count"`
	DateFirstReported *time.Time  `json:"dateFirstReported,omitempty" db:"date_first_reported"`
	DateLastReported  *time.Time  `json:"dateLastReported,omitempty" db:"date_last_reported"`
	Status            *ScamStatus `json:"status,omitempty" db:"status"`
	EstimatedLosses   float64     `json:"estimatedLosses" db:"estimated_losses"`
	Locations         []Location  `json:"locations" db:"-"`
	PrimaryLocation   *string     `json:"primaryLocation,omitempty" db:"primary_location"`

	// New fields
	RiskLevel          *RiskLevel            `json:"riskLevel,omitempty" db:"risk_level"`
	ContactMethods     []ContactMethod       `json:"contactMethods" db:"-"`
	TransferMethods    []MoneyTransferMethod `json:"transferMethods" db:"-"`
	Demographics       []VictimDemographic   `json:"demographics" db:"-"`
	RelatedScamIDs     []uuid.UUID           `json:"relatedScamIds" db:"-"`
	Evidence           []Evidence            `json:"evidence" db:"-"`
	VerificationStatus *string               `json:"verificationStatus,omitempty" db:"verification_status"`

	// AI-related fields
	SimilarityVector []float32 `json:"similarityVector,omitempty" db:"-"`
	Keywords         []string  `json:"keywords" db:"-"`
	ScamPattern      *string   `json:"scamPattern,omitempty" db:"scam_pattern"`

	// Metadata
	CreatedAt      *time.Time `json:"createdAt,omitempty" db:"created_at"`
	UpdatedAt      *time.Time `json:"updatedAt,omitempty" db:"updated_at"`
	LastAnalyzedAt *time.Time `json:"lastAnalyzedAt,omitempty" db:"last_analyzed_at"`
	Rank           *float64   `json:"rank,omitempty"`
}

type ScamReport struct {
	ID             uuid.UUID       `json:"id"`
	ScamID         uuid.UUID       `json:"scamId"`
	ReporterEmail  string          `json:"reporterEmail"`
	Description    string          `json:"description"`
	LossAmount     float64         `json:"lossAmount"`
	DateOccurred   time.Time       `json:"dateOccurred"`
	Location       Location        `json:"location"`
	ContactMethods []ContactMethod `json:"contactMethods"`
	Evidence       []Evidence      `json:"evidence"`
	Status         string          `json:"status"` // "PENDING", "VERIFIED", "REJECTED"
	CreatedAt      time.Time       `json:"createdAt"`
}
