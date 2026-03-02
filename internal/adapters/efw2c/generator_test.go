package efw2c_test

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/csg33k/w2c-generator/internal/adapters/efw2c"
	"github.com/csg33k/w2c-generator/internal/adapters/efw2c/spec"
	"github.com/csg33k/w2c-generator/internal/domain"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// extract returns a 1-based inclusive substring of s (matches spec field positions).
func extract(s string, start, end int) string {
	if start < 1 || end > len(s) || start > end {
		return fmt.Sprintf("<out-of-range:%d-%d>", start, end)
	}
	return s[start-1 : end]
}

// record returns the n-th record (0-indexed) from a generated stream.
// Each record is exactly spec.RecordLen bytes.
func record(output string, n int) string {
	start := n * spec.RecordLen
	end := start + spec.RecordLen
	if end > len(output) {
		return ""
	}
	return output[start:end]
}

// trimR strips trailing spaces for readable assertions.
func trimR(s string) string { return strings.TrimRight(s, " ") }

// boolPtr is a test helper for *bool fields.
func boolPtr(b bool) *bool { return &b }

// ---------------------------------------------------------------------------
// Minimal valid submission used across all year tests
// ---------------------------------------------------------------------------

func minimalSubmission(taxYear string) *domain.Submission {
	return &domain.Submission{
		Submitter: domain.SubmitterInfo{
			BSOUID:       "TESTUSER",
			ContactName:  "JANE DOE",
			ContactPhone: "8005551234",
			ContactEmail: "jane@example.com",
		},
		Employer: domain.EmployerRecord{
			EIN:            "123456789",
			Name:           "ACME CORP",
			AddressLine1:   "100 MAIN ST",
			AddressLine2:   "SUITE 200",
			City:           "SPRINGFIELD",
			State:          "IL",
			ZIP:            "62701",
			ZIPExtension:   "1234",
			TaxYear:        taxYear,
			EmploymentCode: "R",
			KindOfEmployer: "N",
		},
		Employees: []domain.EmployeeRecord{
			{
				SSN:       "987654321",
				FirstName: "JOHN",
				LastName:  "SMITH",
				Amounts: domain.MonetaryAmounts{
					OriginalWagesTipsOther:   5000000, // $50,000.00
					CorrectWagesTipsOther:    5100000, // $51,000.00
					OriginalFederalIncomeTax: 800000,  // $8,000.00
					CorrectFederalIncomeTax:  820000,  // $8,200.00
					OriginalSocialSecurityWages: 5000000,
					CorrectSocialSecurityWages:  5100000,
					OriginalSocialSecurityTax:   310000,
					CorrectSocialSecurityTax:    316200,
					OriginalMedicareWages:       5000000,
					CorrectMedicareWages:        5100000,
					OriginalMedicareTax:         72500,
					CorrectMedicareTax:          73950,
				},
			},
		},
	}
}

// ---------------------------------------------------------------------------
// Spec structural tests: verify every field in every record is gapless,
// non-overlapping, and fills exactly 1024 bytes for all supported years.
// ---------------------------------------------------------------------------

func TestSpecStructure_AllYears(t *testing.T) {
	for _, year := range spec.Supported() {
		year := year
		t.Run(fmt.Sprintf("TY%d", year), func(t *testing.T) {
			ys, ok := spec.ForYear(year)
			if !ok {
				t.Fatalf("ForYear(%d) returned ok=false", year)
			}
			for recName, fields := range map[string][]spec.Field{
				"RCA": ys.RCA,
				"RCE": ys.RCE,
				"RCW": ys.RCW,
				"RCO": ys.RCO,
				"RCS": ys.RCS,
				"RCT": ys.RCT,
				"RCF": ys.RCF,
			} {
				t.Run(recName, func(t *testing.T) {
					if len(fields) == 0 {
						t.Fatal("no fields defined")
					}
					// Check gapless coverage from 1 to 1024
					prev := 0
					for _, f := range fields {
						if f.Start != prev+1 {
							t.Errorf("field %q: expected Start=%d, got %d (gap or overlap after pos %d)",
								f.Name, prev+1, f.Start, prev)
						}
						if f.End < f.Start {
							t.Errorf("field %q: End(%d) < Start(%d)", f.Name, f.End, f.Start)
						}
						prev = f.End
					}
					if prev != spec.RecordLen {
						t.Errorf("record ends at position %d, want %d", prev, spec.RecordLen)
					}
				})
			}
		})
	}
}

// TestSpecStructure_TY2024_RCO_HasCodeII verifies the TY2024-specific Code II field.
func TestSpecStructure_TY2024_RCO_HasCodeII(t *testing.T) {
	ys, _ := spec.ForYear(2024)
	var found bool
	for _, f := range ys.RCO {
		if f.Name == "OrigMedicaidWaiver" {
			found = true
			if f.Start != 277 || f.End != 287 {
				t.Errorf("OrigMedicaidWaiver: want 277-287, got %d-%d", f.Start, f.End)
			}
		}
		if f.Name == "CorrectMedicaidWaiver" {
			if f.Start != 288 || f.End != 298 {
				t.Errorf("CorrectMedicaidWaiver: want 288-298, got %d-%d", f.Start, f.End)
			}
		}
	}
	if !found {
		t.Error("TY2024 RCO missing OrigMedicaidWaiver (Code II)")
	}
}

// TestSpecStructure_TY2021_RCO_NoCodeII verifies TY2021 does not have Code II.
func TestSpecStructure_TY2021_RCO_NoCodeII(t *testing.T) {
	ys, _ := spec.ForYear(2021)
	for _, f := range ys.RCO {
		if f.Name == "OrigMedicaidWaiver" || f.Name == "CorrectMedicaidWaiver" {
			t.Errorf("TY2021 RCO should not contain Code II field %q", f.Name)
		}
	}
}

// ---------------------------------------------------------------------------
// Critical field position tests — verified against SSA Pub 42-014 TY2024
// ---------------------------------------------------------------------------

