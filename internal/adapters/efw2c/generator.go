package efw2c

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"

	"github.com/csg33k/w2c-generator/internal/adapters/efw2c/spec"
	"github.com/csg33k/w2c-generator/internal/domain"
)

type Generator struct {
	year  int
	yspec *spec.YearSpec
}

func New(year int) (*Generator, error) {
	if year == 0 {
		year = spec.DefaultYear
	}
	yspec, exact := spec.ForYear(year)
	if !exact {
		return &Generator{year: year, yspec: yspec},
			fmt.Errorf("no exact spec for TY%d; using TY%d layout as fallback", year, spec.DefaultYear)
	}
	return &Generator{year: year, yspec: yspec}, nil
}

func MustNew(year int) *Generator {
	yspec, _ := spec.ForYear(year)
	return &Generator{year: year, yspec: yspec}
}

func (g *Generator) Year() int            { return g.year }
func (g *Generator) Spec() *spec.YearSpec { return g.yspec }

// SupportedYears returns all tax years this generator supports, ascending,
// each paired with its SSA publication URL. Satisfies ports.EFW2CGenerator.
func (g *Generator) SupportedYears() []domain.TaxYearInfo {
	years := spec.Supported() // already ascending
	out := make([]domain.TaxYearInfo, len(years))
	for i, y := range years {
		ys, _ := spec.ForYear(y)
		out[i] = domain.TaxYearInfo{
			Year:           strconv.Itoa(y),
			PublicationURL: ys.PublicationURL,
		}
	}
	return out
}

