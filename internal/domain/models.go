package domain

import "time"

// TaxYear2021 is the only supported tax year for this tool.
const TaxYear2021 = "2021"

// EmployerRecord holds the employer (RCE) fields for EFW2C.
type EmployerRecord struct {
	EIN                 string // 9 digits, no dashes
	Name                string
	AddressLine1        string
	AddressLine2        string
	City                string
	State               string
	ZIP                 string
	ZIPExtension        string
	TaxYear             string
	AgentIndicator      string // "0" = not agent
	AgentEIN            string
	TerminatingBusiness bool
}

// MonetaryAmounts holds both "originally reported" and "correct" values
// for wages/compensation and tax withholdings on the W2C (RCW record).
type MonetaryAmounts struct {
	// Wages & Compensation (Box 1, 3, 5)
	OriginalWagesTipsOther      int64 // in cents
	CorrectWagesTipsOther       int64
	OriginalSocialSecurityWages int64
	CorrectSocialSecurityWages  int64
	OriginalMedicareWages       int64
	CorrectMedicareWages        int64

	// Tax Withholdings (Box 2, 4, 6)
	OriginalFederalIncomeTax  int64
	CorrectFederalIncomeTax   int64
	OriginalSocialSecurityTax int64
	CorrectSocialSecurityTax  int64
	OriginalMedicareTax       int64
	CorrectMedicareTax        int64
}

// EmployeeRecord represents one W-2c correction (RCW record).
type EmployeeRecord struct {
	ID           int64
	SubmissionID int64
	SSN          string // 9 digits
	OriginalSSN  string // if SSN was corrected
	FirstName    string
	MiddleName   string
	LastName     string
	Suffix       string
	AddressLine1 string
	AddressLine2 string
	City         string
	State        string
	ZIP          string
	ZIPExtension string
	Amounts      MonetaryAmounts
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Submission groups an employer + a batch of employee corrections.
type Submission struct {
	ID          int64
	Employer    EmployerRecord
	Employees   []EmployeeRecord
	CreatedAt   time.Time
	SubmittedAt *time.Time
	Notes       string
}
