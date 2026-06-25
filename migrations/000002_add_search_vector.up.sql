-- Add search_vector column
ALTER TABLE scams ADD COLUMN IF NOT EXISTS search_vector tsvector;

-- Create function to automatically update search_vector
CREATE OR REPLACE FUNCTION scams_search_vector_update() RETURNS trigger AS $$
BEGIN
  NEW.search_vector :=
    setweight(to_tsvector('english', COALESCE(NEW.title, '')), 'A') ||
    setweight(to_tsvector('english', COALESCE(NEW.description, '')), 'B') ||
    setweight(to_tsvector('english', COALESCE(NEW.type, '')), 'C');
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger to automatically update search_vector
DROP TRIGGER IF EXISTS scams_search_vector_update ON scams;
CREATE TRIGGER scams_search_vector_update
  BEFORE INSERT OR UPDATE
  ON scams
  FOR EACH ROW
  EXECUTE FUNCTION scams_search_vector_update();

-- Update existing records
UPDATE scams SET
  search_vector = setweight(to_tsvector('english', COALESCE(title, '')), 'A') ||
                 setweight(to_tsvector('english', COALESCE(description, '')), 'B') ||
                 setweight(to_tsvector('english', COALESCE(type, '')), 'C');

-- Create index for full-text search
CREATE INDEX IF NOT EXISTS idx_scams_search_vector ON scams USING gin(search_vector);
