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
	OriginalEIN         string // EIN correction only — leave blank otherwise
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

// MonetaryAmounts holds all monetary correction fields for an employee.
// Each field is stored in cents (int64) to avoid floating-point errors.
// "Original" = previously reported, "Correct" = corrected amount.
// Both must be provided when making a correction; leave zero if no correction.
type MonetaryAmounts struct {
	// Boxes 1–7 (RCW record, positions 244-397)
	OriginalWagesTipsOther      int64
	CorrectWagesTipsOther       int64
	OriginalFederalIncomeTax    int64
	CorrectFederalIncomeTax     int64
	OriginalSocialSecurityWages int64
	CorrectSocialSecurityWages  int64
	OriginalSocialSecurityTax   int64
	CorrectSocialSecurityTax    int64
	OriginalMedicareWages       int64
	CorrectMedicareWages        int64
	OriginalMedicareTax         int64
	CorrectMedicareTax          int64
	OriginalSocialSecurityTips  int64
	CorrectSocialSecurityTips   int64

	// Box 8 — Allocated Tips (RCO record, positions 13-34)
	OriginalAllocatedTips int64
	CorrectAllocatedTips  int64

	// Box 10 — Dependent Care Benefits (RCW, positions 420-441)
	OriginalDependentCare int64
	CorrectDependentCare  int64

	// Box 11 — Nonqualified Plans (RCW, two separate positions)
	//   Section 457 portion (positions 596-617)
	OriginalNonqualPlan457 int64
	CorrectNonqualPlan457  int64
	//   Non-Section 457 portion (positions 640-661)
	OriginalNonqualNotSection457 int64
	CorrectNonqualNotSection457  int64

	// Box 12 codes in RCW ─────────────────────────────────────────────
	// Code D — Elective deferrals to 401(k) (positions 442-463)
	OriginalCode401k int64
	CorrectCode401k  int64
	// Code E — Elective deferrals to 403(b) (positions 464-485)
	OriginalCode403b int64
	CorrectCode403b  int64
	// Code G — Elective deferrals to governmental 457(b) (positions 508-529)
	OriginalCode457bGovt int64
	CorrectCode457bGovt  int64
	// Code W — Employer HSA contributions (positions 618-639)
	OriginalCodeW_HSA int64
	CorrectCodeW_HSA  int64
	// Code AA — Designated Roth 401(k) (positions 772-793)
	OriginalCodeAA_Roth401k int64
	CorrectCodeAA_Roth401k  int64
	// Code BB — Designated Roth 403(b) (positions 794-815)
	OriginalCodeBB_Roth403b int64
	CorrectCodeBB_Roth403b  int64
	// Code DD — Employer-sponsored health coverage cost (positions 816-837)
	OriginalCodeDD_EmpHealth int64
	CorrectCodeDD_EmpHealth  int64

	// Box 16 — State wages, tips, etc. (RCS record)
	OriginalStateWages int64
	CorrectStateWages  int64
	// Box 17 — State income tax (RCS record)
	OriginalStateIncomeTax int64
	CorrectStateIncomeTax  int64
	// Box 18 — Local wages, tips, etc.
	OriginalLocalWages int64
	CorrectLocalWages  int64
	// Box 19 — Local income tax
	OriginalLocalIncomeTax int64
	CorrectLocalIncomeTax  int64
}

// Box13Flags holds the Box 13 checkbox corrections.
// The "Orig" field is the previously reported value; "Correct" is the correction.
// Use blank/nil when not correcting a particular checkbox.
type Box13Flags struct {
	OrigStatutoryEmployee    *bool
	CorrectStatutoryEmployee *bool
	OrigRetirementPlan       *bool
	CorrectRetirementPlan    *bool
	OrigThirdPartySickPay    *bool
	CorrectThirdPartySickPay *bool
}

type EmployeeRecord struct {
	ID           int64
	SubmissionID int64
	// Correct SSN (what it should be). Required.
	SSN string
	// OriginalSSN is only populated when correcting a previously wrong SSN.
	OriginalSSN string

	// Name — current/correct values (required)
	FirstName  string
	MiddleName string
	LastName   string
	Suffix     string

	// Name corrections — only populated when correcting previously wrong name fields.
	// When populated, the original name fields are written to RCW OrigFirstName etc.
	// and the current (FirstName/LastName etc.) are written to CorrectFirstName etc.
	OriginalFirstName  string
	OriginalMiddleName string
	OriginalLastName   string
	OriginalSuffix     string

	AddressLine1 string
	AddressLine2 string
	City         string
	State        string
	ZIP          string
	ZIPExtension string

	Amounts MonetaryAmounts

	// Box 13 corrections (orig/correct pairs for each checkbox)
	Box13 Box13Flags

	// Box 15 — State / Employer's state ID number
	OriginalStateCode     string
	CorrectStateCode      string
	OriginalStateIDNumber string
	CorrectStateIDNumber  string
	// Box 20 — Locality name
	OriginalLocalityName string
	CorrectLocalityName  string

	CreatedAt time.Time
	UpdatedAt time.Time
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
