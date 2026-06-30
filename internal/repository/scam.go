package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"scam-directory/internal/models"
	"time"

	"encoding/json"

	"github.com/gofrs/uuid"
	"github.com/jmoiron/sqlx"
)

type ScamRepository struct {
	db *sqlx.DB
}

func NewScamRepository(db *sqlx.DB) *ScamRepository {
	return &ScamRepository{db: db}
}

// CreateScam inserts a new scam record
func (r *ScamRepository) CreateScam(ctx context.Context, scam *models.Scam) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	// Insert main scam record
	query := `
		INSERT INTO scams (
			id, title, description, type, report_count,
			date_first_reported, date_last_reported, status,
			estimated_losses, primary_location, risk_level,
			verification_status, scam_pattern, reporter_id, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
		) RETURNING id`

	err = tx.QueryRowContext(
		ctx,
		query,
		scam.ID,
		scam.Title,
		scam.Description,
		scam.Type,
		scam.ReportCount,
		scam.DateFirstReported,
		scam.DateLastReported,
		scam.Status,
		scam.EstimatedLosses,
		scam.PrimaryLocation,
		scam.RiskLevel,
		scam.VerificationStatus,
		scam.ScamPattern,
		scam.ReporterID,
		scam.CreatedAt,
		scam.UpdatedAt,
	).Scan(&scam.ID)

	if err != nil {
		return fmt.Errorf("failed to insert scam: %v", err)
	}

	// Insert locations if any
	if len(scam.Locations) > 0 {
		locationQuery := `
			INSERT INTO locations (scam_id, city, province, country, report_count)
			VALUES ($1, $2, $3, $4, $5)`

		for _, loc := range scam.Locations {
			_, err = tx.ExecContext(ctx, locationQuery,
				scam.ID, loc.City, loc.Province, loc.Country, loc.ReportCount)
			if err != nil {
				return fmt.Errorf("failed to insert location: %v", err)
			}
		}
	}

	// Insert contact methods if any
	if len(scam.ContactMethods) > 0 {
		contactQuery := `
			INSERT INTO contact_methods (scam_id, type, value, is_valid)
			VALUES ($1, $2, $3, $4)`

		for _, cm := range scam.ContactMethods {
			_, err = tx.ExecContext(ctx, contactQuery,
				scam.ID, cm.Type, cm.Value, cm.IsValid)
			if err != nil {
				return fmt.Errorf("failed to insert contact method: %v", err)
			}
		}
	}

	// Insert transfer methods if any
	if len(scam.TransferMethods) > 0 {
		transferQuery := `
			INSERT INTO transfer_methods (scam_id, type, description)
			VALUES ($1, $2, $3)`

		for _, tm := range scam.TransferMethods {
			_, err = tx.ExecContext(ctx, transferQuery,
				scam.ID, tm.Type, tm.Description)
			if err != nil {
				return fmt.Errorf("failed to insert transfer method: %v", err)
			}
		}
	}

	// Insert scammer names if any
	if len(scam.ScammerNames) > 0 {
		nameQuery := `
			INSERT INTO scammer_names (scam_id, name)
			VALUES ($1, $2)`

		for _, name := range scam.ScammerNames {
			if name == "" {
				continue
			}
			_, err = tx.ExecContext(ctx, nameQuery, scam.ID, name)
			if err != nil {
				return fmt.Errorf("failed to insert scammer name: %v", err)
			}
		}
	}

	return tx.Commit()
}

// AddContactMethod adds a new contact method to an existing scam.
func (r *ScamRepository) AddContactMethod(ctx context.Context, scamID uuid.UUID, cm *models.ContactMethod) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO contact_methods (scam_id, type, value, is_valid) VALUES ($1, $2, $3, $4)
		 ON CONFLICT (scam_id, type, value) DO NOTHING`,
		scamID, cm.Type, cm.Value, cm.IsValid)
	if err != nil {
		return fmt.Errorf("failed to add contact method: %v", err)
	}
	return r.touchScam(ctx, scamID)
}

// AddTransferMethod adds a new transfer method to an existing scam.
func (r *ScamRepository) AddTransferMethod(ctx context.Context, scamID uuid.UUID, tm *models.MoneyTransferMethod) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO transfer_methods (scam_id, type, description) VALUES ($1, $2, $3)`,
		scamID, tm.Type, tm.Description)
	if err != nil {
		return fmt.Errorf("failed to add transfer method: %v", err)
	}
	return r.touchScam(ctx, scamID)
}