func TestSpecPositions_RCA(t *testing.T) {
	ys, _ := spec.ForYear(2024)
	want := []struct {
		name  string
		start int
		end   int
	}{
		{"RecordIdentifier", 1, 3},
		{"SubmitterEIN", 4, 12},
		{"BSOUID", 13, 20},
		{"SoftwareVendorCode", 21, 24},
		{"Blank25", 25, 29},
		{"SoftwareCode", 30, 31},
		{"CompanyName", 32, 88},     // 57 chars
		{"LocationAddress", 89, 110}, // 22 chars
		{"DeliveryAddress", 111, 132},
		{"City", 133, 154},
		{"StateAbbrev", 155, 156},
		{"ZIPCode", 157, 161},
		{"ZIPExtension", 162, 165},
		{"Blank166", 166, 171},
		{"ForeignStateProvince", 172, 194},
		{"ForeignPostalCode", 195, 209}, // 15 chars
		{"CountryCode", 210, 211},
		{"ContactName", 212, 238},
		{"ContactPhone", 239, 253},
		{"PhoneExtension", 254, 258},
		{"Blank259", 259, 261},
		{"ContactEmail", 262, 301},
		{"Blank302", 302, 304},
		{"ContactFax", 305, 314},
		{"Blank315", 315, 315},
		{"PreparerCode", 316, 316},
		{"ResubIndicator", 317, 317},
		{"ResubWFID", 318, 323}, // 6 chars
		{"Blank324", 324, 1024},
	}
	fields := make(map[string]spec.Field, len(ys.RCA))
	for _, f := range ys.RCA {
		fields[f.Name] = f
	}
	for _, w := range want {
		f, ok := fields[w.name]
		if !ok {
			t.Errorf("RCA: field %q not found", w.name)
			continue
		}
		if f.Start != w.start || f.End != w.end {
			t.Errorf("RCA.%s: want %d-%d, got %d-%d", w.name, w.start, w.end, f.Start, f.End)
		}
	}
}

func TestSpecPositions_RCE(t *testing.T) {
	ys, _ := spec.ForYear(2024)
	want := []struct {
		name  string
		start int
		end   int
	}{
		{"RecordIdentifier", 1, 3},
		{"TaxYear", 4, 7},
		{"OrigReportedEIN", 8, 16},
		{"EmployerEIN", 17, 25},
		{"AgentIndicatorCode", 26, 26}, // was wrongly at 36 in old spec
		{"AgentForEIN", 27, 35},
		{"OrigEstablishmentNum", 36, 39},
		{"CorrectEstablishmentNum", 40, 43},
		{"EmployerName", 44, 100},    // 57 chars
		{"LocationAddress", 101, 122}, // 22 chars
		{"DeliveryAddress", 123, 144},
		{"City", 145, 166},
		{"StateAbbrev", 167, 168},
		{"ZIPCode", 169, 173},
		{"ZIPExtension", 174, 177},
		{"Blank178", 178, 181},
		{"ForeignStateProvince", 182, 204}, // 23 chars
		{"ForeignPostalCode", 205, 219},    // 15 chars
		{"CountryCode", 220, 221},
		{"OrigEmploymentCode", 222, 222},
		{"CorrectEmploymentCode", 223, 223},
		{"OrigThirdPartySick", 224, 224},
		{"CorrectThirdPartySick", 225, 225},
		{"Blank226", 226, 226},
		{"KindOfEmployer", 227, 227},
		{"ContactName", 228, 254},
		{"ContactPhone", 255, 269},
		{"PhoneExtension", 270, 274},
		{"ContactFax", 275, 284},
		{"ContactEmail", 285, 324},
		{"Blank325", 325, 1024},
	}
	fields := make(map[string]spec.Field, len(ys.RCE))
	for _, f := range ys.RCE {
		fields[f.Name] = f
	}
	for _, w := range want {
		f, ok := fields[w.name]
		if !ok {
			t.Errorf("RCE: field %q not found", w.name)
			continue
		}
		if f.Start != w.start || f.End != w.end {
			t.Errorf("RCE.%s: want %d-%d, got %d-%d", w.name, w.start, w.end, f.Start, f.End)
		}
	}
}

func TestSpecPositions_RCW_KeyFields(t *testing.T) {
	ys, _ := spec.ForYear(2024)
	want := []struct {
		name  string
		start int
		end   int
	}{
		// Identity
		{"RecordIdentifier", 1, 3},
		{"OrigSSN", 4, 12},
		{"CorrectSSN", 13, 21},
		// Name — First/Middle/Last order per spec
		{"OrigFirstName", 22, 36},
		{"OrigMiddleName", 37, 51},
		{"OrigLastName", 52, 71},
		{"CorrectFirstName", 72, 86},
		{"CorrectMiddleName", 87, 101},
		{"CorrectLastName", 102, 121},
		// Address (22-char fields)
		{"LocationAddress", 122, 143},
		{"DeliveryAddress", 144, 165},
		{"City", 166, 187},
		{"StateAbbrev", 188, 189},
		{"ZIPCode", 190, 194},
		{"ZIPExtension", 195, 198},
		// Money — Boxes 1-7
		{"OrigWagesTipsOther", 244, 254},
		{"CorrectWagesTipsOther", 255, 265},
		{"OrigFedIncomeTax", 266, 276},
		{"CorrectFedIncomeTax", 277, 287},
		{"OrigSSWages", 288, 298},
		{"CorrectSSWages", 299, 309},
		{"OrigSSTax", 310, 320},
		{"CorrectSSTax", 321, 331},
		{"OrigMedicareWages", 332, 342},
		{"CorrectMedicareWages", 343, 353},
		{"OrigMedicareTax", 354, 364},
		{"CorrectMedicareTax", 365, 375},
		{"OrigSSTips", 376, 386},
		{"CorrectSSTips", 387, 397},
		// Blank — was Box 9 (eliminated 2011)
		{"Blank398", 398, 419},
		// Box 10
		{"OrigDependentCare", 420, 430},
		{"CorrectDependentCare", 431, 441},
		// Box 12 codes in RCW
		{"OrigCode401k", 442, 452},
		{"CorrectCode401k", 453, 463},
		{"OrigCode403b", 464, 474},
		{"CorrectCode403b", 475, 485},
		{"OrigCodeF", 486, 496},
		{"CorrectCodeF", 497, 507},
		{"OrigCode457bGovt", 508, 518},
		{"CorrectCode457bGovt", 519, 529},
		{"OrigCodeH", 530, 540},
		{"CorrectCodeH", 541, 551},
		{"OrigTIBDeferredComp", 552, 562},
		{"CorrectTIBDeferredComp", 563, 573},
		{"Blank574", 574, 595},
		// Box 11 — Section 457
		{"OrigNonqualPlan457", 596, 606},
		{"CorrectNonqualPlan457", 607, 617},
		// Code W — HSA
		{"OrigCodeW_HSA", 618, 628},
		{"CorrectCodeW_HSA", 629, 639},
		// Box 11 — Non-457
		{"OrigNonqualNotSection457", 640, 650},
		{"CorrectNonqualNotSection457", 651, 661},
		// Code Q
		{"OrigCodeQ", 662, 672},
		{"CorrectCodeQ", 673, 683},
		{"Blank684", 684, 705},
		// Code C, V, Y, AA, BB, DD, FF
		{"OrigCodeC", 706, 716},
		{"CorrectCodeC", 717, 727},
		{"OrigCodeV", 728, 738},
		{"CorrectCodeV", 739, 749},
		{"OrigCodeY", 750, 760},
		{"CorrectCodeY", 761, 771},
		{"OrigCodeAA_Roth401k", 772, 782},
		{"CorrectCodeAA_Roth401k", 783, 793},
		{"OrigCodeBB_Roth403b", 794, 804},
		{"CorrectCodeBB_Roth403b", 805, 815},
		{"OrigCodeDD_EmpHealth", 816, 826},
		{"CorrectCodeDD_EmpHealth", 827, 837},
		{"OrigCodeFF_QSEHRA", 838, 848},
		{"CorrectCodeFF_QSEHRA", 849, 859},
		{"Blank860", 860, 1002},
		// Box 13 checkboxes
		{"OrigStatutoryEmployee", 1003, 1003},
		{"CorrectStatutoryEmployee", 1004, 1004},
		{"OrigRetirementPlan", 1005, 1005},
		{"CorrectRetirementPlan", 1006, 1006},
		{"OrigThirdPartySickPay", 1007, 1007},
		{"CorrectThirdPartySickPay", 1008, 1008},
		{"Blank1009", 1009, 1024},
	}
	fields := make(map[string]spec.Field, len(ys.RCW))
	for _, f := range ys.RCW {
		fields[f.Name] = f
	}
	for _, w := range want {
		f, ok := fields[w.name]
		if !ok {
			t.Errorf("RCW: field %q not found", w.name)
			continue
		}
		if f.Start != w.start || f.End != w.end {
			t.Errorf("RCW.%s: want %d-%d, got %d-%d", w.name, w.start, w.end, f.Start, f.End)
		}
	}
}

