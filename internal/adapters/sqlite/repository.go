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
			ein, orig_ein, employer_name, addr1, addr2, city, state, zip, zip_ext,
			agent_indicator, agent_ein, terminating, notes,
			bso_uid, contact_name, contact_phone, contact_email, preparer_code,
			employment_code, kind_of_employer,
			employer_contact_name, employer_contact_phone, employer_contact_email,
		    created_at, tax_year
	    ) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		s.Employer.EIN, s.Employer.OriginalEIN, s.Employer.Name,
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
		SELECT id, ein, orig_ein, employer_name, addr1, addr2, city, state, zip, zip_ext,
		       agent_indicator, agent_ein, terminating, notes,
		       bso_uid, contact_name, contact_phone, contact_email, preparer_code,
		       employment_code, kind_of_employer,
		       employer_contact_name, employer_contact_phone, employer_contact_email,
		       created_at, submitted_at, tax_year
		FROM submissions WHERE id=?`, id).Scan(
		&s.ID, &s.Employer.EIN, &s.Employer.OriginalEIN, &s.Employer.Name,
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
	if s.Employer.TaxYear == "" {
		s.Employer.TaxYear = domain.DefaultTaxYear
	}
	if submittedAt.Valid {
		s.SubmittedAt = &submittedAt.Time
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, submission_id, ssn, original_ssn,
		       first_name, middle_name, last_name, suffix,
		       orig_first_name, orig_middle_name, orig_last_name, orig_suffix,
		       addr1, addr2, city, state, zip, zip_ext,
		       orig_wages, corr_wages,
		       orig_ss_wages, corr_ss_wages,
		       orig_med_wages, corr_med_wages,
		       orig_fed_tax, corr_fed_tax,
		       orig_ss_tax, corr_ss_tax,
		       orig_med_tax, corr_med_tax,
		       orig_ss_tips, corr_ss_tips,
		       orig_alloc_tips, corr_alloc_tips,
		       orig_dep_care, corr_dep_care,
		       orig_nonqual_457, corr_nonqual_457,
		       orig_nonqual_not457, corr_nonqual_not457,
		       orig_code_d, corr_code_d,
		       orig_code_e, corr_code_e,
		       orig_code_g, corr_code_g,
		       orig_code_w, corr_code_w,
		       orig_code_aa, corr_code_aa,
		       orig_code_bb, corr_code_bb,
		       orig_code_dd, corr_code_dd,
		       orig_state_code, corr_state_code,
		       orig_state_id, corr_state_id,
		       orig_state_wages, corr_state_wages,
		       orig_state_tax, corr_state_tax,
		       orig_local_wages, corr_local_wages,
		       orig_local_tax, corr_local_tax,
		       orig_locality_name, corr_locality_name,
		       orig_statutory_emp, corr_statutory_emp,
		       orig_retirement_plan, corr_retirement_plan,
		       orig_third_party_sick, corr_third_party_sick,
		       created_at, updated_at
		FROM employees WHERE submission_id=? ORDER BY id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var e domain.EmployeeRecord
		var (
			origStatuory, corrStatutory       sql.NullInt64
			origRetirement, corrRetirement     sql.NullInt64
			origThirdParty, corrThirdParty     sql.NullInt64
		)
		if err := rows.Scan(
			&e.ID, &e.SubmissionID, &e.SSN, &e.OriginalSSN,
			&e.FirstName, &e.MiddleName, &e.LastName, &e.Suffix,
			&e.OriginalFirstName, &e.OriginalMiddleName, &e.OriginalLastName, &e.OriginalSuffix,
			&e.AddressLine1, &e.AddressLine2, &e.City, &e.State, &e.ZIP, &e.ZIPExtension,
			&e.Amounts.OriginalWagesTipsOther, &e.Amounts.CorrectWagesTipsOther,
			&e.Amounts.OriginalSocialSecurityWages, &e.Amounts.CorrectSocialSecurityWages,
			&e.Amounts.OriginalMedicareWages, &e.Amounts.CorrectMedicareWages,
			&e.Amounts.OriginalFederalIncomeTax, &e.Amounts.CorrectFederalIncomeTax,
			&e.Amounts.OriginalSocialSecurityTax, &e.Amounts.CorrectSocialSecurityTax,
			&e.Amounts.OriginalMedicareTax, &e.Amounts.CorrectMedicareTax,
			&e.Amounts.OriginalSocialSecurityTips, &e.Amounts.CorrectSocialSecurityTips,
			&e.Amounts.OriginalAllocatedTips, &e.Amounts.CorrectAllocatedTips,
			&e.Amounts.OriginalDependentCare, &e.Amounts.CorrectDependentCare,
			&e.Amounts.OriginalNonqualPlan457, &e.Amounts.CorrectNonqualPlan457,
			&e.Amounts.OriginalNonqualNotSection457, &e.Amounts.CorrectNonqualNotSection457,
			&e.Amounts.OriginalCode401k, &e.Amounts.CorrectCode401k,
			&e.Amounts.OriginalCode403b, &e.Amounts.CorrectCode403b,
			&e.Amounts.OriginalCode457bGovt, &e.Amounts.CorrectCode457bGovt,
			&e.Amounts.OriginalCodeW_HSA, &e.Amounts.CorrectCodeW_HSA,
			&e.Amounts.OriginalCodeAA_Roth401k, &e.Amounts.CorrectCodeAA_Roth401k,
			&e.Amounts.OriginalCodeBB_Roth403b, &e.Amounts.CorrectCodeBB_Roth403b,
			&e.Amounts.OriginalCodeDD_EmpHealth, &e.Amounts.CorrectCodeDD_EmpHealth,
			&e.OriginalStateCode, &e.CorrectStateCode,
			&e.OriginalStateIDNumber, &e.CorrectStateIDNumber,
			&e.Amounts.OriginalStateWages, &e.Amounts.CorrectStateWages,
			&e.Amounts.OriginalStateIncomeTax, &e.Amounts.CorrectStateIncomeTax,
			&e.Amounts.OriginalLocalWages, &e.Amounts.CorrectLocalWages,
			&e.Amounts.OriginalLocalIncomeTax, &e.Amounts.CorrectLocalIncomeTax,
			&e.OriginalLocalityName, &e.CorrectLocalityName,
			&origStatuory, &corrStatutory,
			&origRetirement, &corrRetirement,
			&origThirdParty, &corrThirdParty,
			&e.CreatedAt, &e.UpdatedAt,
		); err != nil {
			return nil, err
		}
		e.Box13 = nullIntToBox13(origStatuory, corrStatutory, origRetirement, corrRetirement,
			origThirdParty, corrThirdParty)
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
		SET ein=?, orig_ein=?, employer_name=?, addr1=?, addr2=?, city=?, state=?, zip=?, zip_ext=?,
		    agent_indicator=?, agent_ein=?, terminating=?, notes=?,
		    bso_uid=?, contact_name=?, contact_phone=?, contact_email=?, preparer_code=?,
		    employment_code=?, kind_of_employer=?,
		    employer_contact_name=?, employer_contact_phone=?, employer_contact_email=?,
		    tax_year=?
        WHERE id=?`,
		s.Employer.EIN, s.Employer.OriginalEIN, s.Employer.Name,
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
	b13 := box13ToNullInt(e.Box13)
	res, err := r.db.ExecContext(ctx, `
		INSERT INTO employees (
			submission_id, ssn, original_ssn,
			first_name, middle_name, last_name, suffix,
			orig_first_name, orig_middle_name, orig_last_name, orig_suffix,
			addr1, addr2, city, state, zip, zip_ext,
			orig_wages, corr_wages,
			orig_ss_wages, corr_ss_wages,
			orig_med_wages, corr_med_wages,
			orig_fed_tax, corr_fed_tax,
			orig_ss_tax, corr_ss_tax,
			orig_med_tax, corr_med_tax,
			orig_ss_tips, corr_ss_tips,
			orig_alloc_tips, corr_alloc_tips,
			orig_dep_care, corr_dep_care,
			orig_nonqual_457, corr_nonqual_457,
			orig_nonqual_not457, corr_nonqual_not457,
			orig_code_d, corr_code_d,
			orig_code_e, corr_code_e,
			orig_code_g, corr_code_g,
			orig_code_w, corr_code_w,
			orig_code_aa, corr_code_aa,
			orig_code_bb, corr_code_bb,
			orig_code_dd, corr_code_dd,
			orig_state_code, corr_state_code,
			orig_state_id, corr_state_id,
			orig_state_wages, corr_state_wages,
			orig_state_tax, corr_state_tax,
			orig_local_wages, corr_local_wages,
			orig_local_tax, corr_local_tax,
			orig_locality_name, corr_locality_name,
			orig_statutory_emp, corr_statutory_emp,
			orig_retirement_plan, corr_retirement_plan,
			orig_third_party_sick, corr_third_party_sick,
			created_at, updated_at
		) VALUES (
			?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?
		)`,
		submissionID, e.SSN, e.OriginalSSN,
		e.FirstName, e.MiddleName, e.LastName, e.Suffix,
		e.OriginalFirstName, e.OriginalMiddleName, e.OriginalLastName, e.OriginalSuffix,
		e.AddressLine1, e.AddressLine2, e.City, e.State, e.ZIP, e.ZIPExtension,
		e.Amounts.OriginalWagesTipsOther, e.Amounts.CorrectWagesTipsOther,
		e.Amounts.OriginalSocialSecurityWages, e.Amounts.CorrectSocialSecurityWages,
		e.Amounts.OriginalMedicareWages, e.Amounts.CorrectMedicareWages,
		e.Amounts.OriginalFederalIncomeTax, e.Amounts.CorrectFederalIncomeTax,
		e.Amounts.OriginalSocialSecurityTax, e.Amounts.CorrectSocialSecurityTax,
		e.Amounts.OriginalMedicareTax, e.Amounts.CorrectMedicareTax,
		e.Amounts.OriginalSocialSecurityTips, e.Amounts.CorrectSocialSecurityTips,
		e.Amounts.OriginalAllocatedTips, e.Amounts.CorrectAllocatedTips,
		e.Amounts.OriginalDependentCare, e.Amounts.CorrectDependentCare,
		e.Amounts.OriginalNonqualPlan457, e.Amounts.CorrectNonqualPlan457,
		e.Amounts.OriginalNonqualNotSection457, e.Amounts.CorrectNonqualNotSection457,
		e.Amounts.OriginalCode401k, e.Amounts.CorrectCode401k,
		e.Amounts.OriginalCode403b, e.Amounts.CorrectCode403b,
		e.Amounts.OriginalCode457bGovt, e.Amounts.CorrectCode457bGovt,
		e.Amounts.OriginalCodeW_HSA, e.Amounts.CorrectCodeW_HSA,
		e.Amounts.OriginalCodeAA_Roth401k, e.Amounts.CorrectCodeAA_Roth401k,
		e.Amounts.OriginalCodeBB_Roth403b, e.Amounts.CorrectCodeBB_Roth403b,
		e.Amounts.OriginalCodeDD_EmpHealth, e.Amounts.CorrectCodeDD_EmpHealth,
		e.OriginalStateCode, e.CorrectStateCode,
		e.OriginalStateIDNumber, e.CorrectStateIDNumber,
		e.Amounts.OriginalStateWages, e.Amounts.CorrectStateWages,
		e.Amounts.OriginalStateIncomeTax, e.Amounts.CorrectStateIncomeTax,
		e.Amounts.OriginalLocalWages, e.Amounts.CorrectLocalWages,
		e.Amounts.OriginalLocalIncomeTax, e.Amounts.CorrectLocalIncomeTax,
		e.OriginalLocalityName, e.CorrectLocalityName,
		b13.origStat, b13.corrStat,
		b13.origRet, b13.corrRet,
		b13.origThird, b13.corrThird,
		now, now,
	)
	if err != nil {
		return err
	}
	id, _ := res.LastInsertId()
	e.ID = id
	return nil
}

