package sqlite

import (
	"context"
	"database/sql"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/csg33k/w2c-generator/internal/domain"
)

type Repository struct {
	db *sql.DB
}

// New opens the SQLite database. Schema migrations are managed by dbmate;
// run `dbmate up` before starting the server.
func New(dsn string) (*Repository, error) {
	db, err := sql.Open("sqlite3", dsn+"?_foreign_keys=on")
	if err != nil {
		return nil, err
	}
	return &Repository{db: db}, nil
}

// ── Submissions ───────────────────────────────────────────────────────────────

func (r *Repository) CreateSubmission(ctx context.Context, s *domain.Submission) error {
	s.CreatedAt = time.Now()
	res, err := r.db.ExecContext(ctx, `
		INSERT INTO submissions (
			ein, employer_name, addr1, addr2, city, state, zip, zip_ext,
			agent_indicator, agent_ein, terminating, notes,
			bso_uid, contact_name, contact_phone, contact_email, preparer_code,
			employment_code, kind_of_employer,
			employer_contact_name, employer_contact_phone, employer_contact_email,
		    created_at, tax_year
	    ) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		s.Employer.EIN, s.Employer.Name,
		s.Employer.AddressLine1, s.Employer.AddressLine2,
		s.Employer.City, s.Employer.State, s.Employer.ZIP, s.Employer.ZIPExtension,
		s.Employer.AgentIndicator, s.Employer.AgentEIN,
		boolToInt(s.Employer.TerminatingBusiness),
		s.Notes,
		s.Submitter.BSOUID, s.Submitter.ContactName,
		s.Submitter.ContactPhone, s.Submitter.ContactEmail, s.Submitter.PreparerCode,
		s.Employer.EmploymentCode, s.Employer.KindOfEmployer,
		s.Employer.ContactName, s.Employer.ContactPhone, s.Employer.ContactEmail,
		s.CreatedAt, s.Employer.TaxYear,
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
		SELECT id, ein, employer_name, addr1, addr2, city, state, zip, zip_ext,
		       agent_indicator, agent_ein, terminating, notes,
		       bso_uid, contact_name, contact_phone, contact_email, preparer_code,
		       employment_code, kind_of_employer,
		       employer_contact_name, employer_contact_phone, employer_contact_email,
		       created_at, submitted_at, tax_year
		FROM submissions WHERE id=?`, id).Scan(
		&s.ID, &s.Employer.EIN, &s.Employer.Name,
		&s.Employer.AddressLine1, &s.Employer.AddressLine2,
		&s.Employer.City, &s.Employer.State, &s.Employer.ZIP, &s.Employer.ZIPExtension,
		&s.Employer.AgentIndicator, &s.Employer.AgentEIN,
		&terminating, &s.Notes,
		&s.Submitter.BSOUID, &s.Submitter.ContactName,
		&s.Submitter.ContactPhone, &s.Submitter.ContactEmail, &s.Submitter.PreparerCode,
		&s.Employer.EmploymentCode, &s.Employer.KindOfEmployer,
		&s.Employer.ContactName, &s.Employer.ContactPhone, &s.Employer.ContactEmail,
		&s.CreatedAt, &submittedAt, &s.Employer.TaxYear,
	)
	if err != nil {
		return nil, err
	}
	s.Employer.TerminatingBusiness = terminating == 1
	// TaxYear is now persisted; only fall back if the column was empty
	// (e.g. rows created before migration 0003).
	if s.Employer.TaxYear == "" {
		s.Employer.TaxYear = domain.DefaultTaxYear
	}
	if submittedAt.Valid {
		s.SubmittedAt = &submittedAt.Time
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, submission_id, ssn, original_ssn,
		       first_name, middle_name, last_name, suffix,
		       addr1, addr2, city, state, zip, zip_ext,
		       orig_wages, corr_wages,
		       orig_ss_wages, corr_ss_wages,
		       orig_med_wages, corr_med_wages,
		       orig_fed_tax, corr_fed_tax,
		       orig_ss_tax, corr_ss_tax,
		       orig_med_tax, corr_med_tax,
		       orig_ss_tips, corr_ss_tips,
		       orig_state_code, corr_state_code,
		       orig_state_id, corr_state_id,
		       orig_state_wages, corr_state_wages,
		       orig_state_tax, corr_state_tax,
		       orig_local_wages, corr_local_wages,
		       orig_local_tax, corr_local_tax,
		       orig_locality_name, corr_locality_name,
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
			&e.Amounts.OriginalSocialSecurityTips, &e.Amounts.CorrectSocialSecurityTips,
			&e.OriginalStateCode, &e.CorrectStateCode,
			&e.OriginalStateIDNumber, &e.CorrectStateIDNumber,
			&e.Amounts.OriginalStateWages, &e.Amounts.CorrectStateWages,
			&e.Amounts.OriginalStateIncomeTax, &e.Amounts.CorrectStateIncomeTax,
			&e.Amounts.OriginalLocalWages, &e.Amounts.CorrectLocalWages,
			&e.Amounts.OriginalLocalIncomeTax, &e.Amounts.CorrectLocalIncomeTax,
			&e.OriginalLocalityName, &e.CorrectLocalityName,
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
		if err := rows.Scan(&s.ID, &s.Employer.EIN, &s.Employer.Name, &s.Notes, &s.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, s)
	}
	return list, nil
}

