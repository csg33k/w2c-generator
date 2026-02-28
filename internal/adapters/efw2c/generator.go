package efw2c

import (
	"context"
	"fmt"
	"io"
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

// Generate writes a complete EFW2C byte stream (no CR/LF between records).
func (g *Generator) Generate(ctx context.Context, s *domain.Submission, w io.Writer) error {
	records := []string{
		g.buildRCA(s),
		g.buildRCE(s),
	}

	var (
		origWages, corrWages   int64
		origFed, corrFed       int64
		origSS, corrSS         int64
		origSSTax, corrSSTax   int64
		origMed, corrMed       int64
		origMedTax, corrMedTax int64
	)
	for i := range s.Employees {
		e := &s.Employees[i]
		records = append(records, g.buildRCW(e))
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
	}
	records = append(records,
		g.buildRCT(origWages, corrWages, origFed, corrFed, origSS, corrSS,
			origSSTax, corrSSTax, origMed, corrMed, origMedTax, corrMedTax),
		g.buildRCF(len(s.Employees)),
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
	// SoftwareCode left blank — not a software vendor
	b.put("CompanyName", g.yspec.RCA, padAlpha(s.Employer.Name, 35))
	b.put("LocationAddress", g.yspec.RCA, padAlpha(s.Employer.AddressLine1, 40))
	b.put("DeliveryAddress", g.yspec.RCA, padAlpha(s.Employer.AddressLine2, 40))
	b.put("StateAbbrev", g.yspec.RCA, padAlpha(s.Employer.State, 2))
	b.put("ZIPCode", g.yspec.RCA, padNumeric(s.Employer.ZIP, 5))
	b.put("ZIPExtension", g.yspec.RCA, padNumeric(s.Employer.ZIPExtension, 4))
	// ForeignStateProvince left blank (domestic address)
	// CountryCode left blank (USA)
	b.put("ContactName", g.yspec.RCA, padAlpha(sub.ContactName, 27))
	b.put("ContactPhone", g.yspec.RCA, padNumeric(sub.ContactPhone, 15))
	b.put("ContactEmail", g.yspec.RCA, padEmail(sub.ContactEmail, 40))
	b.put("PreparerCode", g.yspec.RCA, preparerCode)
	b.put("ResubIndicator", g.yspec.RCA, resubIndicator)
	if sub.ResubWFID != "" {
		b.put("ResubWFID", g.yspec.RCA, padAlpha(sub.ResubWFID, 9))
	}
	return b.String()
}

func (g *Generator) buildRCE(s *domain.Submission) string {
	b := newBuf()
	b.put("RecordIdentifier", g.yspec.RCE, "RCE")
	b.put("TaxYear", g.yspec.RCE, s.Employer.TaxYear)
	b.put("EmployerEIN", g.yspec.RCE, cleanDigits(s.Employer.EIN, 9))
	b.put("AgentForEIN", g.yspec.RCE, padNumeric(s.Employer.AgentEIN, 9))
	b.put("AgentIndicatorCode", g.yspec.RCE, defaultStr(s.Employer.AgentIndicator, "0"))
	b.put("TerminatingBusiness", g.yspec.RCE, boolChar(s.Employer.TerminatingBusiness))
	b.put("EmploymentCode", g.yspec.RCE, "R")
	b.put("EmployerName", g.yspec.RCE, padAlpha(s.Employer.Name, 35))
	b.put("LocationAddress", g.yspec.RCE, padAlpha(s.Employer.AddressLine1, 40))
	b.put("DeliveryAddress", g.yspec.RCE, padAlpha(s.Employer.AddressLine2, 40))
	b.put("City", g.yspec.RCE, padAlpha(s.Employer.City, 39))
	b.put("StateAbbrev", g.yspec.RCE, padAlpha(s.Employer.State, 2))
	b.put("ZIPCode", g.yspec.RCE, padNumeric(s.Employer.ZIP, 5))
	b.put("ZIPExtension", g.yspec.RCE, padNumeric(s.Employer.ZIPExtension, 4))
	return b.String()
}

func (g *Generator) buildRCW(e *domain.EmployeeRecord) string {
	b := newBuf()
	b.put("RecordIdentifier", g.yspec.RCW, "RCW")
	b.put("OrigSSN", g.yspec.RCW, cleanDigits(e.SSN, 9))
	b.put("CorrectSSN", g.yspec.RCW, cleanDigits(e.OriginalSSN, 9))
	b.put("OrigLastName", g.yspec.RCW, padAlpha(e.LastName, 15))
	b.put("OrigFirstName", g.yspec.RCW, padAlpha(e.FirstName, 12))
	b.put("OrigMiddleName", g.yspec.RCW, firstChar(e.MiddleName))
	b.put("OrigSuffix", g.yspec.RCW, padAlpha(e.Suffix, 4))
	b.put("LocationAddress", g.yspec.RCW, padAlpha(e.AddressLine1, 39))
	b.put("DeliveryAddress", g.yspec.RCW, padAlpha(e.AddressLine2, 39))
	b.put("City", g.yspec.RCW, padAlpha(e.City, 39))
	b.put("StateAbbrev", g.yspec.RCW, padAlpha(e.State, 2))
	b.put("ZIPCode", g.yspec.RCW, padNumeric(e.ZIP, 5))
	b.put("ZIPExtension", g.yspec.RCW, padNumeric(e.ZIPExtension, 4))
	b.put("OrigWagesTipsOther", g.yspec.RCW, money(e.Amounts.OriginalWagesTipsOther))
	b.put("CorrectWagesTipsOther", g.yspec.RCW, money(e.Amounts.CorrectWagesTipsOther))
	b.put("OrigFedIncomeTax", g.yspec.RCW, money(e.Amounts.OriginalFederalIncomeTax))
	b.put("CorrectFedIncomeTax", g.yspec.RCW, money(e.Amounts.CorrectFederalIncomeTax))
	b.put("OrigSSWages", g.yspec.RCW, money(e.Amounts.OriginalSocialSecurityWages))
	b.put("CorrectSSWages", g.yspec.RCW, money(e.Amounts.CorrectSocialSecurityWages))
	b.put("OrigSSTax", g.yspec.RCW, money(e.Amounts.OriginalSocialSecurityTax))
	b.put("CorrectSSTax", g.yspec.RCW, money(e.Amounts.CorrectSocialSecurityTax))
	b.put("OrigMedicareWages", g.yspec.RCW, money(e.Amounts.OriginalMedicareWages))
	b.put("CorrectMedicareWages", g.yspec.RCW, money(e.Amounts.CorrectMedicareWages))
	b.put("OrigMedicareTax", g.yspec.RCW, money(e.Amounts.OriginalMedicareTax))
	b.put("CorrectMedicareTax", g.yspec.RCW, money(e.Amounts.CorrectMedicareTax))
	return b.String()
}

func (g *Generator) buildRCT(
	origWages, corrWages,
	origFed, corrFed,
	origSS, corrSS,
	origSSTax, corrSSTax,
	origMed, corrMed,
	origMedTax, corrMedTax int64,
) string {
	b := newBuf()
	b.put("RecordIdentifier", g.yspec.RCT, "RCT")
	b.put("OrigTotalWagesTips", g.yspec.RCT, money(origWages))
	b.put("CorrectTotalWagesTips", g.yspec.RCT, money(corrWages))
	b.put("OrigTotalFedIncomeTax", g.yspec.RCT, money(origFed))
	b.put("CorrectTotalFedIncomeTax", g.yspec.RCT, money(corrFed))
	b.put("OrigTotalSSWages", g.yspec.RCT, money(origSS))
	b.put("CorrectTotalSSWages", g.yspec.RCT, money(corrSS))
	b.put("OrigTotalSSTax", g.yspec.RCT, money(origSSTax))
	b.put("CorrectTotalSSTax", g.yspec.RCT, money(corrSSTax))
	b.put("OrigTotalMedicareWages", g.yspec.RCT, money(origMed))
	b.put("CorrectTotalMedicareWages", g.yspec.RCT, money(corrMed))
	b.put("OrigTotalMedicareTax", g.yspec.RCT, money(origMedTax))
	b.put("CorrectTotalMedicareTax", g.yspec.RCT, money(corrMedTax))
	return b.String()
}

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
			copy(b.data[f.Start-1:f.End], []byte(value))
			return
		}
	}
	panic(fmt.Sprintf("efw2c: field %q not found in spec — generator bug", fieldName))
}

func (b *fixedBuf) String() string { return string(b.data) }

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
	var b strings.Builder
	for _, r := range s {
		if unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	result := b.String()
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
	var b strings.Builder
	for _, r := range s {
		if unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	result := b.String()
	if len(result) > n {
		return result[:n]
	}
	return result + strings.Repeat("0", n-len(result))
}

// money formats cents as a 12-char zero-padded integer (no decimal point).
func money(cents int64) string {
	if cents < 0 {
		cents = 0
	}
	return fmt.Sprintf("%012d", cents)
}

func boolChar(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

func firstChar(s string) string {
	for _, r := range s {
		return string(unicode.ToUpper(r))
	}
	return " "
}

func defaultStr(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}
