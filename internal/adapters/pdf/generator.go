// Package pdf generates a human-readable W-2C correction PDF report.
// One page is produced per employee; each page shows the employer header,
// employee identity information, and a table comparing original vs. corrected
// amounts for every W-2C box.
package pdf

import (
	"fmt"
	"io"
	"strings"

	"github.com/go-pdf/fpdf"

	"github.com/csg33k/w2c-generator/internal/domain"
)

// GeneratePDF writes a multi-page PDF (one page per employee) to w.
func GeneratePDF(s *domain.Submission, w io.Writer) error {
	pdf := fpdf.New("P", "mm", "Letter", "")
	pdf.SetMargins(18, 18, 18)
	pdf.SetAutoPageBreak(true, 18)
	pdf.AliasNbPages("{nb}")

	for i := range s.Employees {
		pdf.AddPage()
		drawEmployeePage(pdf, s, &s.Employees[i])
	}

	return pdf.Output(w)
}

func drawEmployeePage(pdf *fpdf.Fpdf, s *domain.Submission, e *domain.EmployeeRecord) {
	pageW, pageH := pdf.GetPageSize()
	marginL, marginT, marginR, marginB := pdf.GetMargins()
	contentW := pageW - marginL - marginR

	// ── Header bar ───────────────────────────────────────────────────────────
	pdf.SetFillColor(30, 30, 30)
	pdf.Rect(marginL, marginT, contentW, 10, "F")
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Helvetica", "B", 11)
	pdf.SetXY(marginL+2, marginT+1.5)
	pdf.CellFormat(contentW-4, 7, "W-2C  WAGE AND TAX STATEMENT CORRECTION", "", 0, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 9)
	pdf.CellFormat(0, 7, "Page "+fmt.Sprint(pdf.PageNo())+" of {nb}", "", 1, "R", false, 0, "")
	pdf.SetTextColor(0, 0, 0)

	y := marginT + 13

	// ── Employer section ─────────────────────────────────────────────────────
	pdf.SetFillColor(240, 240, 240)
	pdf.SetFont("Helvetica", "B", 8)
	pdf.SetXY(marginL, y)
	pdf.CellFormat(contentW, 5.5, "EMPLOYER INFORMATION", "LRT", 1, "L", true, 0, "")
	y += 5.5

	pdf.SetFont("Helvetica", "", 9)
	pdf.SetXY(marginL, y)
	// Two-column employer layout
	colHalf := contentW / 2
	pdf.CellFormat(colHalf, 6, "Employer: "+s.Employer.Name, "L", 0, "L", false, 0, "")
	pdf.CellFormat(colHalf, 6, "EIN: "+formatEIN(s.Employer.EIN)+"   Tax Year: "+s.Employer.TaxYear, "R", 1, "L", false, 0, "")
	y += 6
	if s.Employer.AddressLine1 != "" {
		pdf.SetXY(marginL, y)
		pdf.CellFormat(contentW, 5.5, s.Employer.AddressLine1+cityLine(s.Employer.City, s.Employer.State, s.Employer.ZIP), "LB", 1, "L", false, 0, "")
		y += 5.5
	} else {
		// draw bottom border to close employer box
		pdf.SetXY(marginL, y)
		pdf.CellFormat(contentW, 0, "", "LB", 1, "L", false, 0, "")
	}

	y += 4

	// ── Employee section ─────────────────────────────────────────────────────
	pdf.SetFillColor(240, 240, 240)
	pdf.SetFont("Helvetica", "B", 8)
	pdf.SetXY(marginL, y)
	pdf.CellFormat(contentW, 5.5, "EMPLOYEE INFORMATION", "LRT", 1, "L", true, 0, "")
	y += 5.5

	name := e.LastName + ", " + e.FirstName
	if e.MiddleName != "" {
		name += " " + e.MiddleName
	}
	if e.Suffix != "" {
		name += " " + e.Suffix
	}

	pdf.SetFont("Helvetica", "B", 10)
	pdf.SetXY(marginL, y)
	pdf.CellFormat(colHalf, 6.5, name, "L", 0, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 9)
	pdf.CellFormat(colHalf, 6.5, "SSN: "+formatSSN(e.SSN), "R", 1, "R", false, 0, "")
	y += 6.5

	if e.OriginalSSN != "" {
		pdf.SetFont("Helvetica", "I", 8.5)
		pdf.SetXY(marginL, y)
		pdf.CellFormat(contentW, 5.5, "Original SSN: "+formatSSN(e.OriginalSSN), "LR", 1, "L", false, 0, "")
		y += 5.5
	}

	pdf.SetFont("Helvetica", "", 9)
	if e.AddressLine1 != "" {
		pdf.SetXY(marginL, y)
		pdf.CellFormat(contentW, 5.5, e.AddressLine1, "LR", 1, "L", false, 0, "")
		y += 5.5
	}
	if e.AddressLine2 != "" {
		pdf.SetXY(marginL, y)
		pdf.CellFormat(contentW, 5.5, e.AddressLine2, "LR", 1, "L", false, 0, "")
		y += 5.5
	}
	addrLine := cityLine(e.City, e.State, e.ZIP)
	if addrLine != "" {
		pdf.SetXY(marginL, y)
		pdf.CellFormat(contentW, 5.5, strings.TrimPrefix(addrLine, ", "), "LB", 1, "L", false, 0, "")
		y += 5.5
	} else {
		pdf.SetXY(marginL, y)
		pdf.CellFormat(contentW, 0, "", "LB", 1, "L", false, 0, "")
	}

	y += 5

	// ── Corrections table ─────────────────────────────────────────────────────
	descW := contentW * 0.52
	origW := (contentW - descW) / 2
	corrW := contentW - descW - origW

	// Table header
	pdf.SetFillColor(30, 30, 30)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Helvetica", "B", 8.5)
	pdf.SetXY(marginL, y)
	pdf.CellFormat(descW, 7, "Description", "1", 0, "L", true, 0, "")
	pdf.CellFormat(origW, 7, "Original Amount", "1", 0, "C", true, 0, "")
	pdf.CellFormat(corrW, 7, "Corrected Amount", "1", 1, "C", true, 0, "")
	y += 7
	pdf.SetTextColor(0, 0, 0)

	type amtRow struct {
		label string
		orig  int64
		corr  int64
	}

	rows := []amtRow{
		{"Box 1 - Wages, Tips, Other Comp.", e.Amounts.OriginalWagesTipsOther, e.Amounts.CorrectWagesTipsOther},
		{"Box 2 - Federal Income Tax Withheld", e.Amounts.OriginalFederalIncomeTax, e.Amounts.CorrectFederalIncomeTax},
		{"Box 3 - Social Security Wages", e.Amounts.OriginalSocialSecurityWages, e.Amounts.CorrectSocialSecurityWages},
		{"Box 4 - Social Security Tax Withheld", e.Amounts.OriginalSocialSecurityTax, e.Amounts.CorrectSocialSecurityTax},
		{"Box 5 - Medicare Wages and Tips", e.Amounts.OriginalMedicareWages, e.Amounts.CorrectMedicareWages},
		{"Box 6 - Medicare Tax Withheld", e.Amounts.OriginalMedicareTax, e.Amounts.CorrectMedicareTax},
		{"Box 7 - Social Security Tips", e.Amounts.OriginalSocialSecurityTips, e.Amounts.CorrectSocialSecurityTips},
	}

	// Optional state/local rows - only include when non-zero
	if e.Amounts.OriginalStateWages != 0 || e.Amounts.CorrectStateWages != 0 {
		rows = append(rows, amtRow{"Box 16 - State Wages, Tips, etc.", e.Amounts.OriginalStateWages, e.Amounts.CorrectStateWages})
	}
	if e.Amounts.OriginalStateIncomeTax != 0 || e.Amounts.CorrectStateIncomeTax != 0 {
		rows = append(rows, amtRow{"Box 17 - State Income Tax", e.Amounts.OriginalStateIncomeTax, e.Amounts.CorrectStateIncomeTax})
	}
	if e.Amounts.OriginalLocalWages != 0 || e.Amounts.CorrectLocalWages != 0 {
		rows = append(rows, amtRow{"Box 18 - Local Wages, Tips, etc.", e.Amounts.OriginalLocalWages, e.Amounts.CorrectLocalWages})
	}
	if e.Amounts.OriginalLocalIncomeTax != 0 || e.Amounts.CorrectLocalIncomeTax != 0 {
		rows = append(rows, amtRow{"Box 19 - Local Income Tax", e.Amounts.OriginalLocalIncomeTax, e.Amounts.CorrectLocalIncomeTax})
	}

	rowH := 6.5
	for i, r := range rows {
		pdf.SetXY(marginL, y)
		changed := r.orig != r.corr
		// Alternating row background
		if i%2 == 0 {
			pdf.SetFillColor(250, 250, 250)
		} else {
			pdf.SetFillColor(255, 255, 255)
		}
		if changed {
			pdf.SetFont("Helvetica", "B", 8.5)
		} else {
			pdf.SetFont("Helvetica", "", 8.5)
		}
		pdf.CellFormat(descW, rowH, r.label, "1", 0, "L", true, 0, "")
		pdf.CellFormat(origW, rowH, "$"+centsToDisplay(r.orig), "1", 0, "R", true, 0, "")

		// Highlight changed correction values in a subtle way
		if changed {
			pdf.SetFillColor(220, 240, 220) // light green for corrected
		}
		pdf.CellFormat(corrW, rowH, "$"+centsToDisplay(r.corr), "1", 1, "R", true, 0, "")
		if changed {
			// restore alternating fill for next iteration
			if i%2 == 0 {
				pdf.SetFillColor(250, 250, 250)
			} else {
				pdf.SetFillColor(255, 255, 255)
			}
		}
		y += rowH
	}

	// ── State / Locality block (Box 15 & 20) ─────────────────────────────────
	hasStateLocality := e.OriginalStateCode != "" || e.CorrectStateCode != "" ||
		e.OriginalStateIDNumber != "" || e.CorrectStateIDNumber != "" ||
		e.OriginalLocalityName != "" || e.CorrectLocalityName != ""

	if hasStateLocality {
		y += 5
		pdf.SetFillColor(240, 240, 240)
		pdf.SetFont("Helvetica", "B", 8)
		pdf.SetXY(marginL, y)
		pdf.CellFormat(contentW, 5.5, "BOX 15 / BOX 20 - STATE & LOCALITY", "LRT", 1, "L", true, 0, "")
		y += 5.5

		pdf.SetFont("Helvetica", "", 8.5)

		if e.OriginalStateCode != "" || e.CorrectStateCode != "" {
			pdf.SetXY(marginL, y)
			pdf.CellFormat(contentW/3, 5.5, "State Code:", "L", 0, "L", false, 0, "")
			pdf.CellFormat(contentW/3, 5.5, e.OriginalStateCode, "", 0, "C", false, 0, "")
			pdf.CellFormat(contentW/3, 5.5, "->  "+e.CorrectStateCode, "R", 1, "L", false, 0, "")
			y += 5.5
		}
		if e.OriginalStateIDNumber != "" || e.CorrectStateIDNumber != "" {
			pdf.SetXY(marginL, y)
			pdf.CellFormat(contentW/3, 5.5, "State ID Number:", "L", 0, "L", false, 0, "")
			pdf.CellFormat(contentW/3, 5.5, e.OriginalStateIDNumber, "", 0, "C", false, 0, "")
			pdf.CellFormat(contentW/3, 5.5, "->  "+e.CorrectStateIDNumber, "R", 1, "L", false, 0, "")
			y += 5.5
		}
		if e.OriginalLocalityName != "" || e.CorrectLocalityName != "" {
			pdf.SetXY(marginL, y)
			pdf.CellFormat(contentW/3, 5.5, "Locality Name:", "L", 0, "L", false, 0, "")
			pdf.CellFormat(contentW/3, 5.5, e.OriginalLocalityName, "", 0, "C", false, 0, "")
			pdf.CellFormat(contentW/3, 5.5, "->  "+e.CorrectLocalityName, "R", 1, "L", false, 0, "")
			y += 5.5
		}
		// close box
		pdf.SetXY(marginL, y)
		pdf.CellFormat(contentW, 0, "", "LB", 1, "L", false, 0, "")
	}

	// ── Footer ─────────────────────────────────────────────────────────────────
	pdf.SetXY(marginL, pageH-marginB-6)
	pdf.SetFont("Helvetica", "I", 7.5)
	pdf.SetTextColor(130, 130, 130)
	pdf.CellFormat(contentW/2, 5, "Generated by W-2C Generator", "", 0, "L", false, 0, "")
	pdf.CellFormat(contentW/2, 5, s.Employer.Name+" | EIN "+formatEIN(s.Employer.EIN)+" | TY "+s.Employer.TaxYear, "", 0, "R", false, 0, "")
	pdf.SetTextColor(0, 0, 0)
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func formatSSN(ssn string) string {
	digits := strings.ReplaceAll(ssn, "-", "")
	if len(digits) == 9 {
		return digits[:3] + "-" + digits[3:5] + "-" + digits[5:]
	}
	return ssn
}

func formatEIN(ein string) string {
	digits := strings.ReplaceAll(ein, "-", "")
	if len(digits) == 9 {
		return digits[:2] + "-" + digits[2:]
	}
	return ein
}

func centsToDisplay(cents int64) string {
	return fmt.Sprintf("%.2f", float64(cents)/100)
}

// cityLine returns ", City, ST ZIP" ready to append to an address, or "".
func cityLine(city, state, zip string) string {
	if city == "" && state == "" && zip == "" {
		return ""
	}
	s := ""
	if city != "" {
		s += ", " + city
	}
	if state != "" {
		if city != "" {
			s += ", " + state
		} else {
			s += ", " + state
		}
	}
	if zip != "" {
		s += " " + zip
	}
	return s
}
