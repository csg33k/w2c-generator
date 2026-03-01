CREATE TABLE IF NOT EXISTS "schema_migrations" (version varchar(128) primary key);
CREATE TABLE submissions (
                                           id               INTEGER PRIMARY KEY AUTOINCREMENT,
                                           ein              TEXT    NOT NULL,
                                           employer_name    TEXT    NOT NULL,
                                           addr1            TEXT,
                                           addr2            TEXT,
                                           city             TEXT,
                                           state            TEXT,
                                           zip              TEXT,
                                           zip_ext          TEXT,
                                           agent_indicator  TEXT    NOT NULL DEFAULT '0',
                                           agent_ein        TEXT    NOT NULL DEFAULT '',
                                           terminating      INTEGER NOT NULL DEFAULT 0,
                                           notes            TEXT    NOT NULL DEFAULT '',
                                           created_at       DATETIME NOT NULL,
                                           submitted_at     DATETIME
, bso_uid          TEXT NOT NULL DEFAULT '', contact_name     TEXT NOT NULL DEFAULT '', contact_phone    TEXT NOT NULL DEFAULT '', contact_email    TEXT NOT NULL DEFAULT '', preparer_code    TEXT NOT NULL DEFAULT 'L', kind_of_employer       TEXT NOT NULL DEFAULT 'N', employer_contact_name  TEXT NOT NULL DEFAULT '', employer_contact_phone TEXT NOT NULL DEFAULT '', employer_contact_email TEXT NOT NULL DEFAULT '', employment_code        TEXT NOT NULL DEFAULT 'R', tax_year TEXT NOT NULL DEFAULT '2021');
CREATE TABLE employees (
                                         id             INTEGER PRIMARY KEY AUTOINCREMENT,
                                         submission_id  INTEGER NOT NULL REFERENCES submissions(id) ON DELETE CASCADE,
                                         ssn            TEXT    NOT NULL,
                                         original_ssn   TEXT    NOT NULL DEFAULT '',
                                         first_name     TEXT,
                                         middle_name    TEXT,
                                         last_name      TEXT,
                                         suffix         TEXT    NOT NULL DEFAULT '',
                                         addr1          TEXT,
                                         addr2          TEXT,
                                         city           TEXT,
                                         state          TEXT,
                                         zip            TEXT,
                                         zip_ext        TEXT,
                                         orig_wages     INTEGER NOT NULL DEFAULT 0,
                                         corr_wages     INTEGER NOT NULL DEFAULT 0,
                                         orig_ss_wages  INTEGER NOT NULL DEFAULT 0,
                                         corr_ss_wages  INTEGER NOT NULL DEFAULT 0,
                                         orig_med_wages INTEGER NOT NULL DEFAULT 0,
                                         corr_med_wages INTEGER NOT NULL DEFAULT 0,
                                         orig_fed_tax   INTEGER NOT NULL DEFAULT 0,
                                         corr_fed_tax   INTEGER NOT NULL DEFAULT 0,
                                         orig_ss_tax    INTEGER NOT NULL DEFAULT 0,
                                         corr_ss_tax    INTEGER NOT NULL DEFAULT 0,
                                         orig_med_tax   INTEGER NOT NULL DEFAULT 0,
                                         corr_med_tax   INTEGER NOT NULL DEFAULT 0,
                                         created_at     DATETIME NOT NULL,
                                         updated_at     DATETIME NOT NULL
, orig_ss_tips INTEGER NOT NULL DEFAULT 0, corr_ss_tips INTEGER NOT NULL DEFAULT 0);
-- Dbmate schema migrations
INSERT INTO "schema_migrations" (version) VALUES
  ('20260228000001'),
  ('20260228000002'),
  ('20260301170046');
