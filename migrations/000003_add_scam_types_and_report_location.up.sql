-- Add scam_types lookup table
CREATE TABLE IF NOT EXISTS scam_types (
    slug        VARCHAR(100) PRIMARY KEY,
    label       VARCHAR(150) NOT NULL,
    description TEXT,
    icon        VARCHAR(50),
    created_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Seed known scam types (skip if already present)
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
    ('other',             'Other',                 'Scams that do not fit a standard category.')
ON CONFLICT (slug) DO NOTHING;

-- Normalise any free-text type values that exist in the DB to canonical slugs
UPDATE scams SET type = 'sim-swap'          WHERE LOWER(type) IN ('sim swap', 'sim-swap', 'simswap');
UPDATE scams SET type = 'phishing'          WHERE LOWER(type) IN ('phishing');
UPDATE scams SET type = 'investment-fraud'  WHERE LOWER(type) IN ('investment fraud', 'investment-fraud', 'investment scam');
UPDATE scams SET type = 'romance-scam'      WHERE LOWER(type) IN ('romance scam', 'romance-scam');
UPDATE scams SET type = 'job-scam'          WHERE LOWER(type) IN ('job scam', 'job-scam', 'employment scam');
UPDATE scams SET type = 'crypto-scam'       WHERE LOWER(type) IN ('crypto scam', 'crypto-scam', 'cryptocurrency scam');
UPDATE scams SET type = 'online-shopping'   WHERE LOWER(type) IN ('online shopping', 'online-shopping', 'shopping scam');
UPDATE scams SET type = 'advance-fee'       WHERE LOWER(type) IN ('advance fee', 'advance-fee', '419', 'nigerian scam');
UPDATE scams SET type = 'identity-theft'    WHERE LOWER(type) IN ('identity theft', 'identity-theft');
UPDATE scams SET type = 'vishing'           WHERE LOWER(type) IN ('vishing', 'voice scam');
UPDATE scams SET type = 'smishing'          WHERE LOWER(type) IN ('smishing', 'sms scam');
UPDATE scams SET type = 'rental-scam'       WHERE LOWER(type) IN ('rental scam', 'rental-scam');
UPDATE scams SET type = 'lottery-scam'      WHERE LOWER(type) IN ('lottery scam', 'lottery-scam', 'prize scam');
UPDATE scams SET type = 'tech-support-scam' WHERE LOWER(type) IN ('tech support scam', 'tech-support-scam', 'tech support');
-- Anything still unrecognised falls back to 'other'
UPDATE scams SET type = 'other'
    WHERE type NOT IN (SELECT slug FROM scam_types);

-- Add FK on scams.type referencing scam_types (only if not already set)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE constraint_name = 'scams_type_fkey'
          AND table_name = 'scams'
    ) THEN
        ALTER TABLE scams
            ADD CONSTRAINT scams_type_fkey
            FOREIGN KEY (type) REFERENCES scam_types(slug) ON UPDATE CASCADE;
    END IF;
END$$;

-- Add location columns to scam_reports (if not already present)
ALTER TABLE scam_reports
    ADD COLUMN IF NOT EXISTS city     VARCHAR(100),
    ADD COLUMN IF NOT EXISTS province VARCHAR(100),
    ADD COLUMN IF NOT EXISTS country  VARCHAR(100);
