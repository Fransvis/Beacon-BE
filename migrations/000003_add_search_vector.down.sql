-- Drop trigger and function
DROP TRIGGER IF EXISTS scams_search_vector_update ON scams;
DROP FUNCTION IF EXISTS scams_search_vector_update();

-- Drop search vector column and index
DROP INDEX IF EXISTS idx_scams_search_vector;
ALTER TABLE scams DROP COLUMN IF EXISTS search_vector;