func (r *Repository) GetEmployee(ctx context.Context, id int64) (*domain.EmployeeRecord, error) {
	e := &domain.EmployeeRecord{}
	var (
		origStat, corrStat     sql.NullInt64
		origRet, corrRet       sql.NullInt64
		origThird, corrThird   sql.NullInt64
	)
	err := r.db.QueryRowContext(ctx, `
		SELECT id, submission_id, ssn, original_ssn,
		       first_name, middle_name, last_name, suffix,
		       orig_first_name, orig_middle_name, orig_last_name, orig_suffix,
		       addr1, addr2, city, state, zip, zip_ext,
		       orig_wages, corr_wages,
		       orig_ss_wages, corr_ss_wages,
		       orig_med_wages, corr_med_wages,
		       orig_fed_tax, corr_fed_tax,
		       orig_ss_tax, corr_ss_tax,
		       orig_med_tax, corr_med_tax,
		       orig_ss_tips, corr_ss_tips,
		       orig_alloc_tips, corr_alloc_tips,
		       orig_dep_care, corr_dep_care,
		       orig_nonqual_457, corr_nonqual_457,
		       orig_nonqual_not457, corr_nonqual_not457,
		       orig_code_d, corr_code_d,
		       orig_code_e, corr_code_e,
		       orig_code_g, corr_code_g,
		       orig_code_w, corr_code_w,
		       orig_code_aa, corr_code_aa,
		       orig_code_bb, corr_code_bb,
		       orig_code_dd, corr_code_dd,
		       orig_state_code, corr_state_code,
		       orig_state_id, corr_state_id,
		       orig_state_wages, corr_state_wages,
		       orig_state_tax, corr_state_tax,
		       orig_local_wages, corr_local_wages,
		       orig_local_tax, corr_local_tax,
		       orig_locality_name, corr_locality_name,
		       orig_statutory_emp, corr_statutory_emp,
		       orig_retirement_plan, corr_retirement_plan,
		       orig_third_party_sick, corr_third_party_sick,
		       created_at, updated_at
		FROM employees WHERE id=?`, id).Scan(
		&e.ID, &e.SubmissionID, &e.SSN, &e.OriginalSSN,
		&e.FirstName, &e.MiddleName, &e.LastName, &e.Suffix,
		&e.OriginalFirstName, &e.OriginalMiddleName, &e.OriginalLastName, &e.OriginalSuffix,
		&e.AddressLine1, &e.AddressLine2, &e.City, &e.State, &e.ZIP, &e.ZIPExtension,
		&e.Amounts.OriginalWagesTipsOther, &e.Amounts.CorrectWagesTipsOther,
		&e.Amounts.OriginalSocialSecurityWages, &e.Amounts.CorrectSocialSecurityWages,
		&e.Amounts.OriginalMedicareWages, &e.Amounts.CorrectMedicareWages,
		&e.Amounts.OriginalFederalIncomeTax, &e.Amounts.CorrectFederalIncomeTax,
		&e.Amounts.OriginalSocialSecurityTax, &e.Amounts.CorrectSocialSecurityTax,
		&e.Amounts.OriginalMedicareTax, &e.Amounts.CorrectMedicareTax,
		&e.Amounts.OriginalSocialSecurityTips, &e.Amounts.CorrectSocialSecurityTips,
		&e.Amounts.OriginalAllocatedTips, &e.Amounts.CorrectAllocatedTips,
		&e.Amounts.OriginalDependentCare, &e.Amounts.CorrectDependentCare,
		&e.Amounts.OriginalNonqualPlan457, &e.Amounts.CorrectNonqualPlan457,
		&e.Amounts.OriginalNonqualNotSection457, &e.Amounts.CorrectNonqualNotSection457,
		&e.Amounts.OriginalCode401k, &e.Amounts.CorrectCode401k,
		&e.Amounts.OriginalCode403b, &e.Amounts.CorrectCode403b,
		&e.Amounts.OriginalCode457bGovt, &e.Amounts.CorrectCode457bGovt,
		&e.Amounts.OriginalCodeW_HSA, &e.Amounts.CorrectCodeW_HSA,
		&e.Amounts.OriginalCodeAA_Roth401k, &e.Amounts.CorrectCodeAA_Roth401k,
		&e.Amounts.OriginalCodeBB_Roth403b, &e.Amounts.CorrectCodeBB_Roth403b,
		&e.Amounts.OriginalCodeDD_EmpHealth, &e.Amounts.CorrectCodeDD_EmpHealth,
		&e.OriginalStateCode, &e.CorrectStateCode,
		&e.OriginalStateIDNumber, &e.CorrectStateIDNumber,
		&e.Amounts.OriginalStateWages, &e.Amounts.CorrectStateWages,
		&e.Amounts.OriginalStateIncomeTax, &e.Amounts.CorrectStateIncomeTax,
		&e.Amounts.OriginalLocalWages, &e.Amounts.CorrectLocalWages,
		&e.Amounts.OriginalLocalIncomeTax, &e.Amounts.CorrectLocalIncomeTax,
		&e.OriginalLocalityName, &e.CorrectLocalityName,
		&origStat, &corrStat,
		&origRet, &corrRet,
		&origThird, &corrThird,
		&e.CreatedAt, &e.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	e.Box13 = nullIntToBox13(origStat, corrStat, origRet, corrRet, origThird, corrThird)
	return e, nil
}

func (r *Repository) UpdateEmployee(ctx context.Context, e *domain.EmployeeRecord) error {
	e.UpdatedAt = time.Now()
	b13 := box13ToNullInt(e.Box13)
	_, err := r.db.ExecContext(ctx, `
		UPDATE employees
		SET ssn=?, original_ssn=?,
		    first_name=?, middle_name=?, last_name=?, suffix=?,
		    orig_first_name=?, orig_middle_name=?, orig_last_name=?, orig_suffix=?,
		    addr1=?, addr2=?, city=?, state=?, zip=?, zip_ext=?,
		    orig_wages=?, corr_wages=?,
		    orig_ss_wages=?, corr_ss_wages=?,
		    orig_med_wages=?, corr_med_wages=?,
		    orig_fed_tax=?, corr_fed_tax=?,
		    orig_ss_tax=?, corr_ss_tax=?,
		    orig_med_tax=?, corr_med_tax=?,
		    orig_ss_tips=?, corr_ss_tips=?,
		    orig_alloc_tips=?, corr_alloc_tips=?,
		    orig_dep_care=?, corr_dep_care=?,
		    orig_nonqual_457=?, corr_nonqual_457=?,
		    orig_nonqual_not457=?, corr_nonqual_not457=?,
		    orig_code_d=?, corr_code_d=?,
		    orig_code_e=?, corr_code_e=?,
		    orig_code_g=?, corr_code_g=?,
		    orig_code_w=?, corr_code_w=?,
		    orig_code_aa=?, corr_code_aa=?,
		    orig_code_bb=?, corr_code_bb=?,
		    orig_code_dd=?, corr_code_dd=?,
		    orig_state_code=?, corr_state_code=?,
		    orig_state_id=?, corr_state_id=?,
		    orig_state_wages=?, corr_state_wages=?,
		    orig_state_tax=?, corr_state_tax=?,
		    orig_local_wages=?, corr_local_wages=?,
		    orig_local_tax=?, corr_local_tax=?,
		    orig_locality_name=?, corr_locality_name=?,
		    orig_statutory_emp=?, corr_statutory_emp=?,
		    orig_retirement_plan=?, corr_retirement_plan=?,
		    orig_third_party_sick=?, corr_third_party_sick=?,
		    updated_at=?
		WHERE id=?`,
		e.SSN, e.OriginalSSN,
		e.FirstName, e.MiddleName, e.LastName, e.Suffix,
		e.OriginalFirstName, e.OriginalMiddleName, e.OriginalLastName, e.OriginalSuffix,
		e.AddressLine1, e.AddressLine2, e.City, e.State, e.ZIP, e.ZIPExtension,
		e.Amounts.OriginalWagesTipsOther, e.Amounts.CorrectWagesTipsOther,
		e.Amounts.OriginalSocialSecurityWages, e.Amounts.CorrectSocialSecurityWages,
		e.Amounts.OriginalMedicareWages, e.Amounts.CorrectMedicareWages,
		e.Amounts.OriginalFederalIncomeTax, e.Amounts.CorrectFederalIncomeTax,
		e.Amounts.OriginalSocialSecurityTax, e.Amounts.CorrectSocialSecurityTax,
		e.Amounts.OriginalMedicareTax, e.Amounts.CorrectMedicareTax,
		e.Amounts.OriginalSocialSecurityTips, e.Amounts.CorrectSocialSecurityTips,
		e.Amounts.OriginalAllocatedTips, e.Amounts.CorrectAllocatedTips,
		e.Amounts.OriginalDependentCare, e.Amounts.CorrectDependentCare,
		e.Amounts.OriginalNonqualPlan457, e.Amounts.CorrectNonqualPlan457,
		e.Amounts.OriginalNonqualNotSection457, e.Amounts.CorrectNonqualNotSection457,
		e.Amounts.OriginalCode401k, e.Amounts.CorrectCode401k,
		e.Amounts.OriginalCode403b, e.Amounts.CorrectCode403b,
		e.Amounts.OriginalCode457bGovt, e.Amounts.CorrectCode457bGovt,
		e.Amounts.OriginalCodeW_HSA, e.Amounts.CorrectCodeW_HSA,
		e.Amounts.OriginalCodeAA_Roth401k, e.Amounts.CorrectCodeAA_Roth401k,
		e.Amounts.OriginalCodeBB_Roth403b, e.Amounts.CorrectCodeBB_Roth403b,
		e.Amounts.OriginalCodeDD_EmpHealth, e.Amounts.CorrectCodeDD_EmpHealth,
		e.OriginalStateCode, e.CorrectStateCode,
		e.OriginalStateIDNumber, e.CorrectStateIDNumber,
		e.Amounts.OriginalStateWages, e.Amounts.CorrectStateWages,
		e.Amounts.OriginalStateIncomeTax, e.Amounts.CorrectStateIncomeTax,
		e.Amounts.OriginalLocalWages, e.Amounts.CorrectLocalWages,
		e.Amounts.OriginalLocalIncomeTax, e.Amounts.CorrectLocalIncomeTax,
		e.OriginalLocalityName, e.CorrectLocalityName,
		b13.origStat, b13.corrStat,
		b13.origRet, b13.corrRet,
		b13.origThird, b13.corrThird,
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

type box13NullInts struct {
	origStat, corrStat   sql.NullInt64
	origRet, corrRet     sql.NullInt64
	origThird, corrThird sql.NullInt64
}

func box13ToNullInt(b domain.Box13Flags) box13NullInts {
	return box13NullInts{
		origStat:  boolPtrToNullInt(b.OrigStatutoryEmployee),
		corrStat:  boolPtrToNullInt(b.CorrectStatutoryEmployee),
		origRet:   boolPtrToNullInt(b.OrigRetirementPlan),
		corrRet:   boolPtrToNullInt(b.CorrectRetirementPlan),
		origThird: boolPtrToNullInt(b.OrigThirdPartySickPay),
		corrThird: boolPtrToNullInt(b.CorrectThirdPartySickPay),
	}
}

func boolPtrToNullInt(b *bool) sql.NullInt64 {
	if b == nil {
		return sql.NullInt64{}
	}
	v := int64(0)
	if *b {
		v = 1
	}
	return sql.NullInt64{Int64: v, Valid: true}
}

func nullIntToBox13(oS, cS, oR, cR, oT, cT sql.NullInt64) domain.Box13Flags {
	return domain.Box13Flags{
		OrigStatutoryEmployee:    nullIntToBoolPtr(oS),
		CorrectStatutoryEmployee: nullIntToBoolPtr(cS),
		OrigRetirementPlan:       nullIntToBoolPtr(oR),
		CorrectRetirementPlan:    nullIntToBoolPtr(cR),
		OrigThirdPartySickPay:    nullIntToBoolPtr(oT),
		CorrectThirdPartySickPay: nullIntToBoolPtr(cT),
	}
}

func nullIntToBoolPtr(n sql.NullInt64) *bool {
	if !n.Valid {
		return nil
	}
	v := n.Int64 != 0
	return &v
}
