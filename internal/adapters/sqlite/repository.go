package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/csg33k/w2c-generator/internal/domain"
)

type Repository struct {
	db *sql.DB
}

func New(dsn string) (*Repository, error) {
	db, err := sql.Open("sqlite3", dsn+"?_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	r := &Repository{db: db}
	if err := r.migrate(); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return r, nil
}

// migration is a single versioned schema change, tracked in schema_migrations.
// Migrations run in order on every startup; already-applied ones are skipped.
// Never edit a migration that has shipped — add a new one instead.
type migration struct {
	version int
	sql     string
}

var migrations = []migration{
	{
		// Initial schema.
		version: 1,
		sql: `
		CREATE TABLE IF NOT EXISTS submissions (
			id              INTEGER PRIMARY KEY AUTOINCREMENT,
			ein             TEXT    NOT NULL,
			employer_name   TEXT    NOT NULL,
			addr1           TEXT    DEFAULT '',
			addr2           TEXT    DEFAULT '',
			city            TEXT    DEFAULT '',
			state           TEXT    DEFAULT '',
			zip             TEXT    DEFAULT '',
			zip_ext         TEXT    DEFAULT '',
			agent_indicator TEXT    DEFAULT '0',
			agent_ein       TEXT    DEFAULT '',
			terminating     INTEGER DEFAULT 0,
			notes           TEXT    DEFAULT '',
			created_at      DATETIME NOT NULL,
			submitted_at    DATETIME
		);
		CREATE TABLE IF NOT EXISTS employees (
			id             INTEGER PRIMARY KEY AUTOINCREMENT,
			submission_id  INTEGER NOT NULL REFERENCES submissions(id) ON DELETE CASCADE,
			ssn            TEXT    NOT NULL,
			original_ssn   TEXT    DEFAULT '',
			first_name     TEXT    DEFAULT '',
			middle_name    TEXT    DEFAULT '',
			last_name      TEXT    DEFAULT '',
			suffix         TEXT    DEFAULT '',
			addr1          TEXT    DEFAULT '',
			addr2          TEXT    DEFAULT '',
			city           TEXT    DEFAULT '',
			state          TEXT    DEFAULT '',
			zip            TEXT    DEFAULT '',
			zip_ext        TEXT    DEFAULT '',
			orig_wages     INTEGER DEFAULT 0,
			corr_wages     INTEGER DEFAULT 0,
			orig_ss_wages  INTEGER DEFAULT 0,
			corr_ss_wages  INTEGER DEFAULT 0,
			orig_med_wages INTEGER DEFAULT 0,
			corr_med_wages INTEGER DEFAULT 0,
			orig_fed_tax   INTEGER DEFAULT 0,
			corr_fed_tax   INTEGER DEFAULT 0,
			orig_ss_tax    INTEGER DEFAULT 0,
			corr_ss_tax    INTEGER DEFAULT 0,
			orig_med_tax   INTEGER DEFAULT 0,
			corr_med_tax   INTEGER DEFAULT 0,
			created_at     DATETIME NOT NULL,
			updated_at     DATETIME NOT NULL
		)`,
	},
	{
		// Add BSO submitter fields to submissions (required by AccuWage RCA record).
		// Existing rows get sensible defaults and can be edited through the UI.
		version: 2,
		sql: `
		ALTER TABLE submissions ADD COLUMN bso_uid         TEXT DEFAULT '';
		ALTER TABLE submissions ADD COLUMN contact_name    TEXT DEFAULT '';
		ALTER TABLE submissions ADD COLUMN contact_phone   TEXT DEFAULT '';
		ALTER TABLE submissions ADD COLUMN contact_email   TEXT DEFAULT '';
		ALTER TABLE submissions ADD COLUMN preparer_code   TEXT DEFAULT 'L';
		ALTER TABLE submissions ADD COLUMN resub_indicator TEXT DEFAULT '0';
		ALTER TABLE submissions ADD COLUMN resub_wfid      TEXT DEFAULT ''`,
	},
}

func (r *Repository) migrate() error {
	if _, err := r.db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    INTEGER PRIMARY KEY,
			applied_at DATETIME NOT NULL
		)`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	for _, m := range migrations {
		var count int
		if err := r.db.QueryRow(
			`SELECT COUNT(*) FROM schema_migrations WHERE version=?`, m.version,
		).Scan(&count); err != nil {
			return fmt.Errorf("check migration %d: %w", m.version, err)
		}
		if count > 0 {
			continue // already applied
		}

		// SQLite requires statements to be executed one at a time.
		for _, stmt := range splitSQL(m.sql) {
			if _, err := r.db.Exec(stmt); err != nil {
				return fmt.Errorf("migration %d %q: %w", m.version, stmt[:min(40, len(stmt))], err)
			}
		}

		if _, err := r.db.Exec(
			`INSERT INTO schema_migrations (version, applied_at) VALUES (?,?)`,
			m.version, time.Now(),
		); err != nil {
			return fmt.Errorf("record migration %d: %w", m.version, err)
		}
	}
	return nil
}

// splitSQL splits a semicolon-delimited SQL string into individual statements.
func splitSQL(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ";") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (r *Repository) CreateSubmission(ctx context.Context, s *domain.Submission) error {
	s.CreatedAt = time.Now()
	res, err := r.db.ExecContext(ctx, `
		INSERT INTO submissions (
			bso_uid, contact_name, contact_phone, contact_email, preparer_code,
			resub_indicator, resub_wfid,
			ein, employer_name, addr1, addr2, city, state, zip, zip_ext,
			agent_indicator, agent_ein, terminating,
			notes, created_at
		) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		s.Submitter.BSOUID, s.Submitter.ContactName, s.Submitter.ContactPhone,
		s.Submitter.ContactEmail, s.Submitter.PreparerCode,
		s.Submitter.ResubIndicator, s.Submitter.ResubWFID,
		s.Employer.EIN, s.Employer.Name,
		s.Employer.AddressLine1, s.Employer.AddressLine2,
		s.Employer.City, s.Employer.State, s.Employer.ZIP, s.Employer.ZIPExtension,
		s.Employer.AgentIndicator, s.Employer.AgentEIN,
		boolToInt(s.Employer.TerminatingBusiness),
		s.Notes, s.CreatedAt,
	)
	if err != nil {
		return err
	}
	id, _ := res.LastInsertId()
	s.ID = id
	return nil
}