func (r *Repository) UpdateSubmission(ctx context.Context, s *domain.Submission) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE submissions
		SET ein=?, employer_name=?, addr1=?, addr2=?, city=?, state=?, zip=?, zip_ext=?,
		    agent_indicator=?, agent_ein=?, terminating=?, notes=?,
		    bso_uid=?, contact_name=?, contact_phone=?, contact_email=?, preparer_code=?,
		    employment_code=?, kind_of_employer=?,
		    employer_contact_name=?, employer_contact_phone=?, employer_contact_email=?,
		    tax_year=?
        WHERE id=?`,
		s.Employer.EIN, s.Employer.Name,
		s.Employer.AddressLine1, s.Employer.AddressLine2,
		s.Employer.City, s.Employer.State, s.Employer.ZIP, s.Employer.ZIPExtension,
		s.Employer.AgentIndicator, s.Employer.AgentEIN,
		boolToInt(s.Employer.TerminatingBusiness),
		s.Notes,
		s.Submitter.BSOUID, s.Submitter.ContactName,
		s.Submitter.ContactPhone, s.Submitter.ContactEmail, s.Submitter.PreparerCode,
		s.Employer.EmploymentCode, s.Employer.KindOfEmployer,
		s.Employer.ContactName, s.Employer.ContactPhone, s.Employer.ContactEmail,
		s.Employer.TaxYear, s.ID,
	)
	return err
}

func (r *Repository) DeleteSubmission(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM submissions WHERE id=?`, id)
	return err
}

// ── Employees ─────────────────────────────────────────────────────────────────

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
			orig_wages, corr_wages,
			orig_ss_wages, corr_ss_wages,
			orig_med_wages, corr_med_wages,
			orig_fed_tax, corr_fed_tax,
			orig_ss_tax, corr_ss_tax,
			orig_med_tax, corr_med_tax,
			orig_ss_tips, corr_ss_tips,
			orig_state_code, corr_state_code,
			orig_state_id, corr_state_id,
			orig_state_wages, corr_state_wages,
			orig_state_tax, corr_state_tax,
			orig_local_wages, corr_local_wages,
			orig_local_tax, corr_local_tax,
			orig_locality_name, corr_locality_name,
			created_at, updated_at
		) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		submissionID, e.SSN, e.OriginalSSN,
		e.FirstName, e.MiddleName, e.LastName, e.Suffix,
		e.AddressLine1, e.AddressLine2, e.City, e.State, e.ZIP, e.ZIPExtension,
		e.Amounts.OriginalWagesTipsOther, e.Amounts.CorrectWagesTipsOther,
		e.Amounts.OriginalSocialSecurityWages, e.Amounts.CorrectSocialSecurityWages,
		e.Amounts.OriginalMedicareWages, e.Amounts.CorrectMedicareWages,
		e.Amounts.OriginalFederalIncomeTax, e.Amounts.CorrectFederalIncomeTax,
		e.Amounts.OriginalSocialSecurityTax, e.Amounts.CorrectSocialSecurityTax,
		e.Amounts.OriginalMedicareTax, e.Amounts.CorrectMedicareTax,
		e.Amounts.OriginalSocialSecurityTips, e.Amounts.CorrectSocialSecurityTips,
		e.OriginalStateCode, e.CorrectStateCode,
		e.OriginalStateIDNumber, e.CorrectStateIDNumber,
		e.Amounts.OriginalStateWages, e.Amounts.CorrectStateWages,
		e.Amounts.OriginalStateIncomeTax, e.Amounts.CorrectStateIncomeTax,
		e.Amounts.OriginalLocalWages, e.Amounts.CorrectLocalWages,
		e.Amounts.OriginalLocalIncomeTax, e.Amounts.CorrectLocalIncomeTax,
		e.OriginalLocalityName, e.CorrectLocalityName,
		now, now,
	)
	if err != nil {
		return err
	}
	id, _ := res.LastInsertId()
	e.ID = id
	return nil
}

