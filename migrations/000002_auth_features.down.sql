DROP TABLE IF EXISTS two_factors;
ALTER TABLE users DROP COLUMN IF EXISTS two_factor_enabled;