func (r *Repository) GetSubmission(ctx context.Context, id int64) (*domain.Submission, error) {
	s := &domain.Submission{}
	var terminating int
	var submittedAt sql.NullTime
	err := r.db.QueryRowContext(ctx, `
		SELECT id,
		       bso_uid, contact_name, contact_phone, contact_email, preparer_code,
		       resub_indicator, resub_wfid,
		       ein, employer_name, addr1, addr2, city, state, zip, zip_ext,
		       agent_indicator, agent_ein, terminating,
		       notes, created_at, submitted_at
		FROM submissions WHERE id=?`, id).Scan(
		&s.ID,
		&s.Submitter.BSOUID, &s.Submitter.ContactName, &s.Submitter.ContactPhone,
		&s.Submitter.ContactEmail, &s.Submitter.PreparerCode,
		&s.Submitter.ResubIndicator, &s.Submitter.ResubWFID,
		&s.Employer.EIN, &s.Employer.Name,
		&s.Employer.AddressLine1, &s.Employer.AddressLine2,
		&s.Employer.City, &s.Employer.State, &s.Employer.ZIP, &s.Employer.ZIPExtension,
		&s.Employer.AgentIndicator, &s.Employer.AgentEIN,
		&terminating,
		&s.Notes, &s.CreatedAt, &submittedAt,
	)
	if err != nil {
		return nil, err
	}
	s.Employer.TerminatingBusiness = terminating == 1
	s.Employer.TaxYear = domain.TaxYear2021
	if submittedAt.Valid {
		s.SubmittedAt = &submittedAt.Time
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, submission_id, ssn, original_ssn,
		       first_name, middle_name, last_name, suffix,
		       addr1, addr2, city, state, zip, zip_ext,
		       orig_wages, corr_wages, orig_ss_wages, corr_ss_wages,
		       orig_med_wages, corr_med_wages, orig_fed_tax, corr_fed_tax,
		       orig_ss_tax, corr_ss_tax, orig_med_tax, corr_med_tax,
		       created_at, updated_at
		FROM employees WHERE submission_id=? ORDER BY id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var e domain.EmployeeRecord
		if err := rows.Scan(
			&e.ID, &e.SubmissionID, &e.SSN, &e.OriginalSSN,
			&e.FirstName, &e.MiddleName, &e.LastName, &e.Suffix,
			&e.AddressLine1, &e.AddressLine2, &e.City, &e.State, &e.ZIP, &e.ZIPExtension,
			&e.Amounts.OriginalWagesTipsOther, &e.Amounts.CorrectWagesTipsOther,
			&e.Amounts.OriginalSocialSecurityWages, &e.Amounts.CorrectSocialSecurityWages,
			&e.Amounts.OriginalMedicareWages, &e.Amounts.CorrectMedicareWages,
			&e.Amounts.OriginalFederalIncomeTax, &e.Amounts.CorrectFederalIncomeTax,
			&e.Amounts.OriginalSocialSecurityTax, &e.Amounts.CorrectSocialSecurityTax,
			&e.Amounts.OriginalMedicareTax, &e.Amounts.CorrectMedicareTax,
			&e.CreatedAt, &e.UpdatedAt,
		); err != nil {
			return nil, err
		}
		s.Employees = append(s.Employees, e)
	}
	return s, nil
}

