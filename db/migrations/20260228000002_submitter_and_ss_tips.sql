-- migrate:up

-- RCA / submitter fields (stored on the submission row)
ALTER TABLE submissions ADD COLUMN bso_uid          TEXT NOT NULL DEFAULT '';
ALTER TABLE submissions ADD COLUMN contact_name     TEXT NOT NULL DEFAULT '';
ALTER TABLE submissions ADD COLUMN contact_phone    TEXT NOT NULL DEFAULT '';
ALTER TABLE submissions ADD COLUMN contact_email    TEXT NOT NULL DEFAULT '';
ALTER TABLE submissions ADD COLUMN preparer_code    TEXT NOT NULL DEFAULT 'L';

-- RCE employer contact + kind-of-employer
ALTER TABLE submissions ADD COLUMN kind_of_employer       TEXT NOT NULL DEFAULT 'N';
ALTER TABLE submissions ADD COLUMN employer_contact_name  TEXT NOT NULL DEFAULT '';
ALTER TABLE submissions ADD COLUMN employer_contact_phone TEXT NOT NULL DEFAULT '';
ALTER TABLE submissions ADD COLUMN employer_contact_email TEXT NOT NULL DEFAULT '';
ALTER TABLE submissions ADD COLUMN employment_code        TEXT NOT NULL DEFAULT 'R';

-- RCW Box 7/8 — Social Security Tips
ALTER TABLE employees ADD COLUMN orig_ss_tips INTEGER NOT NULL DEFAULT 0;
ALTER TABLE employees ADD COLUMN corr_ss_tips INTEGER NOT NULL DEFAULT 0;

-- migrate:down
-- SQLite does not support DROP COLUMN before 3.35; list what we can't easily undo.
-- Re-create tables from scratch if rolling back on older SQLite.
SELECT 1; -- no-op placeholder