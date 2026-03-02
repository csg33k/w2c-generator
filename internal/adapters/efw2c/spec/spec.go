// Package spec defines the EFW2C record layout specifications per SSA Pub 42-014.
// Field positions are from the TY2024 specification (Pub 42-014).
// The RCW/RCT/RCF layouts are unchanged from TY2021–TY2023;
// only RCO/RCU had new fields added for TY2024 (Box 12 Code II).
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
	Money11                  // 11-char zero-padded cents, no decimal (RCW/RCO fields)
	Money15                  // 15-char zero-padded cents, no decimal (RCT total fields)
	Fixed                    // literal constant
	Blank                    // must be spaces
	// Money kept as alias for Money11 for backward compat
	Money = Money11
)

type YearSpec struct {
	TaxYear        int
	PublicationURL string
	SSWageBase     int64 // cents
	RCA            []Field
	RCE            []Field
	RCW            []Field
	RCO            []Field // Employee Optional — Box 8, selected Box 12 codes
	RCS            []Field // State Record — optional, SSA does not process
	RCT            []Field
	RCF            []Field
}

const DefaultYear = 2024

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
	// TY2024 adds Box 12 Code II (Medicaid Waiver) to RCO at positions 277-298.
	// Replace the trailing Blank277 (277-1024) with the Code II pair + new trailing blank.
	rco := s.RCO
	rco = rco[:len(rco)-1] // drop Blank277 (277-1024)
	s.RCO = append(rco, []Field{
		{Name: "OrigMedicaidWaiver", Start: 277, End: 287, Type: Money11, Description: "Box 12 Code II orig — Exclusion of Medicaid Waiver Payments (TY2024+)"},
		{Name: "CorrectMedicaidWaiver", Start: 288, End: 298, Type: Money11, Description: "Box 12 Code II corr"},
		{Name: "Blank299", Start: 299, End: 1024, Type: Blank},
	}...)
	return s
}