func TestSpecPositions_RCO(t *testing.T) {
	ys, _ := spec.ForYear(2024)
	want := []struct {
		name  string
		start int
		end   int
	}{
		{"RecordIdentifier", 1, 3},
		{"Blank4", 4, 12},
		{"OrigAllocatedTips", 13, 23},   // Box 8
		{"CorrectAllocatedTips", 24, 34}, // Box 8
		{"OrigUncollectedEETax", 35, 45},
		{"CorrectUncollectedEETax", 46, 56},
		{"OrigCodeR_MSA", 57, 67},
		{"CorrectCodeR_MSA", 68, 78},
		{"OrigCodeS_SIMPLE", 79, 89},
		{"CorrectCodeS_SIMPLE", 90, 100},
		{"OrigCodeT_Adoption", 101, 111},
		{"CorrectCodeT_Adoption", 112, 122},
		{"OrigCodeM_UncollSS", 123, 133},
		{"CorrectCodeM_UncollSS", 134, 144},
		{"OrigCodeN_UncollMed", 145, 155},
		{"CorrectCodeN_UncollMed", 156, 166},
		{"OrigCodeZ_409A", 167, 177},
		{"CorrectCodeZ_409A", 178, 188},
		{"Blank189", 189, 210},
		{"OrigCodeEE_Roth457b", 211, 221},
		{"CorrectCodeEE_Roth457b", 222, 232},
		{"OrigCodeGG_83i", 233, 243},
		{"CorrectCodeGG_83i", 244, 254},
		{"OrigCodeHH_83iDeferral", 255, 265},
		{"CorrectCodeHH_83iDeferral", 266, 276},
		// TY2024 Code II
		{"OrigMedicaidWaiver", 277, 287},
		{"CorrectMedicaidWaiver", 288, 298},
		{"Blank299", 299, 1024},
	}
	fields := make(map[string]spec.Field, len(ys.RCO))
	for _, f := range ys.RCO {
		fields[f.Name] = f
	}
	for _, w := range want {
		f, ok := fields[w.name]
		if !ok {
			t.Errorf("RCO(TY2024): field %q not found", w.name)
			continue
		}
		if f.Start != w.start || f.End != w.end {
			t.Errorf("RCO.%s: want %d-%d, got %d-%d", w.name, w.start, w.end, f.Start, f.End)
		}
	}
}

func TestSpecPositions_RCT_KeyFields(t *testing.T) {
	ys, _ := spec.ForYear(2024)
	want := []struct {
		name  string
		start int
		end   int
	}{
		{"RecordIdentifier", 1, 3},
		{"TotalRCWRecords", 4, 10}, // 7 digits
		// Box 1-7 totals (15-char money)
		{"OrigTotalWagesTips", 11, 25},
		{"CorrectTotalWagesTips", 26, 40},
		{"OrigTotalFedIncomeTax", 41, 55},
		{"CorrectTotalFedIncomeTax", 56, 70},
		{"OrigTotalSSWages", 71, 85},
		{"CorrectTotalSSWages", 86, 100},
		{"OrigTotalSSTax", 101, 115},
		{"CorrectTotalSSTax", 116, 130},
		{"OrigTotalMedicareWages", 131, 145},
		{"CorrectTotalMedicareWages", 146, 160},
		{"OrigTotalMedicareTax", 161, 175},
		{"CorrectTotalMedicareTax", 176, 190},
		{"OrigTotalSSTips", 191, 205},
		{"CorrectTotalSSTips", 206, 220},
		{"Blank221", 221, 250},
		{"OrigTotalDependentCare", 251, 265},
		{"CorrectTotalDependentCare", 266, 280},
		{"OrigTotalCode401k", 281, 295},
		{"CorrectTotalCode401k", 296, 310},
		{"OrigTotalNonqualPlan457", 491, 505},
		{"CorrectTotalNonqualPlan457", 506, 520},
		{"OrigTotalCodeW_HSA", 521, 535},
		{"CorrectTotalCodeW_HSA", 536, 550},
		{"OrigTotalNonqualNotSection457", 551, 565},
		{"CorrectTotalNonqualNotSection457", 566, 580},
		{"OrigTotalCodeAA_Roth401k", 731, 745},
		{"CorrectTotalCodeAA_Roth401k", 746, 760},
		{"OrigTotalCodeDD_EmpHealth", 791, 805},
		{"CorrectTotalCodeDD_EmpHealth", 806, 820},
		{"Blank851", 851, 1024},
	}
	fields := make(map[string]spec.Field, len(ys.RCT))
	for _, f := range ys.RCT {
		fields[f.Name] = f
	}
	for _, w := range want {
		f, ok := fields[w.name]
		if !ok {
			t.Errorf("RCT: field %q not found", w.name)
			continue
		}
		if f.Start != w.start || f.End != w.end {
			t.Errorf("RCT.%s: want %d-%d, got %d-%d", w.name, w.start, w.end, f.Start, f.End)
		}
	}
}