// AddLocation adds a new location to an existing scam.
func (r *ScamRepository) AddLocation(ctx context.Context, scamID uuid.UUID, loc *models.Location) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO locations (scam_id, city, province, country, report_count)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (scam_id, city, province, country) DO NOTHING`,
		scamID, loc.City, loc.Province, loc.Country, loc.ReportCount)
	if err != nil {
		return fmt.Errorf("failed to add location: %v", err)
	}
	return r.touchScam(ctx, scamID)
}

// AddKeyword adds a new keyword to an existing scam.
func (r *ScamRepository) AddKeyword(ctx context.Context, scamID uuid.UUID, keyword string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO keywords (scam_id, keyword) VALUES ($1, $2)
		 ON CONFLICT (scam_id, keyword) DO NOTHING`,
		scamID, keyword)
	if err != nil {
		return fmt.Errorf("failed to add keyword: %v", err)
	}
	return r.touchScam(ctx, scamID)
}

// touchScam updates the updated_at timestamp of a scam.
func (r *ScamRepository) touchScam(ctx context.Context, scamID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE scams SET updated_at = NOW() WHERE id = $1`,
		scamID)
	return err
}

// GetScamByID retrieves a scam with all related data
func (r *ScamRepository) GetScamByID(ctx context.Context, id uuid.UUID) (*models.Scam, error) {
	log.Printf("Fetching scam with ID: %s", id)

	// Temporary struct for scanning
	type dbScam struct {
		models.Scam
		Locations       json.RawMessage `db:"locations"`
		ContactMethods  json.RawMessage `db:"contact_methods"`
		TransferMethods json.RawMessage `db:"transfer_methods"`
		Demographics    json.RawMessage `db:"demographics"`
		Evidence        json.RawMessage `db:"evidence"`
		Keywords        json.RawMessage `db:"keywords"`
		RelatedScamIDs  json.RawMessage `db:"related_scam_ids"`
		ScammerNames    json.RawMessage `db:"scammer_names"`
	}

	var dbResult dbScam
	query := `
        SELECT 
            s.id, 
            s.title,
            s.description,
            s.type,
            s.report_count,
            s.date_first_reported,
            s.date_last_reported,
            s.status,
            s.estimated_losses,
            s.primary_location,
            s.risk_level,
            s.verification_status,
            s.scam_pattern,
            s.created_at,
            s.updated_at,
            s.last_analyzed_at,
            (
                SELECT COALESCE(json_agg(row_to_json(l)), '[]'::json)
                FROM locations l
                WHERE l.scam_id = s.id
            ) as locations,
            (
                SELECT COALESCE(json_agg(row_to_json(cm)), '[]'::json)
                FROM contact_methods cm
                WHERE cm.scam_id = s.id
            ) as contact_methods,
            (
                SELECT COALESCE(json_agg(row_to_json(tm)), '[]'::json)
                FROM transfer_methods tm
                WHERE tm.scam_id = s.id
            ) as transfer_methods,
            (
                SELECT COALESCE(json_agg(row_to_json(d)), '[]'::json)
                FROM demographics d
                WHERE d.scam_id = s.id
            ) as demographics,
            (
                SELECT COALESCE(json_agg(row_to_json(e)), '[]'::json)
                FROM evidence e
                WHERE e.scam_id = s.id
            ) as evidence,
            (
                SELECT COALESCE(json_agg(k.keyword), '[]'::json)
                FROM keywords k
                WHERE k.scam_id = s.id
            ) as keywords,
            (
                SELECT COALESCE(json_agg(rs.related_scam_id), '[]'::json)
                FROM related_scams rs
                WHERE rs.scam_id = s.id
            ) as related_scam_ids,
            (
                SELECT COALESCE(json_agg(sn.name), '[]'::json)
                FROM scammer_names sn
                WHERE sn.scam_id = s.id
            ) as scammer_names
        FROM scams s
        WHERE s.id = $1;`

	log.Printf("Executing query: %s with ID: %s", query, id)

	err := r.db.GetContext(ctx, &dbResult, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("No scam found with ID: %s", id)
		} else {
			log.Printf("Error fetching scam: %v", err)
		}
		return nil, err
	}

	// Copy the base fields
	scam := dbResult.Scam

	// Unmarshal the JSON fields
	if err := json.Unmarshal(dbResult.Locations, &scam.Locations); err != nil {
		log.Printf("Error unmarshaling locations: %v", err)
		return nil, err
	}
	if err := json.Unmarshal(dbResult.ContactMethods, &scam.ContactMethods); err != nil {
		log.Printf("Error unmarshaling contact methods: %v", err)
		return nil, err
	}
	if err := json.Unmarshal(dbResult.TransferMethods, &scam.TransferMethods); err != nil {
		log.Printf("Error unmarshaling transfer methods: %v", err)
		return nil, err
	}
	if err := json.Unmarshal(dbResult.Demographics, &scam.Demographics); err != nil {
		log.Printf("Error unmarshaling demographics: %v", err)
		return nil, err
	}
	if err := json.Unmarshal(dbResult.Evidence, &scam.Evidence); err != nil {
		log.Printf("Error unmarshaling evidence: %v", err)
		return nil, err
	}
	if err := json.Unmarshal(dbResult.Keywords, &scam.Keywords); err != nil {
		log.Printf("Error unmarshaling keywords: %v", err)
		return nil, err
	}
	if err := json.Unmarshal(dbResult.RelatedScamIDs, &scam.RelatedScamIDs); err != nil {
		log.Printf("Error unmarshaling related scam IDs: %v", err)
		return nil, err
	}
	if err := json.Unmarshal(dbResult.ScammerNames, &scam.ScammerNames); err != nil {
		log.Printf("Error unmarshaling scammer names: %v", err)
		return nil, err
	}

	return &scam, nil
}

// GetScamWithRelated retrieves a scam with related data
func (r *ScamRepository) GetScamWithRelated(ctx context.Context, id uuid.UUID) (*models.Scam, error) {
	query := `
		SELECT 
			s.*,
			json_agg(DISTINCT l.*) as locations,
			json_agg(DISTINCT cm.*) as contact_methods,
			json_agg(DISTINCT tm.*) as transfer_methods
		FROM scams s
		LEFT JOIN locations l ON l.scam_id = s.id
		LEFT JOIN contact_methods cm ON cm.scam_id = s.id
		LEFT JOIN transfer_methods tm ON tm.scam_id = s.id
		WHERE s.id = $1
		GROUP BY s.id`

	var scam models.Scam
	err := r.db.GetContext(ctx, &scam, query, id)
	return &scam, err
}

// SearchScams performs a full-text search on scams and returns the matching page
// along with the total number of matches.
func (r *ScamRepository) SearchScams(ctx context.Context, query string, offset, limit int) ([]models.Scam, int, error) {
	log.Printf("Searching for scams with query: %s, offset: %d, limit: %d", query, offset, limit)

	var total int
	var searchQuery string
	var args []interface{}

	if query == "" {
		// Count all scams
		if err := r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM scams`); err != nil {
			return nil, 0, fmt.Errorf("failed to count scams: %v", err)
		}

		searchQuery = `
			SELECT DISTINCT
				s.id, s.title, s.description, s.type, s.report_count,
				s.date_first_reported, s.date_last_reported, s.status,
				s.estimated_losses, s.primary_location, s.risk_level,
				s.verification_status, s.scam_pattern,
				s.created_at, s.updated_at, s.last_analyzed_at,
				0 as rank
			FROM scams s
			ORDER BY s.created_at DESC
			LIMIT $1 OFFSET $2`
		args = []interface{}{limit, offset}
	} else {
		like := "%" + query + "%"

		// Count matching scams
		countQuery := `
			SELECT COUNT(DISTINCT s.id)
			FROM scams s
			LEFT JOIN contact_methods cm ON cm.scam_id = s.id
			WHERE
				s.search_vector @@ plainto_tsquery('english', $1)
				OR s.title ILIKE $2
				OR s.description ILIKE $2
				OR s.type ILIKE $2
				OR cm.value ILIKE $2`
		if err := r.db.GetContext(ctx, &total, countQuery, query, like); err != nil {
			return nil, 0, fmt.Errorf("failed to count search results: %v", err)
		}

		searchQuery = `
			SELECT DISTINCT
				s.id, s.title, s.description, s.type, s.report_count,
				s.date_first_reported, s.date_last_reported, s.status,
				s.estimated_losses, s.primary_location, s.risk_level,
				s.verification_status, s.scam_pattern,
				s.created_at, s.updated_at, s.last_analyzed_at,
				COALESCE(ts_rank(s.search_vector, plainto_tsquery('english', $1)), 0) as rank
			FROM scams s
			LEFT JOIN contact_methods cm ON cm.scam_id = s.id
			WHERE
				s.search_vector @@ plainto_tsquery('english', $1)
				OR s.title ILIKE $2
				OR s.description ILIKE $2
				OR s.type ILIKE $2
				OR cm.value ILIKE $2
			ORDER BY rank DESC
			LIMIT $3 OFFSET $4`
		args = []interface{}{query, like, limit, offset}
	}

	log.Printf("Executing query: %s with params: %v", searchQuery, args)

	scams := make([]models.Scam, 0)
	err := r.db.SelectContext(ctx, &scams, searchQuery, args...)
	if err != nil {
		log.Printf("Error executing search query: %v", err)
		return nil, 0, err
	}

	log.Printf("Found %d scams (total %d)", len(scams), total)
	return scams, total, nil
}