// GetEmployee fetches a single employee record by ID.
// Add this method to internal/adapters/sqlite/repository.go
// alongside DeleteEmployee.
func (r *Repository) GetEmployee(ctx context.Context, id int64) (*domain.EmployeeRecord, error) {
	e := &domain.EmployeeRecord{}
	err := r.db.QueryRowContext(ctx, `
		SELECT id, submission_id, ssn, original_ssn,
		       first_name, middle_name, last_name, suffix,
		       addr1, addr2, city, state, zip, zip_ext,
		       orig_wages, corr_wages,
		       orig_ss_wages, corr_ss_wages,
		       orig_med_wages, corr_med_wages,
		       orig_fed_tax, corr_fed_tax,
		       orig_ss_tax, corr_ss_tax,
		       orig_med_tax, corr_med_tax,
		       orig_ss_tips, corr_ss_tips,
		       orig_state_code, corr_state_code,
		       orig_state_id, corr_state_id,
		       orig_state_wages, corr_state_wages,
		       orig_state_tax, corr_state_tax,
		       orig_local_wages, corr_local_wages,
		       orig_local_tax, corr_local_tax,
		       orig_locality_name, corr_locality_name,
		       created_at, updated_at
		FROM employees WHERE id=?`, id).Scan(
		&e.ID, &e.SubmissionID, &e.SSN, &e.OriginalSSN,
		&e.FirstName, &e.MiddleName, &e.LastName, &e.Suffix,
		&e.AddressLine1, &e.AddressLine2, &e.City, &e.State, &e.ZIP, &e.ZIPExtension,
		&e.Amounts.OriginalWagesTipsOther, &e.Amounts.CorrectWagesTipsOther,
		&e.Amounts.OriginalSocialSecurityWages, &e.Amounts.CorrectSocialSecurityWages,
		&e.Amounts.OriginalMedicareWages, &e.Amounts.CorrectMedicareWages,
		&e.Amounts.OriginalFederalIncomeTax, &e.Amounts.CorrectFederalIncomeTax,
		&e.Amounts.OriginalSocialSecurityTax, &e.Amounts.CorrectSocialSecurityTax,
		&e.Amounts.OriginalMedicareTax, &e.Amounts.CorrectMedicareTax,
		&e.Amounts.OriginalSocialSecurityTips, &e.Amounts.CorrectSocialSecurityTips,
		&e.OriginalStateCode, &e.CorrectStateCode,
		&e.OriginalStateIDNumber, &e.CorrectStateIDNumber,
		&e.Amounts.OriginalStateWages, &e.Amounts.CorrectStateWages,
		&e.Amounts.OriginalStateIncomeTax, &e.Amounts.CorrectStateIncomeTax,
		&e.Amounts.OriginalLocalWages, &e.Amounts.CorrectLocalWages,
		&e.Amounts.OriginalLocalIncomeTax, &e.Amounts.CorrectLocalIncomeTax,
		&e.OriginalLocalityName, &e.CorrectLocalityName,
		&e.CreatedAt, &e.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return e, nil
}

func (r *Repository) UpdateEmployee(ctx context.Context, e *domain.EmployeeRecord) error {
	e.UpdatedAt = time.Now()
	_, err := r.db.ExecContext(ctx, `
		UPDATE employees
		SET ssn=?, original_ssn=?,
		    first_name=?, middle_name=?, last_name=?, suffix=?,
		    addr1=?, addr2=?, city=?, state=?, zip=?, zip_ext=?,
		    orig_wages=?, corr_wages=?,
		    orig_ss_wages=?, corr_ss_wages=?,
		    orig_med_wages=?, corr_med_wages=?,
		    orig_fed_tax=?, corr_fed_tax=?,
		    orig_ss_tax=?, corr_ss_tax=?,
		    orig_med_tax=?, corr_med_tax=?,
		    orig_ss_tips=?, corr_ss_tips=?,
		    orig_state_code=?, corr_state_code=?,
		    orig_state_id=?, corr_state_id=?,
		    orig_state_wages=?, corr_state_wages=?,
		    orig_state_tax=?, corr_state_tax=?,
		    orig_local_wages=?, corr_local_wages=?,
		    orig_local_tax=?, corr_local_tax=?,
		    orig_locality_name=?, corr_locality_name=?,
		    updated_at=?
		WHERE id=?`,
		e.SSN, e.OriginalSSN,
		e.FirstName, e.MiddleName, e.LastName, e.Suffix,
		e.AddressLine1, e.AddressLine2, e.City, e.State, e.ZIP, e.ZIPExtension,
		e.Amounts.OriginalWagesTipsOther, e.Amounts.CorrectWagesTipsOther,
		e.Amounts.OriginalSocialSecurityWages, e.Amounts.CorrectSocialSecurityWages,
		e.Amounts.OriginalMedicareWages, e.Amounts.CorrectMedicareWages,
		e.Amounts.OriginalFederalIncomeTax, e.Amounts.CorrectFederalIncomeTax,
		e.Amounts.OriginalSocialSecurityTax, e.Amounts.CorrectSocialSecurityTax,
		e.Amounts.OriginalMedicareTax, e.Amounts.CorrectMedicareTax,
		e.Amounts.OriginalSocialSecurityTips, e.Amounts.CorrectSocialSecurityTips,
		e.OriginalStateCode, e.CorrectStateCode,
		e.OriginalStateIDNumber, e.CorrectStateIDNumber,
		e.Amounts.OriginalStateWages, e.Amounts.CorrectStateWages,
		e.Amounts.OriginalStateIncomeTax, e.Amounts.CorrectStateIncomeTax,
		e.Amounts.OriginalLocalWages, e.Amounts.CorrectLocalWages,
		e.Amounts.OriginalLocalIncomeTax, e.Amounts.CorrectLocalIncomeTax,
		e.OriginalLocalityName, e.CorrectLocalityName,
		e.UpdatedAt, e.ID,
	)
	return err
}

func (r *Repository) DeleteEmployee(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM employees WHERE id=?`, id)
	return err
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