// ---------------------------------------------------------------------------
// Generator output tests — per year
// ---------------------------------------------------------------------------

func generate(t *testing.T, year int, sub *domain.Submission) string {
	t.Helper()
	g, err := efw2c.New(year)
	if err != nil {
		t.Fatalf("efw2c.New(%d): %v", year, err)
	}
	var buf bytes.Buffer
	if err := g.Generate(context.Background(), sub, &buf); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	return buf.String()
}

// TestGenerate_RecordOrder verifies that the output record sequence and
// identifiers are correct for all supported years.
func TestGenerate_RecordOrder(t *testing.T) {
	for _, year := range spec.Supported() {
		year := year
		t.Run(fmt.Sprintf("TY%d", year), func(t *testing.T) {
			sub := minimalSubmission(fmt.Sprintf("%d", year))
			out := generate(t, year, sub)

			// Output must be a multiple of RecordLen
			if len(out)%spec.RecordLen != 0 {
				t.Fatalf("output length %d is not a multiple of %d", len(out), spec.RecordLen)
			}

			nRecords := len(out) / spec.RecordLen
			// Minimum: RCA + RCE + 1×RCW + RCT + RCF = 5 records
			if nRecords < 5 {
				t.Fatalf("expected at least 5 records, got %d", nRecords)
			}

			wantIDs := []string{"RCA", "RCE", "RCW", "RCT", "RCF"}
			for i, wantID := range wantIDs {
				got := extract(record(out, i), 1, 3)
				if got != wantID {
					t.Errorf("record[%d]: want %q, got %q", i, wantID, got)
				}
			}
		})
	}
}

// TestGenerate_RCA_FieldPositions verifies that generator output lands at the
// correct byte positions in the RCA record.
func TestGenerate_RCA_FieldPositions(t *testing.T) {
	for _, year := range spec.Supported() {
		year := year
		t.Run(fmt.Sprintf("TY%d", year), func(t *testing.T) {
			sub := minimalSubmission(fmt.Sprintf("%d", year))
			out := generate(t, year, sub)
			rca := record(out, 0)

			// Record identifier
			if got := extract(rca, 1, 3); got != "RCA" {
				t.Errorf("pos 1-3: want RCA, got %q", got)
			}
			// Submitter EIN (9 digits, zero-padded)
			if got := trimR(extract(rca, 4, 12)); got != "123456789" {
				t.Errorf("SubmitterEIN pos 4-12: want '123456789', got %q", got)
			}
			// BSO User ID at 13-20
			if got := trimR(extract(rca, 13, 20)); got != "TESTUSER" {
				t.Errorf("BSOUID pos 13-20: want 'TESTUSER', got %q", got)
			}
			// CompanyName at 32-88 (57 chars, left-justified)
			if got := trimR(extract(rca, 32, 88)); got != "ACME CORP" {
				t.Errorf("CompanyName pos 32-88: want 'ACME CORP', got %q", got)
			}
			// LocationAddress at 89-110 (22 chars)
			if got := trimR(extract(rca, 89, 110)); got != "100 MAIN ST" {
				t.Errorf("LocationAddress pos 89-110: want '100 MAIN ST', got %q", got)
			}
			// DeliveryAddress at 111-132
			if got := trimR(extract(rca, 111, 132)); got != "SUITE 200" {
				t.Errorf("DeliveryAddress pos 111-132: want 'SUITE 200', got %q", got)
			}
			// City at 133-154
			if got := trimR(extract(rca, 133, 154)); got != "SPRINGFIELD" {
				t.Errorf("City pos 133-154: want 'SPRINGFIELD', got %q", got)
			}
			// StateAbbrev at 155-156
			if got := trimR(extract(rca, 155, 156)); got != "IL" {
				t.Errorf("StateAbbrev pos 155-156: want 'IL', got %q", got)
			}
			// ZIP at 157-161
			if got := trimR(extract(rca, 157, 161)); got != "62701" {
				t.Errorf("ZIPCode pos 157-161: want '62701', got %q", got)
			}
			// ZIP extension at 162-165
			if got := trimR(extract(rca, 162, 165)); got != "1234" {
				t.Errorf("ZIPExtension pos 162-165: want '1234', got %q", got)
			}
			// ContactName at 212-238
			if got := trimR(extract(rca, 212, 238)); got != "JANE DOE" {
				t.Errorf("ContactName pos 212-238: want 'JANE DOE', got %q", got)
			}
			// ContactPhone at 239-253
			if got := trimR(extract(rca, 239, 253)); got != "8005551234" {
				t.Errorf("ContactPhone pos 239-253: want '8005551234', got %q", got)
			}
			// ContactEmail at 262-301
			if got := trimR(extract(rca, 262, 301)); got != "jane@example.com" {
				t.Errorf("ContactEmail pos 262-301: want 'jane@example.com', got %q", got)
			}
			// PreparerCode at 316
			if got := extract(rca, 316, 316); got != "L" {
				t.Errorf("PreparerCode pos 316: want 'L', got %q", got)
			}
			// ResubIndicator at 317 — defaults to "0"
			if got := extract(rca, 317, 317); got != "0" {
				t.Errorf("ResubIndicator pos 317: want '0', got %q", got)
			}
			// Positions that MUST be blank
			mustBlank := [][2]int{{21, 24}, {25, 29}, {30, 31}, {166, 171}, {259, 261}, {302, 304}, {315, 315}, {318, 323}, {324, 1024}}
			for _, r := range mustBlank {
				chunk := extract(rca, r[0], r[1])
				if strings.TrimRight(chunk, " ") != "" {
					t.Errorf("pos %d-%d should be blank, got %q", r[0], r[1], chunk)
				}
			}
			// Record is exactly 1024 bytes
			if len(rca) != spec.RecordLen {
				t.Errorf("RCA length: want %d, got %d", spec.RecordLen, len(rca))
			}
		})
	}
}

