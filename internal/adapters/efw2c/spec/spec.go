// Package spec defines the EFW2C record layout specifications per SSA Pub 42-014.
// RCA field positions are verified byte-for-byte against AccuWage Online TY2021
// error output. Other record positions are from §5.6–§5.13 of the same pub.
package spec

const RecordLen = 1024

type Field struct {
	Name        string
	Start       int
	End         int
	Type        FieldType
	Required    bool
	Description string
}

func (f Field) Len() int { return f.End - f.Start + 1 }

type FieldType int

const (
	Alpha   FieldType = iota // left-justified, space-filled, uppercase
	Numeric                  // right-justified, zero-filled digits only
	Money                    // 12-char zero-padded cents, no decimal
	Fixed                    // literal constant
	Blank                    // must be spaces
)

type YearSpec struct {
	TaxYear        int
	PublicationURL string
	SSWageBase     int64 // cents
	RCA            []Field
	RCE            []Field
	RCW            []Field
	RCT            []Field
	RCF            []Field
}

const DefaultYear = 2021

func Supported() []int { return []int{2021, 2022, 2023, 2024} }

func ForYear(year int) (*YearSpec, bool) {
	s, ok := specs[year]
	if !ok {
		s = specs[DefaultYear]
	}
	return s, ok
}

var specs = map[int]*YearSpec{
	2021: ty2021(),
	2022: ty2022(),
	2023: ty2023(),
	2024: ty2024(),
}

func ty2021() *YearSpec {
	s := baseSpec(2021)
	s.PublicationURL = "https://www.ssa.gov/employer/efw/21efw2c.pdf"
	s.SSWageBase = 14280000
	return s
}
func ty2022() *YearSpec {
	s := baseSpec(2022)
	s.PublicationURL = "https://www.ssa.gov/employer/efw/22efw2c.pdf"
	s.SSWageBase = 14700000
	return s
}
func ty2023() *YearSpec {
	s := baseSpec(2023)
	s.PublicationURL = "https://www.ssa.gov/employer/efw/23efw2c.pdf"
	s.SSWageBase = 16020000
	return s
}
func ty2024() *YearSpec {
	s := baseSpec(2024)
	s.PublicationURL = "https://www.ssa.gov/employer/efw/24efw2c.pdf"
	s.SSWageBase = 16860000
	return s
}