// FindSimilarScams finds scams with similar patterns or characteristics
func (r *ScamRepository) FindSimilarScams(ctx context.Context, scamID uuid.UUID, limit int) ([]models.Scam, error) {
	query := `
		WITH target_scam AS (
			SELECT type, keywords, scam_pattern
			FROM scams
			WHERE id = $1
		)
		SELECT s.*
		FROM scams s, target_scam t
		WHERE s.id != $1
		AND (
			s.type = t.type
			OR EXISTS (
				SELECT 1 FROM unnest(s.keywords) k1
				JOIN unnest(t.keywords) k2 ON k1 = k2
			)
			OR similarity(s.scam_pattern, t.scam_pattern) > 0.3
		)
		ORDER BY similarity(s.scam_pattern, t.scam_pattern) DESC
		LIMIT $2`

	var scams []models.Scam
	err := r.db.SelectContext(ctx, &scams, query, scamID, limit)
	return scams, err
}

// UpdateScamStatus updates the status of a scam
func (r *ScamRepository) UpdateScamStatus(ctx context.Context, id uuid.UUID, status models.ScamStatus) error {
	query := `
		UPDATE scams 
		SET status = $1, updated_at = NOW() 
		WHERE id = $2`

	_, err := r.db.ExecContext(ctx, query, status, id)
	return err
}