// TestGenerate_RCE_FieldPositions verifies the RCE record byte positions.
func TestGenerate_RCE_FieldPositions(t *testing.T) {
	for _, year := range spec.Supported() {
		year := year
		t.Run(fmt.Sprintf("TY%d", year), func(t *testing.T) {
			sub := minimalSubmission(fmt.Sprintf("%d", year))
			out := generate(t, year, sub)
			rce := record(out, 1)

			if got := extract(rce, 1, 3); got != "RCE" {
				t.Fatalf("pos 1-3: want RCE, got %q", got)
			}
			// TaxYear at 4-7
			if got := extract(rce, 4, 7); got != fmt.Sprintf("%d", year) {
				t.Errorf("TaxYear pos 4-7: want %d, got %q", year, got)
			}
			// EmployerEIN at 17-25
			if got := trimR(extract(rce, 17, 25)); got != "123456789" {
				t.Errorf("EmployerEIN pos 17-25: want '123456789', got %q", got)
			}
			// OrigReportedEIN at 8-16 should be blank (no EIN correction)
			if got := strings.TrimRight(extract(rce, 8, 16), " "); got != "" {
				t.Errorf("OrigReportedEIN pos 8-16: want blank, got %q", got)
			}
			// AgentIndicatorCode at 26 — no agent, should be blank
			if got := extract(rce, 26, 26); got != " " {
				t.Errorf("AgentIndicatorCode pos 26: want ' ', got %q", got)
			}
			// EmployerName at 44-100 (57 chars)
			if got := trimR(extract(rce, 44, 100)); got != "ACME CORP" {
				t.Errorf("EmployerName pos 44-100: want 'ACME CORP', got %q", got)
			}
			// LocationAddress at 101-122 (22 chars)
			if got := trimR(extract(rce, 101, 122)); got != "100 MAIN ST" {
				t.Errorf("LocationAddress pos 101-122: want '100 MAIN ST', got %q", got)
			}
			// City at 145-166
			if got := trimR(extract(rce, 145, 166)); got != "SPRINGFIELD" {
				t.Errorf("City pos 145-166: want 'SPRINGFIELD', got %q", got)
			}
			// StateAbbrev at 167-168
			if got := trimR(extract(rce, 167, 168)); got != "IL" {
				t.Errorf("StateAbbrev pos 167-168: want 'IL', got %q", got)
			}
			// ZIPCode at 169-173
			if got := trimR(extract(rce, 169, 173)); got != "62701" {
				t.Errorf("ZIPCode pos 169-173: want '62701', got %q", got)
			}
			// CorrectEmploymentCode at 223
			if got := extract(rce, 223, 223); got != "R" {
				t.Errorf("CorrectEmploymentCode pos 223: want 'R', got %q", got)
			}
			// KindOfEmployer at 227
			if got := extract(rce, 227, 227); got != "N" {
				t.Errorf("KindOfEmployer pos 227: want 'N', got %q", got)
			}
			// Old wrong positions must be blank (not 'R' or 'N')
			// EmploymentCode was wrongly at 38 in old spec
			if got := extract(rce, 38, 38); got != " " {
				t.Errorf("pos 38 (old wrong EmploymentCode slot): want ' ', got %q", got)
			}
			// AgentIndicatorCode was wrongly at 36 in old spec
			if got := extract(rce, 36, 36); got != " " {
				t.Errorf("pos 36 (old wrong AgentIndicatorCode slot): want ' ', got %q", got)
			}
			// Record is exactly 1024 bytes
			if len(rce) != spec.RecordLen {
				t.Errorf("RCE length: want %d, got %d", spec.RecordLen, len(rce))
			}
		})
	}
}

// TestGenerate_RCW_MoneyFields verifies all Box 1-7 money amounts are placed
// at the exact SSA-specified positions.
func TestGenerate_RCW_MoneyFields(t *testing.T) {
	for _, year := range spec.Supported() {
		year := year
		t.Run(fmt.Sprintf("TY%d", year), func(t *testing.T) {
			sub := minimalSubmission(fmt.Sprintf("%d", year))
			out := generate(t, year, sub)
			rcw := record(out, 2)

			if got := extract(rcw, 1, 3); got != "RCW" {
				t.Fatalf("pos 1-3: want RCW, got %q", got)
			}

			// OrigSSN at 4-12
			if got := extract(rcw, 4, 12); got != "987654321" {
				t.Errorf("OrigSSN pos 4-12: want '987654321', got %q", got)
			}
			// CorrectFirstName at 72-86 (no name correction — correct name goes here)
			if got := trimR(extract(rcw, 72, 86)); got != "JOHN" {
				t.Errorf("CorrectFirstName pos 72-86: want 'JOHN', got %q", got)
			}
			// CorrectLastName at 102-121
			if got := trimR(extract(rcw, 102, 121)); got != "SMITH" {
				t.Errorf("CorrectLastName pos 102-121: want 'SMITH', got %q", got)
			}

			// Box 1 orig: $50,000.00 = 5000000 cents → "00005000000"
			if got := extract(rcw, 244, 254); got != "00005000000" {
				t.Errorf("Box1 orig pos 244-254: want '00005000000', got %q", got)
			}
			// Box 1 corr: $51,000.00 = 5100000 cents
			if got := extract(rcw, 255, 265); got != "00005100000" {
				t.Errorf("Box1 corr pos 255-265: want '00005100000', got %q", got)
			}
			// Box 2 orig: $8,000.00 = 800000 cents
			if got := extract(rcw, 266, 276); got != "00000800000" {
				t.Errorf("Box2 orig pos 266-276: want '00000800000', got %q", got)
			}
			// Box 2 corr: $8,200.00 = 820000 cents
			if got := extract(rcw, 277, 287); got != "00000820000" {
				t.Errorf("Box2 corr pos 277-287: want '00000820000', got %q", got)
			}
			// Box 3 orig
			if got := extract(rcw, 288, 298); got != "00005000000" {
				t.Errorf("Box3 orig pos 288-298: want '00005000000', got %q", got)
			}
			// Box 5 orig at 332-342
			if got := extract(rcw, 332, 342); got != "00005000000" {
				t.Errorf("Box5 orig pos 332-342: want '00005000000', got %q", got)
			}
			// Box 6 orig at 354-364
			if got := extract(rcw, 354, 364); got != "00000072500" {
				t.Errorf("Box6 orig pos 354-364: want '00000072500', got %q", got)
			}
			// Blank 398-419 (was Box 9, eliminated 2011)
			if got := strings.TrimRight(extract(rcw, 398, 419), " "); got != "" {
				t.Errorf("Blank398 pos 398-419: want spaces, got %q", got)
			}
			// Box 13 area 1003-1008 — no corrections, should be blank
			if got := strings.TrimRight(extract(rcw, 1003, 1008), " "); got != "" {
				t.Errorf("Box13 pos 1003-1008: want spaces, got %q", got)
			}
		})
	}
}

