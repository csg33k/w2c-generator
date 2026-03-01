-- migrate:up
CREATE TABLE IF NOT EXISTS submissions (
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
);

CREATE TABLE IF NOT EXISTS employees (
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
);

-- migrate:down
DROP TABLE IF EXISTS employees;
DROP TABLE IF EXISTS submissions;