// AddScamReport creates a new scam report and updates scam statistics
func (r *ScamRepository) AddScamReport(ctx context.Context, report *models.ScamReport) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Insert report
	reportQuery := `
		INSERT INTO scam_reports (
			id, scam_id, reporter_email, description,
			loss_amount, date_occurred, city, province, country,
			status, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id`

	err = tx.QueryRowContext(
		ctx,
		reportQuery,
		report.ID,
		report.ScamID,
		report.ReporterEmail,
		report.Description,
		report.LossAmount,
		report.DateOccurred,
		report.City,
		report.Province,
		report.Country,
		report.Status,
		time.Now(),
	).Scan(&report.ID)

	if err != nil {
		return err
	}

	// Update scam statistics
	updateQuery := `
		UPDATE scams
		SET report_count = report_count + 1,
			date_last_reported = $1,
			estimated_losses = estimated_losses + $2,
			updated_at = NOW()
		WHERE id = $3`

	_, err = tx.ExecContext(
		ctx,
		updateQuery,
		report.DateOccurred,
		report.LossAmount,
		report.ScamID,
	)

	if err != nil {
		return err
	}

	return tx.Commit()
}

// LookupByIdentifier finds scams whose contact_methods match the given value.
func (r *ScamRepository) LookupByIdentifier(ctx context.Context, identifier string) ([]models.Scam, error) {
	query := `
		SELECT DISTINCT
			s.id, s.title, s.description, s.type, s.report_count,
			s.date_first_reported, s.date_last_reported, s.status,
			s.estimated_losses, s.primary_location, s.risk_level,
			s.verification_status, s.scam_pattern,
			s.created_at, s.updated_at, s.last_analyzed_at,
			0 as rank
		FROM scams s
		JOIN contact_methods cm ON cm.scam_id = s.id
		WHERE cm.value ILIKE $1
		ORDER BY s.report_count DESC`

	scams := make([]models.Scam, 0)
	err := r.db.SelectContext(ctx, &scams, query, "%"+identifier+"%")
	return scams, err
}