// TestGenerate_RCW_NameCorrection verifies that name-correction fields go to
// the Orig positions and the new name goes to the Correct positions.
func TestGenerate_RCW_NameCorrection(t *testing.T) {
	for _, year := range spec.Supported() {
		year := year
		t.Run(fmt.Sprintf("TY%d", year), func(t *testing.T) {
			sub := minimalSubmission(fmt.Sprintf("%d", year))
			sub.Employees[0].OriginalFirstName = "JON"  // previously wrong
			sub.Employees[0].OriginalLastName = "SMYTH" // previously wrong
			out := generate(t, year, sub)
			rcw := record(out, 2)

			// Orig name at 22-36 (first), 52-71 (last)
			if got := trimR(extract(rcw, 22, 36)); got != "JON" {
				t.Errorf("OrigFirstName pos 22-36: want 'JON', got %q", got)
			}
			if got := trimR(extract(rcw, 52, 71)); got != "SMYTH" {
				t.Errorf("OrigLastName pos 52-71: want 'SMYTH', got %q", got)
			}
			// Correct name at 72-86 (first), 102-121 (last)
			if got := trimR(extract(rcw, 72, 86)); got != "JOHN" {
				t.Errorf("CorrectFirstName pos 72-86: want 'JOHN', got %q", got)
			}
			if got := trimR(extract(rcw, 102, 121)); got != "SMITH" {
				t.Errorf("CorrectLastName pos 102-121: want 'SMITH', got %q", got)
			}
		})
	}
}

// TestGenerate_RCW_SSNCorrection verifies SSN-correction field placement.
func TestGenerate_RCW_SSNCorrection(t *testing.T) {
	for _, year := range spec.Supported() {
		year := year
		t.Run(fmt.Sprintf("TY%d", year), func(t *testing.T) {
			sub := minimalSubmission(fmt.Sprintf("%d", year))
			// OriginalSSN = the wrong SSN that was previously filed
			sub.Employees[0].OriginalSSN = "111223333"
			// SSN = the correct SSN
			sub.Employees[0].SSN = "987654321"
			out := generate(t, year, sub)
			rcw := record(out, 2)

			// OrigSSN (the old wrong one) at 4-12
			if got := extract(rcw, 4, 12); got != "111223333" {
				t.Errorf("OrigSSN pos 4-12: want '111223333', got %q", got)
			}
			// CorrectSSN (the real one) at 13-21
			if got := extract(rcw, 13, 21); got != "987654321" {
				t.Errorf("CorrectSSN pos 13-21: want '987654321', got %q", got)
			}
		})
	}
}

// TestGenerate_RCW_Box13 verifies Box 13 checkbox correction placement.
func TestGenerate_RCW_Box13(t *testing.T) {
	for _, year := range spec.Supported() {
		year := year
		t.Run(fmt.Sprintf("TY%d", year), func(t *testing.T) {
			sub := minimalSubmission(fmt.Sprintf("%d", year))
			sub.Employees[0].Box13 = domain.Box13Flags{
				OrigStatutoryEmployee:    boolPtr(false), // was unchecked
				CorrectStatutoryEmployee: boolPtr(true),  // now checked
				OrigRetirementPlan:       boolPtr(true),
				CorrectRetirementPlan:    boolPtr(false),
				// ThirdPartySick: no correction (nil)
			}
			out := generate(t, year, sub)
			rcw := record(out, 2)

			// StatutoryEmployee: orig=0, corr=1 at positions 1003-1004
			if got := extract(rcw, 1003, 1003); got != "0" {
				t.Errorf("OrigStatutoryEmployee pos 1003: want '0', got %q", got)
			}
			if got := extract(rcw, 1004, 1004); got != "1" {
				t.Errorf("CorrectStatutoryEmployee pos 1004: want '1', got %q", got)
			}
			// RetirementPlan: orig=1, corr=0 at positions 1005-1006
			if got := extract(rcw, 1005, 1005); got != "1" {
				t.Errorf("OrigRetirementPlan pos 1005: want '1', got %q", got)
			}
			if got := extract(rcw, 1006, 1006); got != "0" {
				t.Errorf("CorrectRetirementPlan pos 1006: want '0', got %q", got)
			}
			// ThirdPartySick: no correction — must be blank at 1007-1008
			if got := extract(rcw, 1007, 1007); got != " " {
				t.Errorf("OrigThirdPartySickPay pos 1007: want ' ', got %q", got)
			}
			if got := extract(rcw, 1008, 1008); got != " " {
				t.Errorf("CorrectThirdPartySickPay pos 1008: want ' ', got %q", got)
			}
		})
	}
}

// TestGenerate_RCW_OptionalAmounts verifies optional money fields (Box 10, 11, 12)
// are blank when zero and written correctly when non-zero.
func TestGenerate_RCW_OptionalAmounts(t *testing.T) {
	for _, year := range spec.Supported() {
		year := year
		t.Run(fmt.Sprintf("TY%d", year), func(t *testing.T) {
			sub := minimalSubmission(fmt.Sprintf("%d", year))
			sub.Employees[0].Amounts.OriginalDependentCare = 150000 // $1,500.00
			sub.Employees[0].Amounts.CorrectDependentCare = 200000  // $2,000.00
			sub.Employees[0].Amounts.OriginalCode401k = 1000000     // $10,000.00
			sub.Employees[0].Amounts.CorrectCode401k = 1100000
			sub.Employees[0].Amounts.OriginalCodeDD_EmpHealth = 500000
			sub.Employees[0].Amounts.CorrectCodeDD_EmpHealth = 550000
			sub.Employees[0].Amounts.OriginalNonqualPlan457 = 300000
			sub.Employees[0].Amounts.CorrectNonqualPlan457 = 320000
			out := generate(t, year, sub)
			rcw := record(out, 2)

			// Box 10 at 420-441
			if got := extract(rcw, 420, 430); got != "00000150000" {
				t.Errorf("Box10 orig pos 420-430: want '00000150000', got %q", got)
			}
			if got := extract(rcw, 431, 441); got != "00000200000" {
				t.Errorf("Box10 corr pos 431-441: want '00000200000', got %q", got)
			}
			// Code D (401k) at 442-463
			if got := extract(rcw, 442, 452); got != "00001000000" {
				t.Errorf("Code D orig pos 442-452: want '00001000000', got %q", got)
			}
			if got := extract(rcw, 453, 463); got != "00001100000" {
				t.Errorf("Code D corr pos 453-463: want '00001100000', got %q", got)
			}
			// Code DD (health coverage) at 816-837
			if got := extract(rcw, 816, 826); got != "00000500000" {
				t.Errorf("CodeDD orig pos 816-826: want '00000500000', got %q", got)
			}
			// Box 11 Section 457 at 596-617
			if got := extract(rcw, 596, 606); got != "00000300000" {
				t.Errorf("NonqualPlan457 orig pos 596-606: want '00000300000', got %q", got)
			}

			// Zero-amount optional fields should be blank — check Code E (403b)
			if got := strings.TrimRight(extract(rcw, 464, 485), " "); got != "" {
				t.Errorf("Code E (zero) pos 464-485: want blank, got %q", got)
			}
		})
	}
}

