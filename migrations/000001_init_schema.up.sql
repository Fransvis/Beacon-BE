-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create enum types
CREATE TYPE scam_status AS ENUM ('ACTIVE', 'RESOLVED', 'UNDER_INVESTIGATION');
CREATE TYPE risk_level AS ENUM ('LOW', 'MEDIUM', 'HIGH');
CREATE TYPE verification_status AS ENUM ('VERIFIED', 'UNVERIFIED', 'DISPUTED');
CREATE TYPE report_status AS ENUM ('PENDING', 'VERIFIED', 'REJECTED');

-- Scam type lookup table (replaces free-text type on scams)
CREATE TABLE scam_types (
    slug        VARCHAR(100) PRIMARY KEY,
    label       VARCHAR(150) NOT NULL,
    description TEXT,
    icon        VARCHAR(50),
    created_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Seed known scam types
INSERT INTO scam_types (slug, label, description) VALUES
    ('sim-swap',          'SIM Swap',              'Criminals port your mobile number to take control of OTPs and banking.'),
    ('phishing',          'Phishing',              'Fake emails, SMSes or websites that steal credentials or personal info.'),
    ('investment-fraud',  'Investment Fraud',      'Ponzi schemes, fake trading platforms, and promises of high returns.'),
    ('romance-scam',      'Romance Scam',          'Fraudsters build fake relationships online to extract money.'),
    ('job-scam',          'Job Scam',              'Fake job offers requiring upfront fees or personal information.'),
    ('crypto-scam',       'Crypto Scam',           'Fraudulent cryptocurrency platforms, wallets, or trading bots.'),
    ('online-shopping',   'Online Shopping',       'Fake online stores or sellers that take payment and deliver nothing.'),
    ('advance-fee',       'Advance Fee / 419',     'Requests for an upfront fee in exchange for a larger promised payout.'),
    ('identity-theft',    'Identity Theft',        'Theft of personal information to commit fraud in your name.'),
    ('vishing',           'Vishing',               'Voice call scams impersonating banks, SARS, or government agencies.'),
    ('smishing',          'Smishing',              'SMS-based phishing targeting banking credentials or personal data.'),
    ('rental-scam',       'Rental Scam',           'Fake rental listings that collect deposits for non-existent properties.'),
    ('lottery-scam',      'Lottery / Prize Scam',  'Fake prize notifications requiring a fee to claim winnings.'),
    ('tech-support-scam', 'Tech Support Scam',     'Fake IT support that gains remote access or charges for non-existent issues.'),
    ('other',             'Other',                 'Scams that do not fit a standard category.');

-- Create base tables
CREATE TABLE scams (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    title VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    type VARCHAR(100) NOT NULL REFERENCES scam_types(slug) ON UPDATE CASCADE,
    report_count INT NOT NULL DEFAULT 0,
    date_first_reported TIMESTAMP WITH TIME ZONE NOT NULL,
    date_last_reported TIMESTAMP WITH TIME ZONE NOT NULL,
    status scam_status NOT NULL DEFAULT 'ACTIVE',
    estimated_losses DECIMAL(15,2) NOT NULL DEFAULT 0,
    primary_location VARCHAR(255),
    risk_level risk_level NOT NULL DEFAULT 'MEDIUM',
    verification_status verification_status NOT NULL DEFAULT 'UNVERIFIED',
    scam_pattern TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    last_analyzed_at TIMESTAMP WITH TIME ZONE
);

-- Location information
CREATE TABLE locations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    scam_id UUID NOT NULL REFERENCES scams(id) ON DELETE CASCADE,
    city VARCHAR(100),
    province VARCHAR(100),
    country VARCHAR(100) NOT NULL,
    report_count INT NOT NULL DEFAULT 1,
    latitude DECIMAL(10,8),
    longitude DECIMAL(11,8),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(scam_id, city, province, country)
);

-- Contact methods used by scammers
CREATE TABLE contact_methods (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    scam_id UUID NOT NULL REFERENCES scams(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL,
    value TEXT NOT NULL,
    is_valid BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(scam_id, type, value)
);

-- Money transfer methods
CREATE TABLE transfer_methods (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    scam_id UUID NOT NULL REFERENCES scams(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Victim demographics
CREATE TABLE demographics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    scam_id UUID NOT NULL REFERENCES scams(id) ON DELETE CASCADE,
    age_range VARCHAR(20),
    location VARCHAR(255),
    occupation VARCHAR(100),
    count INT NOT NULL DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Evidence attachments
CREATE TABLE evidence (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    scam_id UUID NOT NULL REFERENCES scams(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL,
    url TEXT NOT NULL,
    description TEXT,
    uploaded_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Related scams
CREATE TABLE related_scams (
    scam_id UUID NOT NULL REFERENCES scams(id) ON DELETE CASCADE,
    related_scam_id UUID NOT NULL REFERENCES scams(id) ON DELETE CASCADE,
    similarity_score DECIMAL(5,4),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (scam_id, related_scam_id),
    CHECK (scam_id != related_scam_id)
);

-- Keywords for search optimization
CREATE TABLE keywords (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    scam_id UUID NOT NULL REFERENCES scams(id) ON DELETE CASCADE,
    keyword VARCHAR(100) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    UNIQUE(scam_id, keyword)
);

-- Scam reports from users
CREATE TABLE scam_reports (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    scam_id UUID NOT NULL REFERENCES scams(id) ON DELETE CASCADE,
    reporter_email VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    loss_amount DECIMAL(15,2),
    date_occurred TIMESTAMP WITH TIME ZONE NOT NULL,
    -- Location of the reporter / where the scam occurred
    city VARCHAR(100),
    province VARCHAR(100),
    country VARCHAR(100),
    status report_status NOT NULL DEFAULT 'PENDING',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Indexes for better query performance
CREATE INDEX idx_scams_type ON scams(type);
CREATE INDEX idx_scams_status ON scams(status);
CREATE INDEX idx_scams_risk_level ON scams(risk_level);
CREATE INDEX idx_scams_verification_status ON scams(verification_status);
CREATE INDEX idx_scams_date_last_reported ON scams(date_last_reported);
CREATE INDEX idx_locations_country ON locations(country);
CREATE INDEX idx_contact_methods_type_value ON contact_methods(type, value);
CREATE INDEX idx_keywords_keyword ON keywords(keyword);
CREATE INDEX idx_scam_reports_status ON scam_reports(status);

-- Full text search
ALTER TABLE scams ADD COLUMN search_vector tsvector;
CREATE INDEX idx_scams_search ON scams USING gin(search_vector);

-- Update trigger for search vector
CREATE FUNCTION scams_search_vector_update() RETURNS trigger AS $$
BEGIN
    NEW.search_vector :=
        setweight(to_tsvector('english', COALESCE(NEW.title, '')), 'A') ||
        setweight(to_tsvector('english', COALESCE(NEW.description, '')), 'B') ||
        setweight(to_tsvector('english', COALESCE(NEW.type, '')), 'C') ||
        setweight(to_tsvector('english', COALESCE(NEW.scam_pattern, '')), 'C');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER scams_search_vector_update
    BEFORE INSERT OR UPDATE ON scams
    FOR EACH ROW
    EXECUTE FUNCTION scams_search_vector_update();
