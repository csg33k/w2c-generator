-- migrate:up
ALTER TABLE submissions ADD COLUMN tax_year TEXT NOT NULL DEFAULT '2021';

-- migrate:down
-- SQLite 3.35+ supports DROP COLUMN directly
ALTER TABLE submissions DROP COLUMN tax_year;