-- migrate:up

-- Box 15 — State code and employer state ID number
ALTER TABLE employees ADD COLUMN orig_state_code TEXT NOT NULL DEFAULT '';
ALTER TABLE employees ADD COLUMN corr_state_code TEXT NOT NULL DEFAULT '';
ALTER TABLE employees ADD COLUMN orig_state_id   TEXT NOT NULL DEFAULT '';
ALTER TABLE employees ADD COLUMN corr_state_id   TEXT NOT NULL DEFAULT '';

-- Box 16 — State wages, tips, etc.
ALTER TABLE employees ADD COLUMN orig_state_wages INTEGER NOT NULL DEFAULT 0;
ALTER TABLE employees ADD COLUMN corr_state_wages INTEGER NOT NULL DEFAULT 0;

-- Box 17 — State income tax
ALTER TABLE employees ADD COLUMN orig_state_tax INTEGER NOT NULL DEFAULT 0;
ALTER TABLE employees ADD COLUMN corr_state_tax INTEGER NOT NULL DEFAULT 0;

-- Box 18 — Local wages, tips, etc.
ALTER TABLE employees ADD COLUMN orig_local_wages INTEGER NOT NULL DEFAULT 0;
ALTER TABLE employees ADD COLUMN corr_local_wages INTEGER NOT NULL DEFAULT 0;

-- Box 19 — Local income tax
ALTER TABLE employees ADD COLUMN orig_local_tax INTEGER NOT NULL DEFAULT 0;
ALTER TABLE employees ADD COLUMN corr_local_tax INTEGER NOT NULL DEFAULT 0;

-- Box 20 — Locality name
ALTER TABLE employees ADD COLUMN orig_locality_name TEXT NOT NULL DEFAULT '';
ALTER TABLE employees ADD COLUMN corr_locality_name TEXT NOT NULL DEFAULT '';

-- migrate:down
ALTER TABLE employees DROP COLUMN orig_state_code;
ALTER TABLE employees DROP COLUMN corr_state_code;
ALTER TABLE employees DROP COLUMN orig_state_id;
ALTER TABLE employees DROP COLUMN corr_state_id;
ALTER TABLE employees DROP COLUMN orig_state_wages;
ALTER TABLE employees DROP COLUMN corr_state_wages;
ALTER TABLE employees DROP COLUMN orig_state_tax;
ALTER TABLE employees DROP COLUMN corr_state_tax;
ALTER TABLE employees DROP COLUMN orig_local_wages;
ALTER TABLE employees DROP COLUMN corr_local_wages;
ALTER TABLE employees DROP COLUMN orig_local_tax;
ALTER TABLE employees DROP COLUMN corr_local_tax;
ALTER TABLE employees DROP COLUMN orig_locality_name;
ALTER TABLE employees DROP COLUMN corr_locality_name;
