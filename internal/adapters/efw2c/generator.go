package efw2c

import (
	"context"
	"fmt"
	"io"
	"strings"
	"unicode"

	"github.com/csg33k/w2c-generator/internal/domain"
)

// Generator implements ports.EFW2CGenerator for the 2021 EFW2C format.
// SSA Publication 42-007 EFW2C Tax Year 2021.
type Generator struct{}

func New() *Generator { return &Generator{} }

// Generate writes the full EFW2C submission file:
//
//	RCA  - Submitter Record
//	RCE  - Employer Record
//	RCW  - Employee Correction Record (one per employee)
//	RCT  - Total Record
//	RCF  - Final Record
func (g *Generator) Generate(ctx context.Context, s *domain.Submission, w io.Writer) error {
	lines := []string{}
	lines = append(lines, g.buildRCA(s))
	lines = append(lines, g.buildRCE(s))
	var totalFedTax, totalSSWages, totalMedWages int64
	for i := range s.Employees {
		lines = append(lines, g.buildRCW(&s.Employees[i]))
		totalFedTax += s.Employees[i].Amounts.CorrectFederalIncomeTax
		totalSSWages += s.Employees[i].Amounts.CorrectSocialSecurityWages
		totalMedWages += s.Employees[i].Amounts.CorrectMedicareWages
	}
	lines = append(lines, g.buildRCT(s, totalFedTax, totalSSWages, totalMedWages))
	lines = append(lines, g.buildRCF(len(s.Employees)))

	for _, l := range lines {
		if _, err := fmt.Fprintln(w, l); err != nil {
			return err
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Record builders
// ---------------------------------------------------------------------------

// RCA – Submitter Record (positions 1-512)
func (g *Generator) buildRCA(s *domain.Submission) string {
	b := newBuf(512)
	b.put(1, 3, "RCA")
	b.put(4, 8, "2021")                     // Tax year
	b.put(9, 9, "1")                        // Software vendor code placeholder
	b.put(10, 18, clean(s.Employer.EIN, 9)) // Submitter EIN
	b.put(19, 57, pad(s.Employer.Name, 39))
	b.put(58, 96, pad(s.Employer.AddressLine1, 39))
	b.put(97, 135, pad(s.Employer.City, 39))
	b.put(136, 137, pad(s.Employer.State, 2))
	b.put(138, 146, pad(s.Employer.ZIP+s.Employer.ZIPExtension, 9))
	b.put(147, 149, "USA")
	b.put(150, 512, spaces(363))
	return b.String()
}

// RCE – Employer Record (positions 1-512)
func (g *Generator) buildRCE(s *domain.Submission) string {
	b := newBuf(512)
	b.put(1, 3, "RCE")
	b.put(4, 4, "1") // Tax year indicator (1 = current)
	b.put(5, 13, clean(s.Employer.EIN, 9))
	b.put(14, 14, s.Employer.AgentIndicator)
	b.put(15, 23, pad(s.Employer.AgentEIN, 9))
	b.put(24, 24, boolChar(s.Employer.TerminatingBusiness))
	b.put(25, 63, pad(s.Employer.Name, 39))
	b.put(64, 102, pad(s.Employer.AddressLine1, 39))
	b.put(103, 141, pad(s.Employer.AddressLine2, 39))
	b.put(142, 180, pad(s.Employer.City, 39))
	b.put(181, 182, pad(s.Employer.State, 2))
	b.put(183, 187, pad(s.Employer.ZIP, 5))
	b.put(188, 191, pad(s.Employer.ZIPExtension, 4))
	b.put(192, 512, spaces(321))
	return b.String()
}

// RCW – Employee Correction Record
func (g *Generator) buildRCW(e *domain.EmployeeRecord) string {
	b := newBuf(512)
	b.put(1, 3, "RCW")
	b.put(4, 12, clean(e.SSN, 9))
	b.put(13, 21, pad(clean(e.OriginalSSN, 9), 9))
	b.put(22, 36, padLeft(e.LastName, 15))
	b.put(37, 48, padLeft(e.FirstName, 12))
	b.put(49, 49, firstChar(e.MiddleName))
	b.put(50, 53, pad(e.Suffix, 4))
	b.put(54, 92, pad(e.AddressLine1, 39))
	b.put(93, 131, pad(e.AddressLine2, 39))
	b.put(132, 170, pad(e.City, 39))
	b.put(171, 172, pad(e.State, 2))
	b.put(173, 177, pad(e.ZIP, 5))
	b.put(178, 181, pad(e.ZIPExtension, 4))
	b.put(182, 182, "U") // USA
	// Box 1: Wages, tips
	b.put(183, 194, money(e.Amounts.OriginalWagesTipsOther))
	b.put(195, 206, money(e.Amounts.CorrectWagesTipsOther))
	// Box 2: Federal income tax
	b.put(207, 218, money(e.Amounts.OriginalFederalIncomeTax))
	b.put(219, 230, money(e.Amounts.CorrectFederalIncomeTax))
	// Box 3: SS wages
	b.put(231, 242, money(e.Amounts.OriginalSocialSecurityWages))
	b.put(243, 254, money(e.Amounts.CorrectSocialSecurityWages))
	// Box 4: SS tax
	b.put(255, 266, money(e.Amounts.OriginalSocialSecurityTax))
	b.put(267, 278, money(e.Amounts.CorrectSocialSecurityTax))
	// Box 5: Medicare wages
	b.put(279, 290, money(e.Amounts.OriginalMedicareWages))
	b.put(291, 302, money(e.Amounts.CorrectMedicareWages))
	// Box 6: Medicare tax
	b.put(303, 314, money(e.Amounts.OriginalMedicareTax))
	b.put(315, 326, money(e.Amounts.CorrectMedicareTax))
	b.put(327, 512, spaces(186))
	return b.String()
}

// RCT – Total Record
func (g *Generator) buildRCT(s *domain.Submission, fedTax, ssWages, medWages int64) string {
	b := newBuf(512)
	b.put(1, 3, "RCT")
	b.put(4, 15, money(fedTax))
	b.put(16, 27, money(ssWages))
	b.put(28, 39, money(medWages))
	b.put(40, 512, spaces(473))
	return b.String()
}

// RCF – Final Record
func (g *Generator) buildRCF(employeeCount int) string {
	b := newBuf(512)
	b.put(1, 3, "RCF")
	b.put(4, 10, fmt.Sprintf("%07d", employeeCount))
	b.put(11, 512, spaces(502))
	return b.String()
}

// ---------------------------------------------------------------------------
// Buffer helpers
// ---------------------------------------------------------------------------

type fixedBuf struct {
	data []byte
}

func newBuf(size int) *fixedBuf {
	d := make([]byte, size)
	for i := range d {
		d[i] = ' '
	}
	return &fixedBuf{data: d}
}

// put places s at 1-based positions [start, end] (inclusive), left-aligned.
func (f *fixedBuf) put(start, end int, s string) {
	width := end - start + 1
	if len(s) > width {
		s = s[:width]
	}
	copy(f.data[start-1:end], []byte(s))
}

func (f *fixedBuf) String() string { return string(f.data) }

// ---------------------------------------------------------------------------
// Formatting helpers
// ---------------------------------------------------------------------------

func pad(s string, n int) string {
	s = strings.ToUpper(s)
	if len(s) >= n {
		return s[:n]
	}
	return s + strings.Repeat(" ", n-len(s))
}

func padLeft(s string, n int) string {
	return pad(s, n)
}

func spaces(n int) string { return strings.Repeat(" ", n) }

func clean(s string, n int) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	result := b.String()
	if len(result) >= n {
		return result[:n]
	}
	return result + strings.Repeat("0", n-len(result))
}

// money formats cents as a 12-char zero-padded string (no decimal).
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
		return string(r)
	}
	return " "
}
