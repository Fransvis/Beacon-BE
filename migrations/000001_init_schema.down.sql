-- Drop triggers
DROP TRIGGER IF EXISTS scams_search_vector_update ON scams;
DROP FUNCTION IF EXISTS scams_search_vector_update();

-- Drop tables
DROP TABLE IF EXISTS scam_reports;
DROP TABLE IF EXISTS keywords;
DROP TABLE IF EXISTS related_scams;
DROP TABLE IF EXISTS evidence;
DROP TABLE IF EXISTS demographics;
DROP TABLE IF EXISTS transfer_methods;
DROP TABLE IF EXISTS contact_methods;
DROP TABLE IF EXISTS locations;
DROP TABLE IF EXISTS scams;

-- Drop enum types
DROP TYPE IF EXISTS report_status;
DROP TYPE IF EXISTS verification_status;
DROP TYPE IF EXISTS risk_level;
DROP TYPE IF EXISTS scam_status;