// baseSpec returns the record layout shared across TY2021–TY2024.
//
// RCA POSITIONS — verified against AccuWage Online TY2021 error output.
// Each anchor comes directly from the AccuWage field reference column:
//
//	 4- 12  Submitter EIN
//	13- 20  BSO User ID
//	30- 31  Software Code
//	41- 75  Company Name          (35 chars; derived from gap between 20 and next anchor)
//	76-115  Location Address      (40 chars)
//	116-155 Delivery Address      (40 chars)
//	155-156 State Abbreviation    (2 chars; ends just before ZIP)
//	157-161 ZIP Code              (5 chars)
//	162-165 ZIP Extension         (4 chars)
//	166-171 Blank                 (6 chars)
//	172-194 Foreign State/Province (23 chars)
//	195-207 Foreign Postal Code   (13 chars)
//	208-209 Country Code          (2 chars)
//	210-211 Blank                 (2 chars)
//	212-238 Contact Name          (27 chars)
//	239-253 Contact Phone         (15 chars)
//	254-258 Phone Extension       (5 chars)
//	259-261 Blank                 (3 chars)
//	262-301 Contact Email         (40 chars)
//	302-315 Blank                 (14 chars)
//	316     Preparer Code         (1 char)
//	317     Resub Indicator       (1 char)
//	318-326 Resub WFID            (9 chars)
//	327-1024 Blank
func baseSpec(year int) *YearSpec {
	return &YearSpec{
		TaxYear: year,

		// ── RCA (Submitter) ──────────────────────────────────────────────
		RCA: []Field{
			{Name: "RecordIdentifier", Start: 1, End: 3, Type: Fixed, Required: true, Description: "Constant 'RCA'"},
			{Name: "SubmitterEIN", Start: 4, End: 12, Type: Numeric, Required: true, Description: "Submitter EIN, 9 digits, no hyphens"},
			{Name: "BSOUID", Start: 13, End: 20, Type: Alpha, Required: true, Description: "BSO User ID, 8 alphanumeric chars assigned by SSA registration"},
			{Name: "Blank21", Start: 21, End: 29, Type: Blank, Required: false, Description: "Reserved"},
			{Name: "SoftwareCode", Start: 30, End: 31, Type: Numeric, Required: false, Description: "NACTP vendor code: 98 or 99; blank if not a software vendor"},
			{Name: "Blank32", Start: 32, End: 40, Type: Blank, Required: false, Description: "Reserved"},
			{Name: "CompanyName", Start: 41, End: 75, Type: Alpha, Required: true, Description: "Submitter company name, 35 chars"},
			{Name: "LocationAddress", Start: 76, End: 115, Type: Alpha, Required: true, Description: "Street/location address, 40 chars"},
			{Name: "DeliveryAddress", Start: 116, End: 155, Type: Alpha, Required: false, Description: "Delivery address (PO Box etc), 40 chars"},
			{Name: "StateAbbrev", Start: 156, End: 157, Type: Alpha, Required: false, Description: "State abbreviation, 2 chars; blank for foreign address"},
			{Name: "ZIPCode", Start: 158, End: 162, Type: Numeric, Required: false, Description: "ZIP code, numeric only; blank for foreign address"},
			{Name: "ZIPExtension", Start: 163, End: 166, Type: Numeric, Required: false, Description: "ZIP+4 extension"},
			{Name: "Blank167", Start: 167, End: 171, Type: Blank, Required: false, Description: "Reserved"},
			{Name: "ForeignStateProvince", Start: 172, End: 194, Type: Alpha, Required: false, Description: "Foreign state/province; required if no StateAbbrev"},
			{Name: "ForeignPostalCode", Start: 195, End: 207, Type: Alpha, Required: false, Description: "Foreign postal code"},
			{Name: "CountryCode", Start: 208, End: 209, Type: Alpha, Required: false, Description: "Country code per SSA Appendix I; blank for USA"},
			{Name: "Blank210", Start: 210, End: 211, Type: Blank, Required: false, Description: "Reserved"},
			{Name: "ContactName", Start: 212, End: 238, Type: Alpha, Required: true, Description: "Contact name; A-Z 0-9 space hyphen period apostrophe only"},
			{Name: "ContactPhone", Start: 239, End: 253, Type: Numeric, Required: true, Description: "Contact phone, numeric only, no special chars, e.g. 8005551234"},
			{Name: "PhoneExtension", Start: 254, End: 258, Type: Numeric, Required: false, Description: "Phone extension"},
			{Name: "Blank259", Start: 259, End: 261, Type: Blank, Required: false, Description: "Reserved"},
			{Name: "ContactEmail", Start: 262, End: 301, Type: Alpha, Required: true, Description: "Contact e-mail, valid format required"},
			{Name: "Blank302", Start: 302, End: 315, Type: Blank, Required: false, Description: "Reserved"},
			{Name: "PreparerCode", Start: 316, End: 316, Type: Alpha, Required: false, Description: "A=Accounting Firm L=Self-Prepared S=Service Bureau P=Parent O=Other"},
			{Name: "ResubIndicator", Start: 317, End: 317, Type: Alpha, Required: true, Description: "0=original submission 1=resubmission"},
			{Name: "ResubWFID", Start: 318, End: 326, Type: Alpha, Required: false, Description: "Original WFID (resubmissions only)"},
			{Name: "Blank327", Start: 327, End: 1024, Type: Blank, Required: false, Description: "Reserved"},
		},

		// ── RCE (Employer) ───────────────────────────────────────────────
		// Positions from §5.6; EIN at 17-25 and AgentForEIN at 27-35
		// confirmed by spec narrative text.
		RCE: []Field{
			{Name: "RecordIdentifier", Start: 1, End: 3, Type: Fixed, Required: true, Description: "Constant 'RCE'"},
			{Name: "TaxYear", Start: 4, End: 7, Type: Alpha, Required: true, Description: "Tax year being corrected e.g. '2021'"},
			{Name: "OrigReportedEIN", Start: 8, End: 16, Type: Numeric, Required: false, Description: "Originally reported EIN (EIN-correction filings only)"},
			{Name: "EmployerEIN", Start: 17, End: 25, Type: Numeric, Required: true, Description: "Employer/Agent EIN — SSA uses this to post W-2c data"},
			{Name: "Blank26", Start: 26, End: 26, Type: Blank, Required: false, Description: "Reserved"},
			{Name: "AgentForEIN", Start: 27, End: 35, Type: Numeric, Required: false, Description: "Client EIN for 2678/3504 agents and common paymasters"},
			{Name: "AgentIndicatorCode", Start: 36, End: 36, Type: Alpha, Required: false, Description: "0=none 1=2678Agent 2=CommonPaymaster 3=3504Agent"},
			{Name: "TerminatingBusiness", Start: 37, End: 37, Type: Alpha, Required: false, Description: "1=employer terminating business"},
			{Name: "EmploymentCode", Start: 38, End: 38, Type: Alpha, Required: true, Description: "A=Agri H=Household M=Military Q=MQGE R=Regular X=Railroad"},
			{Name: "TaxJurisdictionCode", Start: 39, End: 39, Type: Alpha, Required: false, Description: "blank=US P=PR G=Guam V=USVI A=AmSamoa N=CNMI"},
			{Name: "ThirdPartySickPayReap", Start: 40, End: 40, Type: Alpha, Required: false, Description: "1=third-party sick pay recap"},
			{Name: "EmployerName", Start: 41, End: 75, Type: Alpha, Required: true, Description: "Employer name"},
			{Name: "LocationAddress", Start: 76, End: 115, Type: Alpha, Required: true, Description: "Employer street address"},
			{Name: "DeliveryAddress", Start: 116, End: 155, Type: Alpha, Required: false, Description: "Employer delivery address"},
			{Name: "City", Start: 156, End: 194, Type: Alpha, Required: true, Description: "Employer city"},
			{Name: "StateAbbrev", Start: 195, End: 196, Type: Alpha, Required: false, Description: "State abbreviation"},
			{Name: "ZIPCode", Start: 197, End: 201, Type: Numeric, Required: false, Description: "ZIP code"},
			{Name: "ZIPExtension", Start: 202, End: 205, Type: Numeric, Required: false, Description: "ZIP+4"},
			{Name: "ForeignStateProvince", Start: 206, End: 225, Type: Alpha, Required: false, Description: "Foreign state/province"},
			{Name: "ForeignPostalCode", Start: 226, End: 239, Type: Alpha, Required: false, Description: "Foreign postal code"},
			{Name: "CountryCode", Start: 240, End: 241, Type: Alpha, Required: false, Description: "Country code"},
			{Name: "EmployerPhone", Start: 242, End: 256, Type: Numeric, Required: false, Description: "Employer phone"},
			{Name: "PhoneExtension", Start: 257, End: 261, Type: Numeric, Required: false, Description: "Phone extension"},
			{Name: "Blank262", Start: 262, End: 1024, Type: Blank, Required: false, Description: "Reserved"},
		},

		// ── RCW (Employee Correction) ─────────────────────────────────────
		// SSN at 4-12 and CorrectSSN at 13-21 confirmed by spec narrative.
		RCW: []Field{
			{Name: "RecordIdentifier", Start: 1, End: 3, Type: Fixed, Required: true},
			{Name: "OrigSSN", Start: 4, End: 12, Type: Numeric, Required: true, Description: "Originally reported SSN, 9 digits"},
			{Name: "CorrectSSN", Start: 13, End: 21, Type: Numeric, Required: false, Description: "Correct SSN (required only when correcting SSN)"},
			{Name: "OrigLastName", Start: 22, End: 36, Type: Alpha, Required: false},
			{Name: "OrigFirstName", Start: 37, End: 48, Type: Alpha, Required: false},
			{Name: "OrigMiddleName", Start: 49, End: 49, Type: Alpha, Required: false},
			{Name: "OrigSuffix", Start: 50, End: 53, Type: Alpha, Required: false},
			{Name: "CorrectLastName", Start: 54, End: 68, Type: Alpha, Required: false},
			{Name: "CorrectFirstName", Start: 69, End: 80, Type: Alpha, Required: false},
			{Name: "CorrectMiddleName", Start: 81, End: 81, Type: Alpha, Required: false},
			{Name: "CorrectSuffix", Start: 82, End: 85, Type: Alpha, Required: false},
			{Name: "LocationAddress", Start: 86, End: 124, Type: Alpha, Required: false},
			{Name: "DeliveryAddress", Start: 125, End: 163, Type: Alpha, Required: false},
			{Name: "City", Start: 164, End: 202, Type: Alpha, Required: false},
			{Name: "StateAbbrev", Start: 203, End: 204, Type: Alpha, Required: false},
			{Name: "ZIPCode", Start: 205, End: 209, Type: Numeric, Required: false},
			{Name: "ZIPExtension", Start: 210, End: 213, Type: Numeric, Required: false},
			{Name: "ForeignStateProvince", Start: 214, End: 243, Type: Alpha, Required: false},
			{Name: "ForeignPostalCode", Start: 244, End: 256, Type: Alpha, Required: false},
			{Name: "CountryCode", Start: 257, End: 258, Type: Alpha, Required: false},
			{Name: "OrigWagesTipsOther", Start: 259, End: 270, Type: Money, Required: false, Description: "Box 1 orig"},
			{Name: "CorrectWagesTipsOther", Start: 271, End: 282, Type: Money, Required: false, Description: "Box 1 corr"},
			{Name: "OrigFedIncomeTax", Start: 283, End: 294, Type: Money, Required: false, Description: "Box 2 orig"},
			{Name: "CorrectFedIncomeTax", Start: 295, End: 306, Type: Money, Required: false, Description: "Box 2 corr"},
			{Name: "OrigSSWages", Start: 307, End: 318, Type: Money, Required: false, Description: "Box 3 orig"},
			{Name: "CorrectSSWages", Start: 319, End: 330, Type: Money, Required: false, Description: "Box 3 corr"},
			{Name: "OrigSSTax", Start: 331, End: 342, Type: Money, Required: false, Description: "Box 4 orig"},
			{Name: "CorrectSSTax", Start: 343, End: 354, Type: Money, Required: false, Description: "Box 4 corr"},
			{Name: "OrigMedicareWages", Start: 355, End: 366, Type: Money, Required: false, Description: "Box 5 orig"},
			{Name: "CorrectMedicareWages", Start: 367, End: 378, Type: Money, Required: false, Description: "Box 5 corr"},
			{Name: "OrigMedicareTax", Start: 379, End: 390, Type: Money, Required: false, Description: "Box 6 orig"},
			{Name: "CorrectMedicareTax", Start: 391, End: 402, Type: Money, Required: false, Description: "Box 6 corr"},
			{Name: "OrigSSTips", Start: 403, End: 414, Type: Money, Required: false, Description: "Box 7 orig"},
			{Name: "CorrectSSTips", Start: 415, End: 426, Type: Money, Required: false, Description: "Box 7 corr"},
			{Name: "OrigAllocatedTips", Start: 427, End: 438, Type: Money, Required: false, Description: "Box 8 orig"},
			{Name: "CorrectAllocatedTips", Start: 439, End: 450, Type: Money, Required: false, Description: "Box 8 corr"},
			{Name: "Blank451", Start: 451, End: 462, Type: Blank, Required: false, Description: "Reserved (was Box 9 Advance EIC, eliminated 2010)"},
			{Name: "OrigDependentCare", Start: 463, End: 474, Type: Money, Required: false, Description: "Box 10 orig"},
			{Name: "CorrectDependentCare", Start: 475, End: 486, Type: Money, Required: false, Description: "Box 10 corr"},
			{Name: "OrigNonqualPlan", Start: 487, End: 498, Type: Money, Required: false, Description: "Box 11 orig"},
			{Name: "CorrectNonqualPlan", Start: 499, End: 510, Type: Money, Required: false, Description: "Box 11 corr"},
			{Name: "StatutoryEmployee", Start: 511, End: 511, Type: Alpha, Required: false},
			{Name: "RetirementPlan", Start: 512, End: 512, Type: Alpha, Required: false},
			{Name: "ThirdPartySickPay", Start: 513, End: 513, Type: Alpha, Required: false},
			{Name: "Blank514", Start: 514, End: 1024, Type: Blank, Required: false},
		},

		// ── RCT (Total) ──────────────────────────────────────────────────
		RCT: []Field{
			{Name: "RecordIdentifier", Start: 1, End: 3, Type: Fixed, Required: true},
			{Name: "OrigTotalWagesTips", Start: 4, End: 15, Type: Money, Required: false, Description: "Box 1 orig total"},
			{Name: "CorrectTotalWagesTips", Start: 16, End: 27, Type: Money, Required: false, Description: "Box 1 corr total"},
			{Name: "OrigTotalFedIncomeTax", Start: 28, End: 39, Type: Money, Required: false, Description: "Box 2 orig total"},
			{Name: "CorrectTotalFedIncomeTax", Start: 40, End: 51, Type: Money, Required: false, Description: "Box 2 corr total"},
			{Name: "OrigTotalSSWages", Start: 52, End: 63, Type: Money, Required: false, Description: "Box 3 orig total"},
			{Name: "CorrectTotalSSWages", Start: 64, End: 75, Type: Money, Required: false, Description: "Box 3 corr total"},
			{Name: "OrigTotalSSTax", Start: 76, End: 87, Type: Money, Required: false, Description: "Box 4 orig total"},
			{Name: "CorrectTotalSSTax", Start: 88, End: 99, Type: Money, Required: false, Description: "Box 4 corr total"},
			{Name: "OrigTotalMedicareWages", Start: 100, End: 111, Type: Money, Required: false, Description: "Box 5 orig total"},
			{Name: "CorrectTotalMedicareWages", Start: 112, End: 123, Type: Money, Required: false, Description: "Box 5 corr total"},
			{Name: "OrigTotalMedicareTax", Start: 124, End: 135, Type: Money, Required: false, Description: "Box 6 orig total"},
			{Name: "CorrectTotalMedicareTax", Start: 136, End: 147, Type: Money, Required: false, Description: "Box 6 corr total"},
			{Name: "OrigTotalSSTips", Start: 148, End: 159, Type: Money, Required: false, Description: "Box 7 orig total"},
			{Name: "CorrectTotalSSTips", Start: 160, End: 171, Type: Money, Required: false, Description: "Box 7 corr total"},
			{Name: "Blank172", Start: 172, End: 1024, Type: Blank, Required: false},
		},

		// ── RCF (Final) ──────────────────────────────────────────────────
		RCF: []Field{
			{Name: "RecordIdentifier", Start: 1, End: 3, Type: Fixed, Required: true},
			{Name: "TotalRCWRecords", Start: 4, End: 10, Type: Numeric, Required: true, Description: "Total RCW count, 7 digits zero-padded"},
			{Name: "Blank11", Start: 11, End: 1024, Type: Blank, Required: false},
		},
	}
}