// IncrementReportCount bumps the report count and updates date_last_reported for a scam.
func (r *ScamRepository) IncrementReportCount(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE scams SET report_count = report_count + 1, date_last_reported = NOW(), updated_at = NOW() WHERE id = $1`,
		id,
	)
	return err
}

// GetScamTypes returns all rows from the scam_types lookup table.
func (r *ScamRepository) GetScamTypes(ctx context.Context) ([]models.ScamType, error) {
	var types []models.ScamType
	err := r.db.SelectContext(ctx, &types, `SELECT slug, label, description, icon, created_at FROM scam_types ORDER BY label`)
	return types, err
}

// GetScamStatistics retrieves statistics about scams
func (r *ScamRepository) GetScamStatistics(ctx context.Context) (map[string]interface{}, error) {
	query := `
		SELECT
			(SELECT COUNT(*) FROM scams)                                              AS total_scams,
			(SELECT COALESCE(SUM(report_count), 0) FROM scams)                       AS total_reports,
			(SELECT COALESCE(SUM(estimated_losses), 0) FROM scams)                   AS total_losses,
			(SELECT COALESCE(json_object_agg(type, cnt), '{}'::json)
			   FROM (SELECT type, COUNT(*) AS cnt FROM scams GROUP BY type) t)        AS scams_by_type,
			(SELECT COALESCE(json_object_agg(status, cnt), '{}'::json)
			   FROM (SELECT status, COUNT(*) AS cnt FROM scams GROUP BY status) s)   AS scams_by_status`

	row := r.db.QueryRowContext(ctx, query)

	var (
		totalScams   int64
		totalReports int64
		totalLosses  float64
		byType       []byte
		byStatus     []byte
	)

	if err := row.Scan(&totalScams, &totalReports, &totalLosses, &byType, &byStatus); err != nil {
		return nil, err
	}

	var scamsByType, scamsByStatus map[string]interface{}
	if err := json.Unmarshal(byType, &scamsByType); err != nil {
		scamsByType = map[string]interface{}{}
	}
	if err := json.Unmarshal(byStatus, &scamsByStatus); err != nil {
		scamsByStatus = map[string]interface{}{}
	}

	return map[string]interface{}{
		"total_scams":     totalScams,
		"total_reports":   totalReports,
		"total_losses":    totalLosses,
		"scams_by_type":   scamsByType,
		"scams_by_status": scamsByStatus,
	}, nil
}

// RecordExperience inserts a row into scam_experiences, ignoring duplicate user+scam pairs.
func (r *ScamRepository) RecordExperience(ctx context.Context, scamID uuid.UUID, userID *string, ipHash string) error {
	query := `
		INSERT INTO scam_experiences (scam_id, user_id, ip_hash)
		VALUES ($1, $2, $3)
		ON CONFLICT (scam_id, user_id) DO NOTHING`
	_, err := r.db.ExecContext(ctx, query, scamID, userID, ipHash)
	return err
}

// GetMyActivity returns scams reported by or experienced by a user.
func (r *ScamRepository) GetMyActivity(ctx context.Context, userID string) (reported []models.Scam, experienced []models.Scam, err error) {
	baseSelect := `
		SELECT DISTINCT
			s.id, s.title, s.description, s.type, s.report_count,
			s.date_first_reported, s.date_last_reported, s.status,
			s.estimated_losses, s.primary_location, s.risk_level,
			s.verification_status, s.scam_pattern, s.reporter_id,
			s.created_at, s.updated_at, s.last_analyzed_at
		FROM scams s`

	reportedQuery := baseSelect + ` WHERE s.reporter_id = $1 ORDER BY s.created_at DESC`
	experiencedQuery := `
		SELECT
			s.id, s.title, s.description, s.type, s.report_count,
			s.date_first_reported, s.date_last_reported, s.status,
			s.estimated_losses, s.primary_location, s.risk_level,
			s.verification_status, s.scam_pattern, s.reporter_id,
			s.created_at, s.updated_at, s.last_analyzed_at
		FROM scams s
		JOIN scam_experiences se ON se.scam_id = s.id
		WHERE se.user_id = $1 ORDER BY se.created_at DESC`

	reported = make([]models.Scam, 0)
	experienced = make([]models.Scam, 0)

	if err = r.db.SelectContext(ctx, &reported, reportedQuery, userID); err != nil {
		return nil, nil, fmt.Errorf("failed to fetch reported scams: %w", err)
	}
	if err = r.db.SelectContext(ctx, &experienced, experiencedQuery, userID); err != nil {
		return nil, nil, fmt.Errorf("failed to fetch experienced scams: %w", err)
	}

	return reported, experienced, nil
}