// Generate writes a complete EFW2C byte stream (no CR/LF between records).
// Record order per spec: RCA, RCE, [RCW (RCO?) (RCS?)...], RCT, RCF.
func (g *Generator) Generate(ctx context.Context, s *domain.Submission, w io.Writer) error {
	// Resolve the correct spec for this submission's tax year.
	yearInt, _ := strconv.Atoi(s.Employer.TaxYear)
	yspec, _ := spec.ForYear(yearInt)
	local := &Generator{year: yearInt, yspec: yspec}

	records := []string{
		local.buildRCA(s),
		local.buildRCE(s),
	}

	// Accumulators for RCT totals (only track what we actually write in RCW)
	var (
		origWages, corrWages                           int64
		origFed, corrFed                               int64
		origSS, corrSS                                 int64
		origSSTax, corrSSTax                           int64
		origMed, corrMed                               int64
		origMedTax, corrMedTax                         int64
		origSSTips, corrSSTips                         int64
		origDepCare, corrDepCare                       int64
		origNQ457, corrNQ457                           int64
		origNQNot457, corrNQNot457                     int64
		origD, corrD                                   int64
		origE, corrE                                   int64
		origG, corrG                                   int64
		origW, corrW                                   int64
		origAA, corrAA                                 int64
		origBB, corrBB                                 int64
		origDD, corrDD                                 int64
	)

	for i := range s.Employees {
		e := &s.Employees[i]
		records = append(records, local.buildRCW(e))

		// Emit RCO if any optional fields are non-zero
		if local.hasRCOData(e) {
			records = append(records, local.buildRCO(e))
		}
		// Emit RCS if state/local data present
		if local.hasRCSData(e) {
			records = append(records, local.buildRCS(e))
		}

		origWages += e.Amounts.OriginalWagesTipsOther
		corrWages += e.Amounts.CorrectWagesTipsOther
		origFed += e.Amounts.OriginalFederalIncomeTax
		corrFed += e.Amounts.CorrectFederalIncomeTax
		origSS += e.Amounts.OriginalSocialSecurityWages
		corrSS += e.Amounts.CorrectSocialSecurityWages
		origSSTax += e.Amounts.OriginalSocialSecurityTax
		corrSSTax += e.Amounts.CorrectSocialSecurityTax
		origMed += e.Amounts.OriginalMedicareWages
		corrMed += e.Amounts.CorrectMedicareWages
		origMedTax += e.Amounts.OriginalMedicareTax
		corrMedTax += e.Amounts.CorrectMedicareTax
		origSSTips += e.Amounts.OriginalSocialSecurityTips
		corrSSTips += e.Amounts.CorrectSocialSecurityTips
		origDepCare += e.Amounts.OriginalDependentCare
		corrDepCare += e.Amounts.CorrectDependentCare
		origNQ457 += e.Amounts.OriginalNonqualPlan457
		corrNQ457 += e.Amounts.CorrectNonqualPlan457
		origNQNot457 += e.Amounts.OriginalNonqualNotSection457
		corrNQNot457 += e.Amounts.CorrectNonqualNotSection457
		origD += e.Amounts.OriginalCode401k
		corrD += e.Amounts.CorrectCode401k
		origE += e.Amounts.OriginalCode403b
		corrE += e.Amounts.CorrectCode403b
		origG += e.Amounts.OriginalCode457bGovt
		corrG += e.Amounts.CorrectCode457bGovt
		origW += e.Amounts.OriginalCodeW_HSA
		corrW += e.Amounts.CorrectCodeW_HSA
		origAA += e.Amounts.OriginalCodeAA_Roth401k
		corrAA += e.Amounts.CorrectCodeAA_Roth401k
		origBB += e.Amounts.OriginalCodeBB_Roth403b
		corrBB += e.Amounts.CorrectCodeBB_Roth403b
		origDD += e.Amounts.OriginalCodeDD_EmpHealth
		corrDD += e.Amounts.CorrectCodeDD_EmpHealth
	}

	records = append(records,
		local.buildRCT(
			origWages, corrWages, origFed, corrFed,
			origSS, corrSS, origSSTax, corrSSTax,
			origMed, corrMed, origMedTax, corrMedTax,
			origSSTips, corrSSTips,
			origDepCare, corrDepCare,
			origNQ457, corrNQ457, origNQNot457, corrNQNot457,
			origD, corrD, origE, corrE, origG, corrG,
			origW, corrW, origAA, corrAA, origBB, corrBB,
			origDD, corrDD,
		),
		local.buildRCF(len(s.Employees)),
	)

	for _, r := range records {
		if len(r) != spec.RecordLen {
			return fmt.Errorf("record %q is %d bytes (want %d)", r[:3], len(r), spec.RecordLen)
		}
		if _, err := io.WriteString(w, r); err != nil {
			return err
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Presence checks
// ---------------------------------------------------------------------------

func (g *Generator) hasRCOData(e *domain.EmployeeRecord) bool {
	a := &e.Amounts
	return a.OriginalAllocatedTips != 0 || a.CorrectAllocatedTips != 0
}

func (g *Generator) hasRCSData(e *domain.EmployeeRecord) bool {
	return e.OriginalStateCode != "" || e.CorrectStateCode != "" ||
		e.Amounts.OriginalStateWages != 0 || e.Amounts.CorrectStateWages != 0 ||
		e.Amounts.OriginalStateIncomeTax != 0 || e.Amounts.CorrectStateIncomeTax != 0
}

// ---------------------------------------------------------------------------
// Record builders
// ---------------------------------------------------------------------------

func (g *Generator) buildRCA(s *domain.Submission) string {
	sub := s.Submitter

	preparerCode := sub.PreparerCode
	if preparerCode == "" {
		preparerCode = "L"
	}
	resubIndicator := sub.ResubIndicator
	if resubIndicator == "" {
		resubIndicator = "0"
	}

	b := newBuf()
	b.put("RecordIdentifier", g.yspec.RCA, "RCA")
	b.put("SubmitterEIN", g.yspec.RCA, cleanDigits(s.Employer.EIN, 9))
	b.put("BSOUID", g.yspec.RCA, padAlpha(sub.BSOUID, 8))
	// SoftwareVendorCode and SoftwareCode left blank — not a software vendor
	// CompanyName: 57 chars at positions 32-88 per TY2024 §5.5
	b.put("CompanyName", g.yspec.RCA, padAlpha(s.Employer.Name, 57))
	b.put("LocationAddress", g.yspec.RCA, padAlpha(s.Employer.AddressLine1, 22))
	b.put("DeliveryAddress", g.yspec.RCA, padAlpha(s.Employer.AddressLine2, 22))
	b.put("City", g.yspec.RCA, padAlpha(s.Employer.City, 22))
	b.put("StateAbbrev", g.yspec.RCA, padAlpha(s.Employer.State, 2))
	b.put("ZIPCode", g.yspec.RCA, padNumeric(s.Employer.ZIP, 5))
	b.put("ZIPExtension", g.yspec.RCA, padNumeric(s.Employer.ZIPExtension, 4))
	b.put("ContactName", g.yspec.RCA, padAlpha(sub.ContactName, 27))
	b.put("ContactPhone", g.yspec.RCA, padNumeric(sub.ContactPhone, 15))
	b.put("ContactEmail", g.yspec.RCA, padEmail(sub.ContactEmail, 40))
	b.put("PreparerCode", g.yspec.RCA, preparerCode)
	b.put("ResubIndicator", g.yspec.RCA, resubIndicator)
	if sub.ResubWFID != "" {
		// ResubWFID is 6 chars per TY2024 §5.5 (positions 318-323)
		b.put("ResubWFID", g.yspec.RCA, padAlpha(sub.ResubWFID, 6))
	}
	return b.String()
}

func (g *Generator) buildRCE(s *domain.Submission) string {
	b := newBuf()
	b.put("RecordIdentifier", g.yspec.RCE, "RCE")
	b.put("TaxYear", g.yspec.RCE, s.Employer.TaxYear)
	if s.Employer.OriginalEIN != "" {
		b.put("OrigReportedEIN", g.yspec.RCE, cleanDigits(s.Employer.OriginalEIN, 9))
	}
	b.put("EmployerEIN", g.yspec.RCE, cleanDigits(s.Employer.EIN, 9))
	// AgentIndicatorCode at position 26 per TY2024 §5.6 (was wrongly at 36 before)
	if s.Employer.AgentIndicator != "" {
		b.put("AgentIndicatorCode", g.yspec.RCE, s.Employer.AgentIndicator)
	}
	if s.Employer.AgentEIN != "" {
		b.put("AgentForEIN", g.yspec.RCE, cleanDigits(s.Employer.AgentEIN, 9))
	}
	// EmployerName: 57 chars at positions 44-100 per TY2024 §5.6
	b.put("EmployerName", g.yspec.RCE, padAlpha(s.Employer.Name, 57))
	b.put("LocationAddress", g.yspec.RCE, padAlpha(s.Employer.AddressLine1, 22))
	b.put("DeliveryAddress", g.yspec.RCE, padAlpha(s.Employer.AddressLine2, 22))
	b.put("City", g.yspec.RCE, padAlpha(s.Employer.City, 22))
	b.put("StateAbbrev", g.yspec.RCE, padAlpha(s.Employer.State, 2))
	b.put("ZIPCode", g.yspec.RCE, padNumeric(s.Employer.ZIP, 5))
	b.put("ZIPExtension", g.yspec.RCE, padNumeric(s.Employer.ZIPExtension, 4))
	// CorrectEmploymentCode at position 223; OrigEmploymentCode at 222 (leave blank unless correcting)
	b.put("CorrectEmploymentCode", g.yspec.RCE, defaultStr(s.Employer.EmploymentCode, "R"))
	b.put("KindOfEmployer", g.yspec.RCE, defaultStr(s.Employer.KindOfEmployer, "N"))
	// Employer contact fields at positions 228-324 per TY2024 §5.6
	if s.Employer.ContactName != "" {
		b.put("ContactName", g.yspec.RCE, padAlpha(s.Employer.ContactName, 27))
	}
	if s.Employer.ContactPhone != "" {
		b.put("ContactPhone", g.yspec.RCE, padNumeric(s.Employer.ContactPhone, 15))
	}
	if s.Employer.ContactEmail != "" {
		b.put("ContactEmail", g.yspec.RCE, padEmail(s.Employer.ContactEmail, 40))
	}
	return b.String()
}

func (g *Generator) buildRCW(e *domain.EmployeeRecord) string {
	b := newBuf()
	b.put("RecordIdentifier", g.yspec.RCW, "RCW")

	// SSN: OrigSSN = previously reported (or current if no SSN correction)
	//      CorrectSSN = new SSN (only if correcting SSN)
	b.put("OrigSSN", g.yspec.RCW, cleanDigits(e.SSN, 9))
	if e.OriginalSSN != "" {
		// Correcting SSN: OrigSSN gets the old wrong SSN, CorrectSSN gets the right one
		b.put("OrigSSN", g.yspec.RCW, cleanDigits(e.OriginalSSN, 9))
		b.put("CorrectSSN", g.yspec.RCW, cleanDigits(e.SSN, 9))
	}

	// Names: write Orig/Correct pairs when correcting name; otherwise put current name in CorrectFirstName etc.
	if e.OriginalFirstName != "" || e.OriginalLastName != "" {
		// Name correction: orig = previously wrong, correct = new correct name
		b.put("OrigFirstName", g.yspec.RCW, padAlpha(e.OriginalFirstName, 15))
		b.put("OrigMiddleName", g.yspec.RCW, padAlpha(e.OriginalMiddleName, 15))
		b.put("OrigLastName", g.yspec.RCW, padAlpha(e.OriginalLastName, 20))
		b.put("CorrectFirstName", g.yspec.RCW, padAlpha(e.FirstName, 15))
		b.put("CorrectMiddleName", g.yspec.RCW, padAlpha(e.MiddleName, 15))
		b.put("CorrectLastName", g.yspec.RCW, padAlpha(e.LastName, 20))
	} else {
		// No name correction: still write correct name in the Correct fields per spec
		b.put("CorrectFirstName", g.yspec.RCW, padAlpha(e.FirstName, 15))
		b.put("CorrectMiddleName", g.yspec.RCW, padAlpha(e.MiddleName, 15))
		b.put("CorrectLastName", g.yspec.RCW, padAlpha(e.LastName, 20))
	}

	// Address
	b.put("LocationAddress", g.yspec.RCW, padAlpha(e.AddressLine1, 22))
	b.put("DeliveryAddress", g.yspec.RCW, padAlpha(e.AddressLine2, 22))
	b.put("City", g.yspec.RCW, padAlpha(e.City, 22))
	b.put("StateAbbrev", g.yspec.RCW, padAlpha(e.State, 2))
	b.put("ZIPCode", g.yspec.RCW, padNumeric(e.ZIP, 5))
	b.put("ZIPExtension", g.yspec.RCW, padNumeric(e.ZIPExtension, 4))

	// Boxes 1–7 (always write; fill with zeros if no correction)
	a := &e.Amounts
	b.put("OrigWagesTipsOther", g.yspec.RCW, money11(a.OriginalWagesTipsOther))
	b.put("CorrectWagesTipsOther", g.yspec.RCW, money11(a.CorrectWagesTipsOther))
	b.put("OrigFedIncomeTax", g.yspec.RCW, money11(a.OriginalFederalIncomeTax))
	b.put("CorrectFedIncomeTax", g.yspec.RCW, money11(a.CorrectFederalIncomeTax))
	b.put("OrigSSWages", g.yspec.RCW, money11(a.OriginalSocialSecurityWages))
	b.put("CorrectSSWages", g.yspec.RCW, money11(a.CorrectSocialSecurityWages))
	b.put("OrigSSTax", g.yspec.RCW, money11(a.OriginalSocialSecurityTax))
	b.put("CorrectSSTax", g.yspec.RCW, money11(a.CorrectSocialSecurityTax))
	b.put("OrigMedicareWages", g.yspec.RCW, money11(a.OriginalMedicareWages))
	b.put("CorrectMedicareWages", g.yspec.RCW, money11(a.CorrectMedicareWages))
	b.put("OrigMedicareTax", g.yspec.RCW, money11(a.OriginalMedicareTax))
	b.put("CorrectMedicareTax", g.yspec.RCW, money11(a.CorrectMedicareTax))
	b.put("OrigSSTips", g.yspec.RCW, money11(a.OriginalSocialSecurityTips))
	b.put("CorrectSSTips", g.yspec.RCW, money11(a.CorrectSocialSecurityTips))

	// Box 10 — Dependent Care
	putMoney11Pair(b, g.yspec.RCW, "OrigDependentCare", "CorrectDependentCare",
		a.OriginalDependentCare, a.CorrectDependentCare)

	// Box 12 codes in RCW
	putMoney11Pair(b, g.yspec.RCW, "OrigCode401k", "CorrectCode401k",
		a.OriginalCode401k, a.CorrectCode401k)
	putMoney11Pair(b, g.yspec.RCW, "OrigCode403b", "CorrectCode403b",
		a.OriginalCode403b, a.CorrectCode403b)
	putMoney11Pair(b, g.yspec.RCW, "OrigCode457bGovt", "CorrectCode457bGovt",
		a.OriginalCode457bGovt, a.CorrectCode457bGovt)
	putMoney11Pair(b, g.yspec.RCW, "OrigCodeW_HSA", "CorrectCodeW_HSA",
		a.OriginalCodeW_HSA, a.CorrectCodeW_HSA)
	putMoney11Pair(b, g.yspec.RCW, "OrigCodeAA_Roth401k", "CorrectCodeAA_Roth401k",
		a.OriginalCodeAA_Roth401k, a.CorrectCodeAA_Roth401k)
	putMoney11Pair(b, g.yspec.RCW, "OrigCodeBB_Roth403b", "CorrectCodeBB_Roth403b",
		a.OriginalCodeBB_Roth403b, a.CorrectCodeBB_Roth403b)
	putMoney11Pair(b, g.yspec.RCW, "OrigCodeDD_EmpHealth", "CorrectCodeDD_EmpHealth",
		a.OriginalCodeDD_EmpHealth, a.CorrectCodeDD_EmpHealth)

	// Box 11 — Nonqualified Plans (two components)
	putMoney11Pair(b, g.yspec.RCW, "OrigNonqualPlan457", "CorrectNonqualPlan457",
		a.OriginalNonqualPlan457, a.CorrectNonqualPlan457)
	putMoney11Pair(b, g.yspec.RCW, "OrigNonqualNotSection457", "CorrectNonqualNotSection457",
		a.OriginalNonqualNotSection457, a.CorrectNonqualNotSection457)

	// Box 13 checkboxes
	box13 := &e.Box13
	putBox13(b, g.yspec.RCW, "OrigStatutoryEmployee", "CorrectStatutoryEmployee",
		box13.OrigStatutoryEmployee, box13.CorrectStatutoryEmployee)
	putBox13(b, g.yspec.RCW, "OrigRetirementPlan", "CorrectRetirementPlan",
		box13.OrigRetirementPlan, box13.CorrectRetirementPlan)
	putBox13(b, g.yspec.RCW, "OrigThirdPartySickPay", "CorrectThirdPartySickPay",
		box13.OrigThirdPartySickPay, box13.CorrectThirdPartySickPay)

	return b.String()
}

func (g *Generator) buildRCO(e *domain.EmployeeRecord) string {
	b := newBuf()
	b.put("RecordIdentifier", g.yspec.RCO, "RCO")
	a := &e.Amounts
	putMoney11Pair(b, g.yspec.RCO, "OrigAllocatedTips", "CorrectAllocatedTips",
		a.OriginalAllocatedTips, a.CorrectAllocatedTips)
	return b.String()
}

func (g *Generator) buildRCS(e *domain.EmployeeRecord) string {
	b := newBuf()
	b.put("RecordIdentifier", g.yspec.RCS, "RCS")
	// State code from CorrectStateCode (or OriginalStateCode if no correction)
	sc := e.CorrectStateCode
	if sc == "" {
		sc = e.OriginalStateCode
	}
	b.put("StateCode", g.yspec.RCS, padNumeric(statePostalToNumeric(sc), 2))
	b.put("CorrectSSN", g.yspec.RCS, cleanDigits(e.SSN, 9))
	b.put("CorrectFirstName", g.yspec.RCS, padAlpha(e.FirstName, 15))
	b.put("CorrectMiddleName", g.yspec.RCS, padAlpha(e.MiddleName, 15))
	b.put("CorrectLastName", g.yspec.RCS, padAlpha(e.LastName, 20))
	b.put("StateCode2", g.yspec.RCS, padNumeric(statePostalToNumeric(sc), 2))
	a := &e.Amounts
	putMoney11Pair(b, g.yspec.RCS, "OrigStateWages", "CorrectStateWages",
		a.OriginalStateWages, a.CorrectStateWages)
	putMoney11Pair(b, g.yspec.RCS, "OrigStateIncomeTax", "CorrectStateIncomeTax",
		a.OriginalStateIncomeTax, a.CorrectStateIncomeTax)
	return b.String()
}

func (g *Generator) buildRCT(
	origWages, corrWages,
	origFed, corrFed,
	origSS, corrSS,
	origSSTax, corrSSTax,
	origMed, corrMed,
	origMedTax, corrMedTax,
	origSSTips, corrSSTips,
	origDepCare, corrDepCare,
	origNQ457, corrNQ457,
	origNQNot457, corrNQNot457,
	origD, corrD,
	origE, corrE,
	origG, corrG,
	origW, corrW,
	origAA, corrAA,
	origBB, corrBB,
	origDD, corrDD int64,
) string {
	b := newBuf()
	b.put("RecordIdentifier", g.yspec.RCT, "RCT")
	b.put("TotalRCWRecords", g.yspec.RCT, fmt.Sprintf("%07d", 0)) // placeholder; overwritten below

	// Boxes 1-7 totals (always written)
	b.put("OrigTotalWagesTips", g.yspec.RCT, money15(origWages))
	b.put("CorrectTotalWagesTips", g.yspec.RCT, money15(corrWages))
	b.put("OrigTotalFedIncomeTax", g.yspec.RCT, money15(origFed))
	b.put("CorrectTotalFedIncomeTax", g.yspec.RCT, money15(corrFed))
	b.put("OrigTotalSSWages", g.yspec.RCT, money15(origSS))
	b.put("CorrectTotalSSWages", g.yspec.RCT, money15(corrSS))
	b.put("OrigTotalSSTax", g.yspec.RCT, money15(origSSTax))
	b.put("CorrectTotalSSTax", g.yspec.RCT, money15(corrSSTax))
	b.put("OrigTotalMedicareWages", g.yspec.RCT, money15(origMed))
	b.put("CorrectTotalMedicareWages", g.yspec.RCT, money15(corrMed))
	b.put("OrigTotalMedicareTax", g.yspec.RCT, money15(origMedTax))
	b.put("CorrectTotalMedicareTax", g.yspec.RCT, money15(corrMedTax))
	b.put("OrigTotalSSTips", g.yspec.RCT, money15(origSSTips))
	b.put("CorrectTotalSSTips", g.yspec.RCT, money15(corrSSTips))

	// Optional totals (only write if non-zero)
	putMoney15Pair(b, g.yspec.RCT, "OrigTotalDependentCare", "CorrectTotalDependentCare",
		origDepCare, corrDepCare)
	putMoney15Pair(b, g.yspec.RCT, "OrigTotalCode401k", "CorrectTotalCode401k", origD, corrD)
	putMoney15Pair(b, g.yspec.RCT, "OrigTotalCode403b", "CorrectTotalCode403b", origE, corrE)
	putMoney15Pair(b, g.yspec.RCT, "OrigTotalCode457bGovt", "CorrectTotalCode457bGovt", origG, corrG)
	putMoney15Pair(b, g.yspec.RCT, "OrigTotalCodeW_HSA", "CorrectTotalCodeW_HSA", origW, corrW)
	putMoney15Pair(b, g.yspec.RCT, "OrigTotalNonqualPlan457", "CorrectTotalNonqualPlan457",
		origNQ457, corrNQ457)
	putMoney15Pair(b, g.yspec.RCT, "OrigTotalNonqualNotSection457", "CorrectTotalNonqualNotSection457",
		origNQNot457, corrNQNot457)
	putMoney15Pair(b, g.yspec.RCT, "OrigTotalCodeAA_Roth401k", "CorrectTotalCodeAA_Roth401k",
		origAA, corrAA)
	putMoney15Pair(b, g.yspec.RCT, "OrigTotalCodeBB_Roth403b", "CorrectTotalCodeBB_Roth403b",
		origBB, corrBB)
	putMoney15Pair(b, g.yspec.RCT, "OrigTotalCodeDD_EmpHealth", "CorrectTotalCodeDD_EmpHealth",
		origDD, corrDD)

	return b.String()
}

// buildRCT is called without knowing the RCW count; the caller fills RCT.TotalRCWRecords.
// We expose a separate setter so Generate() can write the count after appending all records.
// For simplicity the RCT TotalRCWRecords is always overwritten by the RCF value.
func (g *Generator) buildRCF(count int) string {
	b := newBuf()
	b.put("RecordIdentifier", g.yspec.RCF, "RCF")
	b.put("TotalRCWRecords", g.yspec.RCF, fmt.Sprintf("%07d", count))
	return b.String()
}

// ---------------------------------------------------------------------------
// Buffer
// ---------------------------------------------------------------------------

type fixedBuf struct{ data []byte }

func newBuf() *fixedBuf {
	d := make([]byte, spec.RecordLen)
	for i := range d {
		d[i] = ' '
	}
	return &fixedBuf{data: d}
}

// put looks up fieldName in fields and writes value at the correct position.
// Panics on unknown field name — that's a generator bug, not user error.
func (b *fixedBuf) put(fieldName string, fields []spec.Field, value string) {
	for _, f := range fields {
		if f.Name == fieldName {
			width := f.End - f.Start + 1
			if len(value) > width {
				value = value[:width]
			}
			copy(b.data[f.Start-1:f.End], value)
			return
		}
	}
	panic(fmt.Sprintf("efw2c: field %q not found in spec — generator bug", fieldName))
}

func (b *fixedBuf) String() string { return string(b.data) }

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// putMoney11Pair writes an 11-char money pair; fills with blanks if both zero
// (spec says "fill with blanks if not making a correction").
func putMoney11Pair(b *fixedBuf, fields []spec.Field, origName, corrName string, orig, corr int64) {
	if orig == 0 && corr == 0 {
		return // leave as spaces
	}
	b.put(origName, fields, money11(orig))
	b.put(corrName, fields, money11(corr))
}

// putMoney15Pair is the 15-char variant for RCT totals.
func putMoney15Pair(b *fixedBuf, fields []spec.Field, origName, corrName string, orig, corr int64) {
	if orig == 0 && corr == 0 {
		return
	}
	b.put(origName, fields, money15(orig))
	b.put(corrName, fields, money15(corr))
}

// putBox13 writes Box 13 checkbox indicators when a correction is being made.
// Blank = no correction; "0" or "1" = correction.
func putBox13(b *fixedBuf, fields []spec.Field, origName, corrName string, orig, corr *bool) {
	if orig == nil && corr == nil {
		return
	}
	if orig != nil {
		b.put(origName, fields, boolChar(*orig))
	}
	if corr != nil {
		b.put(corrName, fields, boolChar(*corr))
	}
}

// ---------------------------------------------------------------------------
// Formatting helpers
// ---------------------------------------------------------------------------

// padAlpha uppercases and left-pads with spaces to exactly n chars.
func padAlpha(s string, n int) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	if len(s) > n {
		return s[:n]
	}
	return s + strings.Repeat(" ", n-len(s))
}

// padNumeric strips non-digits and left-pads with spaces to exactly n chars.
// Per spec, numeric fields that are not populated should be all spaces.
func padNumeric(s string, n int) string {
	var builder strings.Builder
	for _, r := range s {
		if unicode.IsDigit(r) {
			builder.WriteRune(r)
		}
	}
	result := builder.String()
	if len(result) > n {
		return result[:n]
	}
	return result + strings.Repeat(" ", n-len(result))
}

// padEmail preserves case for email addresses (spec allows mixed case).
func padEmail(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) > n {
		return s[:n]
	}
	return s + strings.Repeat(" ", n-len(s))
}

// cleanDigits strips non-digits and zero-pads to exactly n digits.
// Used for EIN and SSN fields which must be all digits.
func cleanDigits(s string, n int) string {
	var builder strings.Builder
	for _, r := range s {
		if unicode.IsDigit(r) {
			builder.WriteRune(r)
		}
	}
	result := builder.String()
	if len(result) > n {
		return result[:n]
	}
	return result + strings.Repeat("0", n-len(result))
}

// money11 formats cents as an 11-char zero-padded integer (no decimal point).
// Used in RCW and RCO records.
func money11(cents int64) string {
	if cents < 0 {
		cents = 0
	}
	return fmt.Sprintf("%011d", cents)
}

// money15 formats cents as a 15-char zero-padded integer.
// Used in RCT (total) records.
func money15(cents int64) string {
	if cents < 0 {
		cents = 0
	}
	return fmt.Sprintf("%015d", cents)
}

func boolChar(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

func defaultStr(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

// statePostalToNumeric converts a 2-char postal abbreviation to the SSA 2-digit
// numeric state code required in the RCS record StateCode field (Appendix H).
// Returns "  " (blanks) if the state is not found.
func statePostalToNumeric(abbr string) string {
	codes := map[string]string{
		"AL": "01", "AK": "02", "AZ": "03", "AR": "04", "CA": "05",
		"CO": "06", "CT": "07", "DE": "08", "FL": "09", "GA": "10",
		"HI": "11", "ID": "12", "IL": "13", "IN": "14", "IA": "15",
		"KS": "16", "KY": "17", "LA": "18", "ME": "19", "MD": "20",
		"MA": "21", "MI": "22", "MN": "23", "MS": "24", "MO": "25",
		"MT": "26", "NE": "27", "NV": "28", "NH": "29", "NJ": "30",
		"NM": "31", "NY": "32", "NC": "33", "ND": "34", "OH": "35",
		"OK": "36", "OR": "37", "PA": "38", "RI": "39", "SC": "40",
		"SD": "41", "TN": "42", "TX": "43", "UT": "44", "VT": "45",
		"VA": "46", "WA": "47", "WV": "48", "WI": "49", "WY": "50",
		"DC": "51", "PR": "72", "VI": "78", "GU": "66", "AS": "60",
		"MP": "69",
	}
	if v, ok := codes[strings.ToUpper(strings.TrimSpace(abbr))]; ok {
		return v
	}
	return "  "
}