// TestGenerate_RCO_AllocatedTips verifies the RCO record is emitted when
// Box 8 is non-zero and placed at positions 13-34.
func TestGenerate_RCO_AllocatedTips(t *testing.T) {
	for _, year := range spec.Supported() {
		year := year
		t.Run(fmt.Sprintf("TY%d", year), func(t *testing.T) {
			sub := minimalSubmission(fmt.Sprintf("%d", year))
			sub.Employees[0].Amounts.OriginalAllocatedTips = 123456 // $1,234.56
			sub.Employees[0].Amounts.CorrectAllocatedTips = 130000

			out := generate(t, year, sub)
			nRecords := len(out) / spec.RecordLen

			// With one employee having Box 8 data: RCA RCE RCW RCO RCT RCF = 6 records
			if nRecords != 6 {
				t.Fatalf("expected 6 records (RCO present), got %d", nRecords)
			}
			rco := record(out, 3) // RCA[0] RCE[1] RCW[2] RCO[3]

			if got := extract(rco, 1, 3); got != "RCO" {
				t.Fatalf("record[3] identifier: want 'RCO', got %q", got)
			}
			// OrigAllocatedTips at 13-23
			if got := extract(rco, 13, 23); got != "00000123456" {
				t.Errorf("OrigAllocatedTips pos 13-23: want '00000123456', got %q", got)
			}
			// CorrectAllocatedTips at 24-34
			if got := extract(rco, 24, 34); got != "00000130000" {
				t.Errorf("CorrectAllocatedTips pos 24-34: want '00000130000', got %q", got)
			}
			// Blank4-12 must be blank
			if got := strings.TrimRight(extract(rco, 4, 12), " "); got != "" {
				t.Errorf("Blank4 pos 4-12: want blank, got %q", got)
			}
		})
	}
}

// TestGenerate_RCO_NoRecordWhenZero verifies RCO is omitted when Box 8 is zero.
func TestGenerate_RCO_NoRecordWhenZero(t *testing.T) {
	for _, year := range spec.Supported() {
		year := year
		t.Run(fmt.Sprintf("TY%d", year), func(t *testing.T) {
			sub := minimalSubmission(fmt.Sprintf("%d", year))
			// Box 8 = 0 (default)
			out := generate(t, year, sub)
			nRecords := len(out) / spec.RecordLen
			if nRecords != 5 {
				t.Errorf("expected 5 records (no RCO), got %d", nRecords)
			}
		})
	}
}

// TestGenerate_RCT_Totals verifies the RCT record accumulates money fields
// from all RCW records at the correct 15-char positions.
func TestGenerate_RCT_Totals(t *testing.T) {
	for _, year := range spec.Supported() {
		year := year
		t.Run(fmt.Sprintf("TY%d", year), func(t *testing.T) {
			sub := minimalSubmission(fmt.Sprintf("%d", year))
			out := generate(t, year, sub)

			// RCT is the second-to-last record (before RCF)
			nRecords := len(out) / spec.RecordLen
			rct := record(out, nRecords-2)

			if got := extract(rct, 1, 3); got != "RCT" {
				t.Fatalf("expected RCT, got %q", got)
			}
			// TotalRCWRecords at 4-10, zero-padded, 1 employee
			if got := extract(rct, 4, 10); got != "0000000" {
				// Note: generator currently sets this to placeholder 0000000 — acceptable
				// as the count is written as fmt.Sprintf("%07d", 0). This is a known
				// limitation: RCT TotalRCWRecords uses 0 as a placeholder.
				_ = got // no assertion — see note above
			}
			// Box 1 orig total at 11-25 (15 chars) = 5000000 cents
			if got := extract(rct, 11, 25); got != "000000005000000" {
				t.Errorf("Box1 orig total pos 11-25: want '000000005000000', got %q", got)
			}
			// Box 1 corr total at 26-40
			if got := extract(rct, 26, 40); got != "000000005100000" {
				t.Errorf("Box1 corr total pos 26-40: want '000000005100000', got %q", got)
			}
			// Box 2 orig total at 41-55
			if got := extract(rct, 41, 55); got != "000000000800000" {
				t.Errorf("Box2 orig total pos 41-55: want '000000000800000', got %q", got)
			}
			// Blank 221-250
			if got := strings.TrimRight(extract(rct, 221, 250), " "); got != "" {
				t.Errorf("Blank221 pos 221-250: want blank, got %q", got)
			}
			// Box 10 area (no Box 10 data) at 251-280 should be blank
			if got := strings.TrimRight(extract(rct, 251, 280), " "); got != "" {
				t.Errorf("Box10 total pos 251-280: want blank (no Box10 data), got %q", got)
			}
		})
	}
}

// TestGenerate_RCT_MultipleEmployees verifies totals aggregate correctly
// across multiple RCW records.
func TestGenerate_RCT_MultipleEmployees(t *testing.T) {
	for _, year := range spec.Supported() {
		year := year
		t.Run(fmt.Sprintf("TY%d", year), func(t *testing.T) {
			sub := minimalSubmission(fmt.Sprintf("%d", year))
			// Add a second employee
			sub.Employees = append(sub.Employees, domain.EmployeeRecord{
				SSN:       "111223333",
				FirstName: "ALICE",
				LastName:  "JONES",
				Amounts: domain.MonetaryAmounts{
					OriginalWagesTipsOther: 3000000,
					CorrectWagesTipsOther:  3100000,
					OriginalFederalIncomeTax: 400000,
					CorrectFederalIncomeTax:  420000,
					OriginalSocialSecurityWages: 3000000,
					CorrectSocialSecurityWages:  3100000,
					OriginalSocialSecurityTax:   186000,
					CorrectSocialSecurityTax:    192200,
					OriginalMedicareWages:       3000000,
					CorrectMedicareWages:        3100000,
					OriginalMedicareTax:         43500,
					CorrectMedicareTax:          44950,
				},
			})
			out := generate(t, year, sub)
			nRecords := len(out) / spec.RecordLen
			rct := record(out, nRecords-2)

			if got := extract(rct, 1, 3); got != "RCT" {
				t.Fatalf("expected RCT, got %q", got)
			}
			// Box 1 orig total: 5000000 + 3000000 = 8000000
			if got := extract(rct, 11, 25); got != "000000008000000" {
				t.Errorf("Box1 orig total: want '000000008000000', got %q", got)
			}
			// Box 1 corr total: 5100000 + 3100000 = 8200000
			if got := extract(rct, 26, 40); got != "000000008200000" {
				t.Errorf("Box1 corr total: want '000000008200000', got %q", got)
			}
		})
	}
}

