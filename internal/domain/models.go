package domain

import "time"

const DefaultTaxYear = "2024"

// TaxYearInfo carries a supported tax year string and its SSA publication URL.
// Populated by the generator port so templates can build selects with spec links.
type TaxYearInfo struct {
	Year           string // e.g. "2024"
	PublicationURL string // e.g. "https://www.ssa.gov/employer/efw/24efw2c.pdf"
}

// SubmitterInfo holds the RCA (Submitter) record fields.
type SubmitterInfo struct {
	// BSOUID is the 8-char BSO User ID from SSA registration (required).
	// Obtain at: https://www.ssa.gov/employer/
	BSOUID string
	// ContactName is the human contact at the submitter.
	// Allowed chars: A-Z, 0-9, space, hyphen, period, apostrophe.
	ContactName string

	// ContactPhone is numeric only — no dashes, spaces, or parens.
	// Example: "8005551234"
	ContactPhone string

	// ContactEmail must be a valid email address.
	ContactEmail string

	// PreparerCode: A=Accounting Firm, L=Self-Prepared, S=Service Bureau,
	// P=Parent Company, O=Other. Defaults to "L".
	PreparerCode string

	// ResubIndicator: "0"=original (default), "1"=resubmission.
	ResubIndicator string

	// ResubWFID is the original Wage File ID (resubmissions only).
	ResubWFID string
}

type EmployerRecord struct {
	EIN                 string
	Name                string
	AddressLine1        string
	AddressLine2        string
	City                string
	State               string
	ZIP                 string
	ZIPExtension        string
	TaxYear             string // e.g. "2024" — written into RCE record
	AgentIndicator      string
	AgentEIN            string
	TerminatingBusiness bool
	EmploymentCode      string // A/H/M/Q/R/X/F — defaults to "R"
	KindOfEmployer      string // F/S/T/Y/N
	ContactName         string
	ContactPhone        string
	ContactEmail        string
}

type MonetaryAmounts struct {
	OriginalWagesTipsOther      int64
	CorrectWagesTipsOther       int64
	OriginalSocialSecurityWages int64
	CorrectSocialSecurityWages  int64
	OriginalMedicareWages       int64
	CorrectMedicareWages        int64
	OriginalFederalIncomeTax    int64
	CorrectFederalIncomeTax     int64
	OriginalSocialSecurityTax   int64
	CorrectSocialSecurityTax    int64
	OriginalMedicareTax         int64
	CorrectMedicareTax          int64
	OriginalSocialSecurityTips  int64
	CorrectSocialSecurityTips   int64
}

type EmployeeRecord struct {
	ID           int64
	SubmissionID int64
	SSN          string
	OriginalSSN  string
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

type Submission struct {
	ID          int64
	Submitter   SubmitterInfo
	Employer    EmployerRecord
	Employees   []EmployeeRecord
	CreatedAt   time.Time
	SubmittedAt *time.Time
	Notes       string
}
