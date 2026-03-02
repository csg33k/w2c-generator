-- migrate:up

-- Name correction fields (for correcting previously reported wrong name)
ALTER TABLE employees ADD COLUMN orig_first_name  TEXT NOT NULL DEFAULT '';
ALTER TABLE employees ADD COLUMN orig_middle_name TEXT NOT NULL DEFAULT '';
ALTER TABLE employees ADD COLUMN orig_last_name   TEXT NOT NULL DEFAULT '';
ALTER TABLE employees ADD COLUMN orig_suffix       TEXT NOT NULL DEFAULT '';

-- Box 8 — Allocated Tips (goes in RCO record)
ALTER TABLE employees ADD COLUMN orig_alloc_tips  INTEGER NOT NULL DEFAULT 0;
ALTER TABLE employees ADD COLUMN corr_alloc_tips  INTEGER NOT NULL DEFAULT 0;

-- Box 10 — Dependent Care Benefits
ALTER TABLE employees ADD COLUMN orig_dep_care    INTEGER NOT NULL DEFAULT 0;
ALTER TABLE employees ADD COLUMN corr_dep_care    INTEGER NOT NULL DEFAULT 0;

-- Box 11 — Nonqualified Plans (two components per spec)
ALTER TABLE employees ADD COLUMN orig_nonqual_457     INTEGER NOT NULL DEFAULT 0;
ALTER TABLE employees ADD COLUMN corr_nonqual_457     INTEGER NOT NULL DEFAULT 0;
ALTER TABLE employees ADD COLUMN orig_nonqual_not457  INTEGER NOT NULL DEFAULT 0;
ALTER TABLE employees ADD COLUMN corr_nonqual_not457  INTEGER NOT NULL DEFAULT 0;

-- Box 12 codes
ALTER TABLE employees ADD COLUMN orig_code_d       INTEGER NOT NULL DEFAULT 0; -- 401(k)
ALTER TABLE employees ADD COLUMN corr_code_d       INTEGER NOT NULL DEFAULT 0;
ALTER TABLE employees ADD COLUMN orig_code_e       INTEGER NOT NULL DEFAULT 0; -- 403(b)
ALTER TABLE employees ADD COLUMN corr_code_e       INTEGER NOT NULL DEFAULT 0;
ALTER TABLE employees ADD COLUMN orig_code_g       INTEGER NOT NULL DEFAULT 0; -- 457(b) govt
ALTER TABLE employees ADD COLUMN corr_code_g       INTEGER NOT NULL DEFAULT 0;
ALTER TABLE employees ADD COLUMN orig_code_w       INTEGER NOT NULL DEFAULT 0; -- HSA
ALTER TABLE employees ADD COLUMN corr_code_w       INTEGER NOT NULL DEFAULT 0;
ALTER TABLE employees ADD COLUMN orig_code_aa      INTEGER NOT NULL DEFAULT 0; -- Roth 401(k)
ALTER TABLE employees ADD COLUMN corr_code_aa      INTEGER NOT NULL DEFAULT 0;
ALTER TABLE employees ADD COLUMN orig_code_bb      INTEGER NOT NULL DEFAULT 0; -- Roth 403(b)
ALTER TABLE employees ADD COLUMN corr_code_bb      INTEGER NOT NULL DEFAULT 0;
ALTER TABLE employees ADD COLUMN orig_code_dd      INTEGER NOT NULL DEFAULT 0; -- employer health
ALTER TABLE employees ADD COLUMN corr_code_dd      INTEGER NOT NULL DEFAULT 0;

-- Box 13 checkbox corrections (stored as 0/1/NULL: NULL=no correction, 0=unchecked, 1=checked)
ALTER TABLE employees ADD COLUMN orig_statutory_emp    INTEGER;  -- NULL = no correction
ALTER TABLE employees ADD COLUMN corr_statutory_emp    INTEGER;
ALTER TABLE employees ADD COLUMN orig_retirement_plan  INTEGER;
ALTER TABLE employees ADD COLUMN corr_retirement_plan  INTEGER;
ALTER TABLE employees ADD COLUMN orig_third_party_sick INTEGER;
ALTER TABLE employees ADD COLUMN corr_third_party_sick INTEGER;

-- Employer EIN correction field on submissions
ALTER TABLE submissions ADD COLUMN orig_ein TEXT NOT NULL DEFAULT '';

-- migrate:down
-- SQLite does not support DROP COLUMN in older versions; migration is intentionally irreversible.
