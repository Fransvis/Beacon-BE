ALTER TABLE scams
    DROP COLUMN IF EXISTS money_direction,
    DROP COLUMN IF EXISTS scammer_names;

DROP TYPE IF EXISTS money_direction;
