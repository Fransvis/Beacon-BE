-- Money direction: how money moved in the scam
CREATE TYPE money_direction AS ENUM (
    'paid_scammer',
    'fake_payment_to_me',
    'used_as_mule',
    'info_only',
    'other'
);

ALTER TABLE scams
    ADD COLUMN IF NOT EXISTS money_direction money_direction,
    ADD COLUMN IF NOT EXISTS scammer_names   TEXT[];