// baseSpec returns the record layout shared across TY2021–TY2023.
// TY2024 extends baseSpec with Code II in RCO (see ty2024()).
//
// All positions are per SSA Pub 42-014 TY2024 (RCW/RCT/RCF unchanged since TY2021).
//
// RCA positions verified against AccuWage Online TY2021 error output.
// RCW/RCT positions verified against SSA Pub 42-014 TY2024 §5.7/§5.10.
func baseSpec(year int) *YearSpec {
	return &YearSpec{
		TaxYear: year,

		// ── RCA (Submitter) ──────────────────────────────────────────────
		// Positions verified against SSA Pub 42-014 TY2024 §5.5.
		RCA: []Field{
			{Name: "RecordIdentifier", Start: 1, End: 3, Type: Fixed, Required: true, Description: "Constant 'RCA'"},
			{Name: "SubmitterEIN", Start: 4, End: 12, Type: Numeric, Required: true, Description: "Submitter EIN, 9 digits, no hyphens"},
			{Name: "BSOUID", Start: 13, End: 20, Type: Alpha, Required: true, Description: "BSO User ID, 8 alphanumeric chars assigned by SSA registration"},
			{Name: "SoftwareVendorCode", Start: 21, End: 24, Type: Numeric, Required: false, Description: "NACTP 4-digit vendor code; required only if SoftwareCode=99"},
			{Name: "Blank25", Start: 25, End: 29, Type: Blank, Required: false, Description: "Reserved for SSA use"},
			{Name: "SoftwareCode", Start: 30, End: 31, Type: Numeric, Required: false, Description: "98=In-House Program 99=Off-the-Shelf Software"},
			{Name: "CompanyName", Start: 32, End: 88, Type: Alpha, Required: true, Description: "Submitter company name, 57 chars"},
			{Name: "LocationAddress", Start: 89, End: 110, Type: Alpha, Required: true, Description: "Location address (Attn, Suite, etc.), 22 chars"},
			{Name: "DeliveryAddress", Start: 111, End: 132, Type: Alpha, Required: true, Description: "Delivery address (Street or PO Box), 22 chars"},
			{Name: "City", Start: 133, End: 154, Type: Alpha, Required: true, Description: "City, 22 chars"},
			{Name: "StateAbbrev", Start: 155, End: 156, Type: Alpha, Required: false, Description: "State abbreviation; blank for foreign address"},
			{Name: "ZIPCode", Start: 157, End: 161, Type: Numeric, Required: false, Description: "ZIP code; blank for foreign address"},
			{Name: "ZIPExtension", Start: 162, End: 165, Type: Numeric, Required: false, Description: "ZIP+4 extension"},
			{Name: "Blank166", Start: 166, End: 171, Type: Blank, Required: false, Description: "Reserved for SSA use"},
			{Name: "ForeignStateProvince", Start: 172, End: 194, Type: Alpha, Required: false, Description: "Foreign state/province; required if no StateAbbrev"},
			{Name: "ForeignPostalCode", Start: 195, End: 209, Type: Alpha, Required: false, Description: "Foreign postal code, 15 chars"},
			{Name: "CountryCode", Start: 210, End: 211, Type: Alpha, Required: false, Description: "Country code per SSA Appendix I; blank for USA"},
			{Name: "ContactName", Start: 212, End: 238, Type: Alpha, Required: true, Description: "Contact name; A-Z 0-9 space hyphen period apostrophe only"},
			{Name: "ContactPhone", Start: 239, End: 253, Type: Numeric, Required: true, Description: "Contact phone, numeric only, e.g. 8005551234"},
			{Name: "PhoneExtension", Start: 254, End: 258, Type: Numeric, Required: false, Description: "Phone extension"},
			{Name: "Blank259", Start: 259, End: 261, Type: Blank, Required: false, Description: "Reserved"},
			{Name: "ContactEmail", Start: 262, End: 301, Type: Alpha, Required: true, Description: "Contact e-mail, valid format required"},
			{Name: "Blank302", Start: 302, End: 304, Type: Blank, Required: false, Description: "Reserved"},
			{Name: "ContactFax", Start: 305, End: 314, Type: Numeric, Required: false, Description: "Contact fax number"},
			{Name: "Blank315", Start: 315, End: 315, Type: Blank, Required: false, Description: "Reserved"},
			{Name: "PreparerCode", Start: 316, End: 316, Type: Alpha, Required: false, Description: "A=Accounting Firm L=Self-Prepared S=Service Bureau P=Parent O=Other"},
			{Name: "ResubIndicator", Start: 317, End: 317, Type: Alpha, Required: true, Description: "0=original submission 1=resubmission"},
			{Name: "ResubWFID", Start: 318, End: 323, Type: Alpha, Required: false, Description: "Original WFID (resubmissions only), 6 chars"},
			{Name: "Blank324", Start: 324, End: 1024, Type: Blank, Required: false, Description: "Reserved"},
		},

		// ── RCE (Employer) ───────────────────────────────────────────────
		// Positions verified against SSA Pub 42-014 TY2024 §5.6.
		RCE: []Field{
			{Name: "RecordIdentifier", Start: 1, End: 3, Type: Fixed, Required: true, Description: "Constant 'RCE'"},
			{Name: "TaxYear", Start: 4, End: 7, Type: Alpha, Required: true, Description: "Tax year being corrected e.g. '2024'"},
			{Name: "OrigReportedEIN", Start: 8, End: 16, Type: Numeric, Required: false, Description: "Originally reported EIN (EIN-correction filings only); else blank"},
			{Name: "EmployerEIN", Start: 17, End: 25, Type: Numeric, Required: true, Description: "Employer/Agent EIN — SSA uses this to post W-2c data"},
			{Name: "AgentIndicatorCode", Start: 26, End: 26, Type: Alpha, Required: false, Description: "blank=none 1=2678Agent 2=CommonPaymaster 3=3504Agent"},
			{Name: "AgentForEIN", Start: 27, End: 35, Type: Numeric, Required: false, Description: "Client EIN for 2678/3504 agents and common paymasters"},
			{Name: "OrigEstablishmentNum", Start: 36, End: 39, Type: Alpha, Required: false, Description: "Employer's originally reported establishment number"},
			{Name: "CorrectEstablishmentNum", Start: 40, End: 43, Type: Alpha, Required: false, Description: "Employer's correct establishment number"},
			{Name: "EmployerName", Start: 44, End: 100, Type: Alpha, Required: true, Description: "Employer name, 57 chars"},
			{Name: "LocationAddress", Start: 101, End: 122, Type: Alpha, Required: true, Description: "Employer location address (Attn, Suite, etc.), 22 chars"},
			{Name: "DeliveryAddress", Start: 123, End: 144, Type: Alpha, Required: false, Description: "Employer delivery address (Street or PO Box), 22 chars"},
			{Name: "City", Start: 145, End: 166, Type: Alpha, Required: true, Description: "Employer city, 22 chars"},
			{Name: "StateAbbrev", Start: 167, End: 168, Type: Alpha, Required: false, Description: "State abbreviation; blank for foreign address"},
			{Name: "ZIPCode", Start: 169, End: 173, Type: Numeric, Required: false, Description: "ZIP code; blank for foreign address"},
			{Name: "ZIPExtension", Start: 174, End: 177, Type: Numeric, Required: false, Description: "ZIP+4"},
			{Name: "Blank178", Start: 178, End: 181, Type: Blank, Required: false, Description: "Reserved"},
			{Name: "ForeignStateProvince", Start: 182, End: 204, Type: Alpha, Required: false, Description: "Foreign state/province, 23 chars"},
			{Name: "ForeignPostalCode", Start: 205, End: 219, Type: Alpha, Required: false, Description: "Foreign postal code, 15 chars"},
			{Name: "CountryCode", Start: 220, End: 221, Type: Alpha, Required: false, Description: "Country code per SSA Appendix I; blank for USA"},
			{Name: "OrigEmploymentCode", Start: 222, End: 222, Type: Alpha, Required: false, Description: "Originally reported employment code A/H/M/Q/R/X; blank if no correction"},
			{Name: "CorrectEmploymentCode", Start: 223, End: 223, Type: Alpha, Required: true, Description: "Correct employment code: A=Agri H=Household M=Military Q=MQGE R=Regular X=Railroad"},
			{Name: "OrigThirdPartySick", Start: 224, End: 224, Type: Alpha, Required: false, Description: "Originally reported third-party sick pay indicator"},
			{Name: "CorrectThirdPartySick", Start: 225, End: 225, Type: Alpha, Required: false, Description: "Correct third-party sick pay indicator (1=yes, blank=no)"},
			{Name: "Blank226", Start: 226, End: 226, Type: Blank, Required: false, Description: "Reserved"},
			{Name: "KindOfEmployer", Start: 227, End: 227, Type: Alpha, Required: false, Description: "F=Federal S=State/Local(non-exempt) T=Tax-Exempt Y=State/Local(exempt) N=None apply"},
			{Name: "ContactName", Start: 228, End: 254, Type: Alpha, Required: false, Description: "Employer contact name, 27 chars"},
			{Name: "ContactPhone", Start: 255, End: 269, Type: Numeric, Required: false, Description: "Employer contact phone, 15 chars"},
			{Name: "PhoneExtension", Start: 270, End: 274, Type: Numeric, Required: false, Description: "Employer contact phone extension"},
			{Name: "ContactFax", Start: 275, End: 284, Type: Numeric, Required: false, Description: "Employer contact fax number"},
			{Name: "ContactEmail", Start: 285, End: 324, Type: Alpha, Required: false, Description: "Employer contact e-mail, 40 chars"},
			{Name: "Blank325", Start: 325, End: 1024, Type: Blank, Required: false, Description: "Reserved"},
		},

		// ── RCW (Employee Correction) ─────────────────────────────────────
		// Positions from SSA Pub 42-014 TY2024 §5.7.
		// Money amounts use 11-char format (right-justified, zero-filled).
		RCW: []Field{
			{Name: "RecordIdentifier", Start: 1, End: 3, Type: Fixed, Required: true},
			{Name: "OrigSSN", Start: 4, End: 12, Type: Numeric, Required: true, Description: "Originally reported SSN, 9 digits"},
			{Name: "CorrectSSN", Start: 13, End: 21, Type: Numeric, Required: false, Description: "Correct SSN (required only when correcting SSN)"},
			// Name fields: First / Middle / Last ordering per spec
			{Name: "OrigFirstName", Start: 22, End: 36, Type: Alpha, Required: false, Description: "Originally reported first name (15 chars)"},
			{Name: "OrigMiddleName", Start: 37, End: 51, Type: Alpha, Required: false, Description: "Originally reported middle name or initial (15 chars)"},
			{Name: "OrigLastName", Start: 52, End: 71, Type: Alpha, Required: false, Description: "Originally reported last name (20 chars)"},
			{Name: "CorrectFirstName", Start: 72, End: 86, Type: Alpha, Required: false, Description: "Correct first name (15 chars)"},
			{Name: "CorrectMiddleName", Start: 87, End: 101, Type: Alpha, Required: false, Description: "Correct middle name or initial (15 chars)"},
			{Name: "CorrectLastName", Start: 102, End: 121, Type: Alpha, Required: false, Description: "Correct last name (20 chars)"},
			// Address (22-char fields per spec)
			{Name: "LocationAddress", Start: 122, End: 143, Type: Alpha, Required: false},
			{Name: "DeliveryAddress", Start: 144, End: 165, Type: Alpha, Required: false},
			{Name: "City", Start: 166, End: 187, Type: Alpha, Required: false},
			{Name: "StateAbbrev", Start: 188, End: 189, Type: Alpha, Required: false},
			{Name: "ZIPCode", Start: 190, End: 194, Type: Numeric, Required: false},
			{Name: "ZIPExtension", Start: 195, End: 198, Type: Numeric, Required: false},
			{Name: "Blank199", Start: 199, End: 203, Type: Blank, Required: false},
			{Name: "ForeignStateProvince", Start: 204, End: 226, Type: Alpha, Required: false},
			{Name: "ForeignPostalCode", Start: 227, End: 241, Type: Alpha, Required: false},
			{Name: "CountryCode", Start: 242, End: 243, Type: Alpha, Required: false},
			// Money amounts (11-char, positions 244-397 = Boxes 1-7)
			{Name: "OrigWagesTipsOther", Start: 244, End: 254, Type: Money11, Required: false, Description: "Box 1 orig"},
			{Name: "CorrectWagesTipsOther", Start: 255, End: 265, Type: Money11, Required: false, Description: "Box 1 corr"},
			{Name: "OrigFedIncomeTax", Start: 266, End: 276, Type: Money11, Required: false, Description: "Box 2 orig"},
			{Name: "CorrectFedIncomeTax", Start: 277, End: 287, Type: Money11, Required: false, Description: "Box 2 corr"},
			{Name: "OrigSSWages", Start: 288, End: 298, Type: Money11, Required: false, Description: "Box 3 orig"},
			{Name: "CorrectSSWages", Start: 299, End: 309, Type: Money11, Required: false, Description: "Box 3 corr"},
			{Name: "OrigSSTax", Start: 310, End: 320, Type: Money11, Required: false, Description: "Box 4 orig"},
			{Name: "CorrectSSTax", Start: 321, End: 331, Type: Money11, Required: false, Description: "Box 4 corr"},
			{Name: "OrigMedicareWages", Start: 332, End: 342, Type: Money11, Required: false, Description: "Box 5 orig"},
			{Name: "CorrectMedicareWages", Start: 343, End: 353, Type: Money11, Required: false, Description: "Box 5 corr"},
			{Name: "OrigMedicareTax", Start: 354, End: 364, Type: Money11, Required: false, Description: "Box 6 orig"},
			{Name: "CorrectMedicareTax", Start: 365, End: 375, Type: Money11, Required: false, Description: "Box 6 corr"},
			{Name: "OrigSSTips", Start: 376, End: 386, Type: Money11, Required: false, Description: "Box 7 orig"},
			{Name: "CorrectSSTips", Start: 387, End: 397, Type: Money11, Required: false, Description: "Box 7 corr"},
			// 398-419: was Box 9 Advance EIC, eliminated 2011
			{Name: "Blank398", Start: 398, End: 419, Type: Blank, Required: false, Description: "Reserved (was Box 9 Advance EIC, eliminated 2011)"},
			// Box 10 Dependent Care Benefits (420-441)
			{Name: "OrigDependentCare", Start: 420, End: 430, Type: Money11, Required: false, Description: "Box 10 orig — Dependent Care Benefits"},
			{Name: "CorrectDependentCare", Start: 431, End: 441, Type: Money11, Required: false, Description: "Box 10 corr"},
			// Box 12 codes in RCW (Code D, E, F, G, H, W, Q, C, V, Y, AA, BB, DD, FF)
			{Name: "OrigCode401k", Start: 442, End: 452, Type: Money11, Required: false, Description: "Box 12 Code D orig — 401(k) elective deferrals"},
			{Name: "CorrectCode401k", Start: 453, End: 463, Type: Money11, Required: false, Description: "Box 12 Code D corr"},
			{Name: "OrigCode403b", Start: 464, End: 474, Type: Money11, Required: false, Description: "Box 12 Code E orig — 403(b) elective deferrals"},
			{Name: "CorrectCode403b", Start: 475, End: 485, Type: Money11, Required: false, Description: "Box 12 Code E corr"},
			{Name: "OrigCodeF", Start: 486, End: 496, Type: Money11, Required: false, Description: "Box 12 Code F orig — 408(k)(6) SEP"},
			{Name: "CorrectCodeF", Start: 497, End: 507, Type: Money11, Required: false, Description: "Box 12 Code F corr"},
			{Name: "OrigCode457bGovt", Start: 508, End: 518, Type: Money11, Required: false, Description: "Box 12 Code G orig — 457(b) govt plan deferrals"},
			{Name: "CorrectCode457bGovt", Start: 519, End: 529, Type: Money11, Required: false, Description: "Box 12 Code G corr"},
			{Name: "OrigCodeH", Start: 530, End: 540, Type: Money11, Required: false, Description: "Box 12 Code H orig — 501(c)(18)(D) plan"},
			{Name: "CorrectCodeH", Start: 541, End: 551, Type: Money11, Required: false, Description: "Box 12 Code H corr"},
			{Name: "OrigTIBDeferredComp", Start: 552, End: 562, Type: Money11, Required: false, Description: "Total deferred comp (TIB format only, 1987-2005)"},
			{Name: "CorrectTIBDeferredComp", Start: 563, End: 573, Type: Money11, Required: false, Description: "Total deferred comp corr (TIB only)"},
			{Name: "Blank574", Start: 574, End: 595, Type: Blank, Required: false},
			// Box 11 Nonqualified Plans — two positions (457 and non-457 portions)
			{Name: "OrigNonqualPlan457", Start: 596, End: 606, Type: Money11, Required: false, Description: "Box 11 orig — Nonqualified Plan Section 457"},
			{Name: "CorrectNonqualPlan457", Start: 607, End: 617, Type: Money11, Required: false, Description: "Box 11 corr (Section 457)"},
			{Name: "OrigCodeW_HSA", Start: 618, End: 628, Type: Money11, Required: false, Description: "Box 12 Code W orig — employer HSA contributions"},
			{Name: "CorrectCodeW_HSA", Start: 629, End: 639, Type: Money11, Required: false, Description: "Box 12 Code W corr"},
			{Name: "OrigNonqualNotSection457", Start: 640, End: 650, Type: Money11, Required: false, Description: "Box 11 orig — Nonqualified Plan Not Section 457"},
			{Name: "CorrectNonqualNotSection457", Start: 651, End: 661, Type: Money11, Required: false, Description: "Box 11 corr (Non-457)"},
			{Name: "OrigCodeQ", Start: 662, End: 672, Type: Money11, Required: false, Description: "Box 12 Code Q orig — nontaxable combat pay"},
			{Name: "CorrectCodeQ", Start: 673, End: 683, Type: Money11, Required: false, Description: "Box 12 Code Q corr"},
			{Name: "Blank684", Start: 684, End: 705, Type: Blank, Required: false},
			{Name: "OrigCodeC", Start: 706, End: 716, Type: Money11, Required: false, Description: "Box 12 Code C orig — taxable cost of group-term life insurance"},
			{Name: "CorrectCodeC", Start: 717, End: 727, Type: Money11, Required: false, Description: "Box 12 Code C corr"},
			{Name: "OrigCodeV", Start: 728, End: 738, Type: Money11, Required: false, Description: "Box 12 Code V orig — income from nonstatutory stock options"},
			{Name: "CorrectCodeV", Start: 739, End: 749, Type: Money11, Required: false, Description: "Box 12 Code V corr"},
			{Name: "OrigCodeY", Start: 750, End: 760, Type: Money11, Required: false, Description: "Box 12 Code Y orig — deferrals under 409A NQDC"},
			{Name: "CorrectCodeY", Start: 761, End: 771, Type: Money11, Required: false, Description: "Box 12 Code Y corr"},
			{Name: "OrigCodeAA_Roth401k", Start: 772, End: 782, Type: Money11, Required: false, Description: "Box 12 Code AA orig — designated Roth 401(k)"},
			{Name: "CorrectCodeAA_Roth401k", Start: 783, End: 793, Type: Money11, Required: false, Description: "Box 12 Code AA corr"},
			{Name: "OrigCodeBB_Roth403b", Start: 794, End: 804, Type: Money11, Required: false, Description: "Box 12 Code BB orig — designated Roth 403(b)"},
			{Name: "CorrectCodeBB_Roth403b", Start: 805, End: 815, Type: Money11, Required: false, Description: "Box 12 Code BB corr"},
			{Name: "OrigCodeDD_EmpHealth", Start: 816, End: 826, Type: Money11, Required: false, Description: "Box 12 Code DD orig — employer-sponsored health coverage cost"},
			{Name: "CorrectCodeDD_EmpHealth", Start: 827, End: 837, Type: Money11, Required: false, Description: "Box 12 Code DD corr"},
			{Name: "OrigCodeFF_QSEHRA", Start: 838, End: 848, Type: Money11, Required: false, Description: "Box 12 Code FF orig — QSEHRA permitted benefits"},
			{Name: "CorrectCodeFF_QSEHRA", Start: 849, End: 859, Type: Money11, Required: false, Description: "Box 12 Code FF corr"},
			{Name: "Blank860", Start: 860, End: 1002, Type: Blank, Required: false},
			// Box 13 checkboxes — each has orig and correct indicator
			{Name: "OrigStatutoryEmployee", Start: 1003, End: 1003, Type: Alpha, Required: false, Description: "Box 13 Statutory Employee orig (1=yes 0=no)"},
			{Name: "CorrectStatutoryEmployee", Start: 1004, End: 1004, Type: Alpha, Required: false, Description: "Box 13 Statutory Employee correct"},
			{Name: "OrigRetirementPlan", Start: 1005, End: 1005, Type: Alpha, Required: false, Description: "Box 13 Retirement Plan orig"},
			{Name: "CorrectRetirementPlan", Start: 1006, End: 1006, Type: Alpha, Required: false, Description: "Box 13 Retirement Plan correct"},
			{Name: "OrigThirdPartySickPay", Start: 1007, End: 1007, Type: Alpha, Required: false, Description: "Box 13 Third-Party Sick Pay orig"},
			{Name: "CorrectThirdPartySickPay", Start: 1008, End: 1008, Type: Alpha, Required: false, Description: "Box 13 Third-Party Sick Pay correct"},
			{Name: "Blank1009", Start: 1009, End: 1024, Type: Blank, Required: false},
		},

		// ── RCO (Employee Optional) ───────────────────────────────────────
		// Required if Box 8 (Allocated Tips) or selected Box 12 codes need correction.
		// SSA Pub 42-014 TY2024 §5.8. Money fields are 11 chars.
		// TY2021-2023 do NOT include the Code II field (positions 277-298); ty2024() adds it.
		RCO: []Field{
			{Name: "RecordIdentifier", Start: 1, End: 3, Type: Fixed, Required: true, Description: "Constant 'RCO'"},
			{Name: "Blank4", Start: 4, End: 12, Type: Blank, Required: false, Description: "Reserved for SSA use"},
			{Name: "OrigAllocatedTips", Start: 13, End: 23, Type: Money11, Required: false, Description: "Box 8 orig — Allocated Tips"},
			{Name: "CorrectAllocatedTips", Start: 24, End: 34, Type: Money11, Required: false, Description: "Box 8 corr"},
			{Name: "OrigUncollectedEETax", Start: 35, End: 45, Type: Money11, Required: false, Description: "Box 12 Codes A&B orig — uncollected EE tax on tips"},
			{Name: "CorrectUncollectedEETax", Start: 46, End: 56, Type: Money11, Required: false, Description: "Box 12 Codes A&B corr"},
			{Name: "OrigCodeR_MSA", Start: 57, End: 67, Type: Money11, Required: false, Description: "Box 12 Code R orig — Medical Savings Account"},
			{Name: "CorrectCodeR_MSA", Start: 68, End: 78, Type: Money11, Required: false, Description: "Box 12 Code R corr"},
			{Name: "OrigCodeS_SIMPLE", Start: 79, End: 89, Type: Money11, Required: false, Description: "Box 12 Code S orig — Simple Retirement Account"},
			{Name: "CorrectCodeS_SIMPLE", Start: 90, End: 100, Type: Money11, Required: false, Description: "Box 12 Code S corr"},
			{Name: "OrigCodeT_Adoption", Start: 101, End: 111, Type: Money11, Required: false, Description: "Box 12 Code T orig — Qualified Adoption Expenses"},
			{Name: "CorrectCodeT_Adoption", Start: 112, End: 122, Type: Money11, Required: false, Description: "Box 12 Code T corr"},
			{Name: "OrigCodeM_UncollSS", Start: 123, End: 133, Type: Money11, Required: false, Description: "Box 12 Code M orig — uncollected SS/RRTA on group-term life"},
			{Name: "CorrectCodeM_UncollSS", Start: 134, End: 144, Type: Money11, Required: false, Description: "Box 12 Code M corr"},
			{Name: "OrigCodeN_UncollMed", Start: 145, End: 155, Type: Money11, Required: false, Description: "Box 12 Code N orig — uncollected Medicare on group-term life"},
			{Name: "CorrectCodeN_UncollMed", Start: 156, End: 166, Type: Money11, Required: false, Description: "Box 12 Code N corr"},
			{Name: "OrigCodeZ_409A", Start: 167, End: 177, Type: Money11, Required: false, Description: "Box 12 Code Z orig — income under 409A NQDC that fails section"},
			{Name: "CorrectCodeZ_409A", Start: 178, End: 188, Type: Money11, Required: false, Description: "Box 12 Code Z corr"},
			{Name: "Blank189", Start: 189, End: 210, Type: Blank, Required: false},
			{Name: "OrigCodeEE_Roth457b", Start: 211, End: 221, Type: Money11, Required: false, Description: "Box 12 Code EE orig — designated Roth 457(b)"},
			{Name: "CorrectCodeEE_Roth457b", Start: 222, End: 232, Type: Money11, Required: false, Description: "Box 12 Code EE corr"},
			{Name: "OrigCodeGG_83i", Start: 233, End: 243, Type: Money11, Required: false, Description: "Box 12 Code GG orig — income from qualified equity grants (83(i))"},
			{Name: "CorrectCodeGG_83i", Start: 244, End: 254, Type: Money11, Required: false, Description: "Box 12 Code GG corr"},
			{Name: "OrigCodeHH_83iDeferral", Start: 255, End: 265, Type: Money11, Required: false, Description: "Box 12 Code HH orig — aggregate deferrals under 83(i)"},
			{Name: "CorrectCodeHH_83iDeferral", Start: 266, End: 276, Type: Money11, Required: false, Description: "Box 12 Code HH corr"},
			// TY2021-2023 end with blank 277-1024. TY2024 adds Code II at 277-298; ty2024() appends it.
			{Name: "Blank277", Start: 277, End: 1024, Type: Blank, Required: false},
		},

		// ── RCS (State Record) ────────────────────────────────────────────
		// Optional. SSA and IRS do NOT process this record; for state agencies only.
		// SSA Pub 42-014 TY2024 §5.9. Key fields only — state taxable wages / income tax.
		RCS: []Field{
			{Name: "RecordIdentifier", Start: 1, End: 3, Type: Fixed, Required: true, Description: "Constant 'RCS'"},
			{Name: "StateCode", Start: 4, End: 5, Type: Numeric, Required: true, Description: "State postal numeric code (Appendix H)"},
			{Name: "OrigTaxingEntityCode", Start: 6, End: 10, Type: Alpha, Required: false},
			{Name: "CorrectTaxingEntityCode", Start: 11, End: 15, Type: Alpha, Required: false},
			{Name: "OrigSSN", Start: 16, End: 24, Type: Numeric, Required: false},
			{Name: "CorrectSSN", Start: 25, End: 33, Type: Numeric, Required: false},
			{Name: "OrigFirstName", Start: 34, End: 48, Type: Alpha, Required: false},
			{Name: "OrigMiddleName", Start: 49, End: 63, Type: Alpha, Required: false},
			{Name: "OrigLastName", Start: 64, End: 83, Type: Alpha, Required: false},
			{Name: "CorrectFirstName", Start: 84, End: 98, Type: Alpha, Required: false},
			{Name: "CorrectMiddleName", Start: 99, End: 113, Type: Alpha, Required: false},
			{Name: "CorrectLastName", Start: 114, End: 133, Type: Alpha, Required: false},
			{Name: "LocationAddress", Start: 134, End: 155, Type: Alpha, Required: false},
			{Name: "DeliveryAddress", Start: 156, End: 177, Type: Alpha, Required: false},
			{Name: "City", Start: 178, End: 199, Type: Alpha, Required: false},
			{Name: "StateAbbrev", Start: 200, End: 201, Type: Alpha, Required: false},
			{Name: "ZIPCode", Start: 202, End: 206, Type: Numeric, Required: false},
			{Name: "ZIPExtension", Start: 207, End: 210, Type: Numeric, Required: false},
			{Name: "Blank211", Start: 211, End: 395, Type: Blank, Required: false, Description: "Optional/state-specific fields not required by SSA"},
			{Name: "StateCode2", Start: 396, End: 397, Type: Numeric, Required: false, Description: "State code for Box 16/17 data"},
			{Name: "OrigStateWages", Start: 398, End: 408, Type: Money11, Required: false, Description: "Box 16 orig — state taxable wages"},
			{Name: "CorrectStateWages", Start: 409, End: 419, Type: Money11, Required: false, Description: "Box 16 corr"},
			{Name: "OrigStateIncomeTax", Start: 420, End: 430, Type: Money11, Required: false, Description: "Box 17 orig — state income tax withheld"},
			{Name: "CorrectStateIncomeTax", Start: 431, End: 441, Type: Money11, Required: false, Description: "Box 17 corr"},
			{Name: "Blank442", Start: 442, End: 1024, Type: Blank, Required: false},
		},

		// ── RCT (Total) ──────────────────────────────────────────────────
		// Totals all RCW money fields for the preceding RCE. 15-char money fields.
		// SSA Pub 42-014 TY2024 §5.10.
		RCT: []Field{
			{Name: "RecordIdentifier", Start: 1, End: 3, Type: Fixed, Required: true},
			{Name: "TotalRCWRecords", Start: 4, End: 10, Type: Numeric, Required: true, Description: "Total RCW count, 7 digits zero-padded"},
			{Name: "OrigTotalWagesTips", Start: 11, End: 25, Type: Money15, Required: false, Description: "Box 1 orig total"},
			{Name: "CorrectTotalWagesTips", Start: 26, End: 40, Type: Money15, Required: false, Description: "Box 1 corr total"},
			{Name: "OrigTotalFedIncomeTax", Start: 41, End: 55, Type: Money15, Required: false, Description: "Box 2 orig total"},
			{Name: "CorrectTotalFedIncomeTax", Start: 56, End: 70, Type: Money15, Required: false, Description: "Box 2 corr total"},
			{Name: "OrigTotalSSWages", Start: 71, End: 85, Type: Money15, Required: false, Description: "Box 3 orig total"},
			{Name: "CorrectTotalSSWages", Start: 86, End: 100, Type: Money15, Required: false, Description: "Box 3 corr total"},
			{Name: "OrigTotalSSTax", Start: 101, End: 115, Type: Money15, Required: false, Description: "Box 4 orig total"},
			{Name: "CorrectTotalSSTax", Start: 116, End: 130, Type: Money15, Required: false, Description: "Box 4 corr total"},
			{Name: "OrigTotalMedicareWages", Start: 131, End: 145, Type: Money15, Required: false, Description: "Box 5 orig total"},
			{Name: "CorrectTotalMedicareWages", Start: 146, End: 160, Type: Money15, Required: false, Description: "Box 5 corr total"},
			{Name: "OrigTotalMedicareTax", Start: 161, End: 175, Type: Money15, Required: false, Description: "Box 6 orig total"},
			{Name: "CorrectTotalMedicareTax", Start: 176, End: 190, Type: Money15, Required: false, Description: "Box 6 corr total"},
			{Name: "OrigTotalSSTips", Start: 191, End: 205, Type: Money15, Required: false, Description: "Box 7 orig total"},
			{Name: "CorrectTotalSSTips", Start: 206, End: 220, Type: Money15, Required: false, Description: "Box 7 corr total"},
			{Name: "Blank221", Start: 221, End: 250, Type: Blank, Required: false},
			{Name: "OrigTotalDependentCare", Start: 251, End: 265, Type: Money15, Required: false, Description: "Box 10 orig total"},
			{Name: "CorrectTotalDependentCare", Start: 266, End: 280, Type: Money15, Required: false, Description: "Box 10 corr total"},
			{Name: "OrigTotalCode401k", Start: 281, End: 295, Type: Money15, Required: false, Description: "Box 12 Code D orig total"},
			{Name: "CorrectTotalCode401k", Start: 296, End: 310, Type: Money15, Required: false, Description: "Box 12 Code D corr total"},
			{Name: "OrigTotalCode403b", Start: 311, End: 325, Type: Money15, Required: false, Description: "Box 12 Code E orig total"},
			{Name: "CorrectTotalCode403b", Start: 326, End: 340, Type: Money15, Required: false, Description: "Box 12 Code E corr total"},
			{Name: "OrigTotalCodeF", Start: 341, End: 355, Type: Money15, Required: false, Description: "Box 12 Code F orig total"},
			{Name: "CorrectTotalCodeF", Start: 356, End: 370, Type: Money15, Required: false, Description: "Box 12 Code F corr total"},
			{Name: "OrigTotalCode457bGovt", Start: 371, End: 385, Type: Money15, Required: false, Description: "Box 12 Code G orig total"},
			{Name: "CorrectTotalCode457bGovt", Start: 386, End: 400, Type: Money15, Required: false, Description: "Box 12 Code G corr total"},
			{Name: "OrigTotalCodeH", Start: 401, End: 415, Type: Money15, Required: false, Description: "Box 12 Code H orig total"},
			{Name: "CorrectTotalCodeH", Start: 416, End: 430, Type: Money15, Required: false, Description: "Box 12 Code H corr total"},
			{Name: "OrigTotalTIBDeferredComp", Start: 431, End: 445, Type: Money15, Required: false, Description: "TIB total deferred comp orig"},
			{Name: "CorrectTotalTIBDeferredComp", Start: 446, End: 460, Type: Money15, Required: false, Description: "TIB total deferred comp corr"},
			{Name: "Blank461", Start: 461, End: 490, Type: Blank, Required: false},
			{Name: "OrigTotalNonqualPlan457", Start: 491, End: 505, Type: Money15, Required: false, Description: "Box 11 Section 457 orig total"},
			{Name: "CorrectTotalNonqualPlan457", Start: 506, End: 520, Type: Money15, Required: false, Description: "Box 11 Section 457 corr total"},
			{Name: "OrigTotalCodeW_HSA", Start: 521, End: 535, Type: Money15, Required: false, Description: "Box 12 Code W orig total"},
			{Name: "CorrectTotalCodeW_HSA", Start: 536, End: 550, Type: Money15, Required: false, Description: "Box 12 Code W corr total"},
			{Name: "OrigTotalNonqualNotSection457", Start: 551, End: 565, Type: Money15, Required: false, Description: "Box 11 Non-457 orig total"},
			{Name: "CorrectTotalNonqualNotSection457", Start: 566, End: 580, Type: Money15, Required: false, Description: "Box 11 Non-457 corr total"},
			{Name: "OrigTotalCodeQ", Start: 581, End: 595, Type: Money15, Required: false, Description: "Box 12 Code Q orig total"},
			{Name: "CorrectTotalCodeQ", Start: 596, End: 610, Type: Money15, Required: false, Description: "Box 12 Code Q corr total"},
			{Name: "Blank611", Start: 611, End: 640, Type: Blank, Required: false},
			{Name: "OrigTotalCodeC", Start: 641, End: 655, Type: Money15, Required: false, Description: "Box 12 Code C orig total"},
			{Name: "CorrectTotalCodeC", Start: 656, End: 670, Type: Money15, Required: false, Description: "Box 12 Code C corr total"},
			{Name: "OrigTotalCodeV", Start: 671, End: 685, Type: Money15, Required: false, Description: "Box 12 Code V orig total"},
			{Name: "CorrectTotalCodeV", Start: 686, End: 700, Type: Money15, Required: false, Description: "Box 12 Code V corr total"},
			{Name: "OrigTotalCodeY", Start: 701, End: 715, Type: Money15, Required: false, Description: "Box 12 Code Y orig total"},
			{Name: "CorrectTotalCodeY", Start: 716, End: 730, Type: Money15, Required: false, Description: "Box 12 Code Y corr total"},
			{Name: "OrigTotalCodeAA_Roth401k", Start: 731, End: 745, Type: Money15, Required: false, Description: "Box 12 Code AA orig total"},
			{Name: "CorrectTotalCodeAA_Roth401k", Start: 746, End: 760, Type: Money15, Required: false, Description: "Box 12 Code AA corr total"},
			{Name: "OrigTotalCodeBB_Roth403b", Start: 761, End: 775, Type: Money15, Required: false, Description: "Box 12 Code BB orig total"},
			{Name: "CorrectTotalCodeBB_Roth403b", Start: 776, End: 790, Type: Money15, Required: false, Description: "Box 12 Code BB corr total"},
			{Name: "OrigTotalCodeDD_EmpHealth", Start: 791, End: 805, Type: Money15, Required: false, Description: "Box 12 Code DD orig total"},
			{Name: "CorrectTotalCodeDD_EmpHealth", Start: 806, End: 820, Type: Money15, Required: false, Description: "Box 12 Code DD corr total"},
			{Name: "OrigTotalCodeFF_QSEHRA", Start: 821, End: 835, Type: Money15, Required: false, Description: "Box 12 Code FF orig total"},
			{Name: "CorrectTotalCodeFF_QSEHRA", Start: 836, End: 850, Type: Money15, Required: false, Description: "Box 12 Code FF corr total"},
			{Name: "Blank851", Start: 851, End: 1024, Type: Blank, Required: false},
		},

		// ── RCF (Final) ──────────────────────────────────────────────────
		RCF: []Field{
			{Name: "RecordIdentifier", Start: 1, End: 3, Type: Fixed, Required: true},
			{Name: "TotalRCWRecords", Start: 4, End: 10, Type: Numeric, Required: true, Description: "Total RCW count, 7 digits zero-padded"},
			{Name: "Blank11", Start: 11, End: 1024, Type: Blank, Required: false},
		},
	}
}