func (r *Repository) ListSubmissions(ctx context.Context) ([]domain.Submission, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, ein, employer_name, notes, created_at
		FROM submissions ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []domain.Submission
	for rows.Next() {
		var s domain.Submission
		if err := rows.Scan(
			&s.ID, &s.Employer.EIN, &s.Employer.Name,
			&s.Notes, &s.CreatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, s)
	}
	return list, nil
}

func (r *Repository) DeleteSubmission(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM submissions WHERE id=?`, id)
	return err
}

func (r *Repository) AddEmployee(ctx context.Context, submissionID int64, e *domain.EmployeeRecord) error {
	now := time.Now()
	e.SubmissionID = submissionID
	e.CreatedAt = now
	e.UpdatedAt = now
	res, err := r.db.ExecContext(ctx, `
		INSERT INTO employees (
			submission_id, ssn, original_ssn,
			first_name, middle_name, last_name, suffix,
			addr1, addr2, city, state, zip, zip_ext,
			orig_wages, corr_wages, orig_ss_wages, corr_ss_wages,
			orig_med_wages, corr_med_wages, orig_fed_tax, corr_fed_tax,
			orig_ss_tax, corr_ss_tax, orig_med_tax, corr_med_tax,
			created_at, updated_at
		) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		submissionID, e.SSN, e.OriginalSSN,
		e.FirstName, e.MiddleName, e.LastName, e.Suffix,
		e.AddressLine1, e.AddressLine2, e.City, e.State, e.ZIP, e.ZIPExtension,
		e.Amounts.OriginalWagesTipsOther, e.Amounts.CorrectWagesTipsOther,
		e.Amounts.OriginalSocialSecurityWages, e.Amounts.CorrectSocialSecurityWages,
		e.Amounts.OriginalMedicareWages, e.Amounts.CorrectMedicareWages,
		e.Amounts.OriginalFederalIncomeTax, e.Amounts.CorrectFederalIncomeTax,
		e.Amounts.OriginalSocialSecurityTax, e.Amounts.CorrectSocialSecurityTax,
		e.Amounts.OriginalMedicareTax, e.Amounts.CorrectMedicareTax,
		now, now,
	)
	if err != nil {
		return err
	}
	id, _ := res.LastInsertId()
	e.ID = id
	return nil
}

func (r *Repository) DeleteEmployee(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM employees WHERE id=?`, id)
	return err
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
