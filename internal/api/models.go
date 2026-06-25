package api

import (
	"time"

	"scam-directory/internal/models"
)

type CreateScamRequest struct {
	Title           string                       `json:"title" binding:"required"`
	Description     string                       `json:"description" binding:"required"`
	Type            string                       `json:"type" binding:"required"`
	EstimatedLosses float64                      `json:"estimatedLosses"`
	Locations       []models.Location            `json:"locations"`
	ContactMethods  []models.ContactMethod       `json:"contactMethods"`
	TransferMethods []models.MoneyTransferMethod `json:"transferMethods"`
}

type ScamReportRequest struct {
	ReporterEmail string     `json:"reporterEmail" binding:"required,email"`
	Description   string     `json:"description" binding:"required"`
	LossAmount    float64    `json:"lossAmount"`
	DateOccurred  *time.Time `json:"dateOccurred"`
	City          *string    `json:"city"`
	Province      *string    `json:"province"`
	Country       *string    `json:"country"`
}

type UpdateScamStatusRequest struct {
	Status models.ScamStatus `json:"status" binding:"required"`
}

type SearchResponse struct {
	Results []models.Scam `json:"results"`
	Total   int           `json:"total"`
	Page    int           `json:"page"`
	Limit   int           `json:"limit"`
}

type ErrorResponse struct {
	Error       string `json:"error"`
	Code        string `json:"code,omitempty"`
	Description string `json:"description,omitempty"`
}