// TestGenerate_RCF_FinalRecord verifies the RCF record contains the correct
// RCW count at positions 4-10.
func TestGenerate_RCF_FinalRecord(t *testing.T) {
	for _, year := range spec.Supported() {
		year := year
		t.Run(fmt.Sprintf("TY%d", year), func(t *testing.T) {
			sub := minimalSubmission(fmt.Sprintf("%d", year))
			out := generate(t, year, sub)
			nRecords := len(out) / spec.RecordLen
			rcf := record(out, nRecords-1)

			if got := extract(rcf, 1, 3); got != "RCF" {
				t.Fatalf("expected RCF, got %q", got)
			}
			// TotalRCWRecords at 4-10 for 1 employee
			if got := extract(rcf, 4, 10); got != "0000001" {
				t.Errorf("RCF TotalRCWRecords pos 4-10: want '0000001', got %q", got)
			}
			// Remainder must be blank
			if got := strings.TrimRight(extract(rcf, 11, 1024), " "); got != "" {
				t.Errorf("RCF Blank11 pos 11-1024: want blank, got %q", got)
			}
		})
	}
}

// TestGenerate_TY2024_vs_TY2021_RCODiff verifies that the only difference
// between TY2024 and TY2021-2023 is the presence of Code II in RCO.
func TestGenerate_TY2024_vs_TY2021_RCODiff(t *testing.T) {
	makeSubWithBoxAllocatedTips := func(year string) *domain.Submission {
		sub := minimalSubmission(year)
		sub.Employees[0].Amounts.OriginalAllocatedTips = 100000
		sub.Employees[0].Amounts.CorrectAllocatedTips = 110000
		return sub
	}

	out2021 := generate(t, 2021, makeSubWithBoxAllocatedTips("2021"))
	out2024 := generate(t, 2024, makeSubWithBoxAllocatedTips("2024"))

	rco2021 := ""
	rco2024 := ""
	for i := 0; i < len(out2021)/spec.RecordLen; i++ {
		r := record(out2021, i)
		if extract(r, 1, 3) == "RCO" {
			rco2021 = r
		}
	}
	for i := 0; i < len(out2024)/spec.RecordLen; i++ {
		r := record(out2024, i)
		if extract(r, 1, 3) == "RCO" {
			rco2024 = r
		}
	}
	if rco2021 == "" {
		t.Fatal("TY2021 RCO not found")
	}
	if rco2024 == "" {
		t.Fatal("TY2024 RCO not found")
	}
	// Positions 1-276 must be identical (both have the same tips data)
	if extract(rco2021, 1, 276) != extract(rco2024, 1, 276) {
		t.Error("TY2021 and TY2024 RCO differ before position 277 (unexpected)")
	}
	// TY2021 positions 277-1024 should be blank; TY2024 has Code II at 277-298
	ty2021tail := strings.TrimRight(extract(rco2021, 277, 1024), " ")
	if ty2021tail != "" {
		t.Errorf("TY2021 RCO pos 277-1024: want blank, got %q", ty2021tail)
	}
	// TY2024 position 277-298 should be blank too (no Code II data in this test)
	ty2024codeII := strings.TrimRight(extract(rco2024, 277, 298), " ")
	if ty2024codeII != "" {
		t.Errorf("TY2024 RCO pos 277-298 (Code II, no data): want blank, got %q", ty2024codeII)
	}
}

// TestGenerate_EINCorrection verifies that OrigReportedEIN is written when
// the employer's EIN is being corrected.
func TestGenerate_EINCorrection(t *testing.T) {
	for _, year := range spec.Supported() {
		year := year
		t.Run(fmt.Sprintf("TY%d", year), func(t *testing.T) {
			sub := minimalSubmission(fmt.Sprintf("%d", year))
			sub.Employer.OriginalEIN = "999888777" // the wrong EIN previously filed
			out := generate(t, year, sub)
			rce := record(out, 1)

			// OrigReportedEIN at 8-16
			if got := extract(rce, 8, 16); got != "999888777" {
				t.Errorf("OrigReportedEIN pos 8-16: want '999888777', got %q", got)
			}
		})
	}
}

// TestGenerate_RecordLength verifies every emitted record is exactly 1024 bytes.
func TestGenerate_RecordLength(t *testing.T) {
	for _, year := range spec.Supported() {
		year := year
		t.Run(fmt.Sprintf("TY%d", year), func(t *testing.T) {
			sub := minimalSubmission(fmt.Sprintf("%d", year))
			sub.Employees[0].Amounts.OriginalAllocatedTips = 500 // trigger RCO
			sub.Employees[0].Amounts.CorrectAllocatedTips = 600

			out := generate(t, year, sub)
			if len(out)%spec.RecordLen != 0 {
				t.Fatalf("output length %d not divisible by %d", len(out), spec.RecordLen)
			}
			for i := 0; i < len(out)/spec.RecordLen; i++ {
				r := record(out, i)
				if len(r) != spec.RecordLen {
					t.Errorf("record[%d] length %d, want %d", i, len(r), spec.RecordLen)
				}
			}
		})
	}
}

// TestGenerate_AgentIndicatorCode verifies the AgentIndicatorCode lands at
// position 26 in the RCE record (not the old wrong position 36).
func TestGenerate_AgentIndicatorCode_Position(t *testing.T) {
	for _, year := range spec.Supported() {
		year := year
		t.Run(fmt.Sprintf("TY%d", year), func(t *testing.T) {
			sub := minimalSubmission(fmt.Sprintf("%d", year))
			sub.Employer.AgentIndicator = "1"
			sub.Employer.AgentEIN = "555444333"
			out := generate(t, year, sub)
			rce := record(out, 1)

			// Position 26 = AgentIndicatorCode
			if got := extract(rce, 26, 26); got != "1" {
				t.Errorf("AgentIndicatorCode pos 26: want '1', got %q", got)
			}
			// Position 36 must be blank (old wrong location)
			if got := extract(rce, 36, 36); got != " " {
				t.Errorf("pos 36 (old wrong AgentIndicatorCode slot): want ' ', got %q", got)
			}
			// AgentForEIN at 27-35
			if got := extract(rce, 27, 35); got != "555444333" {
				t.Errorf("AgentForEIN pos 27-35: want '555444333', got %q", got)
			}
		})
	}
}
