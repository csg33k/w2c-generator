package handlers

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/csg33k/w2c-generator/internal/adapters/pdf"
	"github.com/csg33k/w2c-generator/internal/domain"
	"github.com/csg33k/w2c-generator/internal/ports"
	"github.com/csg33k/w2c-generator/internal/templates"
)

type Handler struct {
	repo ports.SubmissionRepository
	gen  ports.EFW2CGenerator
}

func New(repo ports.SubmissionRepository, gen ports.EFW2CGenerator) *Handler {
	return &Handler{repo: repo, gen: gen}
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", h.index)
	mux.HandleFunc("POST /submissions", h.createSubmission)
	mux.HandleFunc("GET /submissions/{id}", h.viewSubmission)
	mux.HandleFunc("DELETE /submissions/{id}", h.deleteSubmission)
	mux.HandleFunc("GET /submissions/{id}/edit", h.editSubmissionForm)
	mux.HandleFunc("GET /submissions/{id}/header", h.getSubmissionHeader)
	mux.HandleFunc("PUT /submissions/{id}", h.updateSubmission)
	mux.HandleFunc("POST /submissions/{id}/employees", h.addEmployee)
	mux.HandleFunc("GET /employees/{id}/edit", h.editEmployeeForm)
	mux.HandleFunc("GET /employees/{id}/card", h.getEmployeeCard)
	mux.HandleFunc("PUT /employees/{id}", h.updateEmployee)
	mux.HandleFunc("DELETE /employees/{id}", h.deleteEmployee)
	mux.HandleFunc("GET /submissions/{id}/generate", h.generateFile)
	mux.HandleFunc("GET /submissions/{id}/pdf", h.generatePDF)
	return mux
}

func (h *Handler) index(w http.ResponseWriter, r *http.Request) {
	submissions, err := h.repo.ListSubmissions(r.Context())
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	render(w, r, templates.Index(submissions, h.gen.SupportedYears()))
}

func (h *Handler) createSubmission(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	s := &domain.Submission{
		Submitter: domain.SubmitterInfo{
			BSOUID:       r.FormValue("bso_uid"),
			ContactName:  r.FormValue("contact_name"),
			ContactPhone: stripNonDigits(r.FormValue("contact_phone")),
			ContactEmail: r.FormValue("contact_email"),
			PreparerCode: r.FormValue("preparer_code"),
		},
		Employer: domain.EmployerRecord{
			EmploymentCode: r.FormValue("employment_code"),
			KindOfEmployer: r.FormValue("kind_of_employer"),
			ContactName:    r.FormValue("employer_contact_name"),
			ContactPhone:   stripNonDigits(r.FormValue("employer_contact_phone")),
			ContactEmail:   r.FormValue("employer_contact_email"),
			EIN:            stripDashes(r.FormValue("ein")),
			Name:           r.FormValue("employer_name"),
			AddressLine1:   r.FormValue("emp_addr1"),
			AddressLine2:   r.FormValue("emp_addr2"),
			City:           r.FormValue("emp_city"),
			State:          r.FormValue("emp_state"),
			ZIP:            r.FormValue("emp_zip"),
			ZIPExtension:   r.FormValue("emp_zip_ext"),
			AgentIndicator: "0",
			TaxYear:        r.FormValue("tax_year"),
		},
		Notes: r.FormValue("notes"),
	}
	// Validate: if the submitted year isn't supported, fall back to default.
	if s.Employer.TaxYear == "" {
		supported := h.gen.SupportedYears()
		s.Employer.TaxYear = supported[len(supported)-1].Year
	}
	if err := h.repo.CreateSubmission(r.Context(), s); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("HX-Redirect", fmt.Sprintf("/submissions/%d", s.ID))
	w.WriteHeader(http.StatusCreated)
}

func (h *Handler) viewSubmission(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		http.Error(w, "invalid id", 400)
		return
	}
	s, err := h.repo.GetSubmission(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	render(w, r, templates.Detail(s))
}

// editSubmissionForm renders the inline edit form for the submission header.
func (h *Handler) editSubmissionForm(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		http.Error(w, "invalid id", 400)
		return
	}
	s, err := h.repo.GetSubmission(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	render(w, r, templates.SubmissionEditForm(s, h.gen.SupportedYears()))
}

// getSubmissionHeader renders the read-only submission header fragment (used by cancel).
func (h *Handler) getSubmissionHeader(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		http.Error(w, "invalid id", 400)
		return
	}
	s, err := h.repo.GetSubmission(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	render(w, r, templates.SubmissionHeader(s))
}

// updateSubmission handles PUT /submissions/{id} and renders the updated header.
func (h *Handler) updateSubmission(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		http.Error(w, "invalid id", 400)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	// Fetch first to preserve CreatedAt, SubmittedAt, Employees, etc.
	s, err := h.repo.GetSubmission(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	s.Submitter.BSOUID = r.FormValue("bso_uid")
	s.Submitter.ContactName = r.FormValue("contact_name")
	s.Submitter.ContactPhone = stripNonDigits(r.FormValue("contact_phone"))
	s.Submitter.ContactEmail = r.FormValue("contact_email")
	s.Submitter.PreparerCode = r.FormValue("preparer_code")
	s.Employer.EIN = stripDashes(r.FormValue("ein"))
	s.Employer.Name = r.FormValue("employer_name")
	s.Employer.AddressLine1 = r.FormValue("emp_addr1")
	s.Employer.AddressLine2 = r.FormValue("emp_addr2")
	s.Employer.City = r.FormValue("emp_city")
	s.Employer.State = r.FormValue("emp_state")
	s.Employer.ZIP = r.FormValue("emp_zip")
	s.Employer.ZIPExtension = r.FormValue("emp_zip_ext")
	s.Employer.EmploymentCode = r.FormValue("employment_code")
	s.Employer.KindOfEmployer = r.FormValue("kind_of_employer")
	s.Employer.ContactName = r.FormValue("employer_contact_name")
	s.Employer.ContactPhone = stripNonDigits(r.FormValue("employer_contact_phone"))
	s.Employer.ContactEmail = r.FormValue("employer_contact_email")
	s.Employer.TaxYear = r.FormValue("tax_year")
	s.Notes = r.FormValue("notes")
	if s.Employer.TaxYear == "" {
		supported := h.gen.SupportedYears()
		s.Employer.TaxYear = supported[len(supported)-1].Year
	}
	if err := h.repo.UpdateSubmission(r.Context(), s); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	render(w, r, templates.SubmissionHeader(s))
}

func (h *Handler) deleteSubmission(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		http.Error(w, "invalid id", 400)
		return
	}
	if err := h.repo.DeleteSubmission(r.Context(), id); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Header().Set("HX-Redirect", "/")
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) addEmployee(w http.ResponseWriter, r *http.Request) {
	subID, err := pathID(r, "id")
	if err != nil {
		http.Error(w, "invalid id", 400)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	e := parseEmployeeForm(r)
	if err := h.repo.AddEmployee(r.Context(), subID, e); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	s, err := h.repo.GetSubmission(r.Context(), subID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	render(w, r, templates.EmployeeList(s))
}

// editEmployeeForm renders the inline edit form for a single employee card.
func (h *Handler) editEmployeeForm(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		http.Error(w, "invalid id", 400)
		return
	}
	e, err := h.repo.GetEmployee(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	render(w, r, templates.EmployeeEditForm(e))
}

// getEmployeeCard renders just the read-only card for a single employee (used by cancel).
func (h *Handler) getEmployeeCard(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		http.Error(w, "invalid id", 400)
		return
	}
	e, err := h.repo.GetEmployee(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	render(w, r, templates.EmployeeCard(*e, e.SubmissionID))
}

// updateEmployee handles PUT /employees/{id} and renders the updated card.
func (h *Handler) updateEmployee(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		http.Error(w, "invalid id", 400)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	// Fetch first to preserve SubmissionID and CreatedAt.
	existing, err := h.repo.GetEmployee(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	e := parseEmployeeForm(r)
	e.ID = existing.ID
	e.SubmissionID = existing.SubmissionID
	e.CreatedAt = existing.CreatedAt
	if err := h.repo.UpdateEmployee(r.Context(), e); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	render(w, r, templates.EmployeeCard(*e, e.SubmissionID))
}

// parseEmployeeForm reads all employee correction fields from an HTTP form request
// and returns a populated EmployeeRecord.  ID, SubmissionID, and CreatedAt are
// zero-valued and must be filled in by the caller.
func parseEmployeeForm(r *http.Request) *domain.EmployeeRecord {
	parseBoolPtr := func(name string) *bool {
		v := r.FormValue(name)
		if v == "" {
			return nil
		}
		b := v == "1"
		return &b
	}
	return &domain.EmployeeRecord{
		SSN:         stripDashes(r.FormValue("ssn")),
		OriginalSSN: stripDashes(r.FormValue("original_ssn")),
		FirstName:   r.FormValue("first_name"),
		MiddleName:  r.FormValue("middle_name"),
		LastName:    r.FormValue("last_name"),
		Suffix:      r.FormValue("suffix"),
		// Name correction fields (only populated when correcting a previously wrong name)
		OriginalFirstName:  r.FormValue("orig_first_name"),
		OriginalMiddleName: r.FormValue("orig_middle_name"),
		OriginalLastName:   r.FormValue("orig_last_name"),
		OriginalSuffix:     r.FormValue("orig_suffix"),
		AddressLine1: r.FormValue("emp_addr1"),
		AddressLine2: r.FormValue("emp_addr2"),
		City:         r.FormValue("emp_city"),
		State:        r.FormValue("emp_state"),
		ZIP:          r.FormValue("emp_zip"),
		ZIPExtension: r.FormValue("emp_zip_ext"),
		OriginalStateCode:     strings.ToUpper(strings.TrimSpace(r.FormValue("orig_state_code"))),
		CorrectStateCode:      strings.ToUpper(strings.TrimSpace(r.FormValue("corr_state_code"))),
		OriginalStateIDNumber: r.FormValue("orig_state_id"),
		CorrectStateIDNumber:  r.FormValue("corr_state_id"),
		OriginalLocalityName:  r.FormValue("orig_locality_name"),
		CorrectLocalityName:   r.FormValue("corr_locality_name"),
		Amounts: domain.MonetaryAmounts{
			// Boxes 1–7
			OriginalWagesTipsOther:      parseCents(r.FormValue("orig_wages")),
			CorrectWagesTipsOther:       parseCents(r.FormValue("corr_wages")),
			OriginalFederalIncomeTax:    parseCents(r.FormValue("orig_fed_tax")),
			CorrectFederalIncomeTax:     parseCents(r.FormValue("corr_fed_tax")),
			OriginalSocialSecurityWages: parseCents(r.FormValue("orig_ss_wages")),
			CorrectSocialSecurityWages:  parseCents(r.FormValue("corr_ss_wages")),
			OriginalSocialSecurityTax:   parseCents(r.FormValue("orig_ss_tax")),
			CorrectSocialSecurityTax:    parseCents(r.FormValue("corr_ss_tax")),
			OriginalMedicareWages:       parseCents(r.FormValue("orig_med_wages")),
			CorrectMedicareWages:        parseCents(r.FormValue("corr_med_wages")),
			OriginalMedicareTax:         parseCents(r.FormValue("orig_med_tax")),
			CorrectMedicareTax:          parseCents(r.FormValue("corr_med_tax")),
			OriginalSocialSecurityTips:  parseCents(r.FormValue("orig_ss_tips")),
			CorrectSocialSecurityTips:   parseCents(r.FormValue("corr_ss_tips")),
			// Box 8 — Allocated Tips
			OriginalAllocatedTips: parseCents(r.FormValue("orig_alloc_tips")),
			CorrectAllocatedTips:  parseCents(r.FormValue("corr_alloc_tips")),
			// Box 10 — Dependent Care Benefits
			OriginalDependentCare: parseCents(r.FormValue("orig_dep_care")),
			CorrectDependentCare:  parseCents(r.FormValue("corr_dep_care")),
			// Box 11 — Nonqualified Plans
			OriginalNonqualPlan457:       parseCents(r.FormValue("orig_nonqual_457")),
			CorrectNonqualPlan457:        parseCents(r.FormValue("corr_nonqual_457")),
			OriginalNonqualNotSection457: parseCents(r.FormValue("orig_nonqual_not457")),
			CorrectNonqualNotSection457:  parseCents(r.FormValue("corr_nonqual_not457")),
			// Box 12 codes
			OriginalCode401k:         parseCents(r.FormValue("orig_code_d")),
			CorrectCode401k:          parseCents(r.FormValue("corr_code_d")),
			OriginalCode403b:         parseCents(r.FormValue("orig_code_e")),
			CorrectCode403b:          parseCents(r.FormValue("corr_code_e")),
			OriginalCode457bGovt:     parseCents(r.FormValue("orig_code_g")),
			CorrectCode457bGovt:      parseCents(r.FormValue("corr_code_g")),
			OriginalCodeW_HSA:        parseCents(r.FormValue("orig_code_w")),
			CorrectCodeW_HSA:         parseCents(r.FormValue("corr_code_w")),
			OriginalCodeAA_Roth401k:  parseCents(r.FormValue("orig_code_aa")),
			CorrectCodeAA_Roth401k:   parseCents(r.FormValue("corr_code_aa")),
			OriginalCodeBB_Roth403b:  parseCents(r.FormValue("orig_code_bb")),
			CorrectCodeBB_Roth403b:   parseCents(r.FormValue("corr_code_bb")),
			OriginalCodeDD_EmpHealth: parseCents(r.FormValue("orig_code_dd")),
			CorrectCodeDD_EmpHealth:  parseCents(r.FormValue("corr_code_dd")),
			// Boxes 16–19 — State / Local
			OriginalStateWages:     parseCents(r.FormValue("orig_state_wages")),
			CorrectStateWages:      parseCents(r.FormValue("corr_state_wages")),
			OriginalStateIncomeTax: parseCents(r.FormValue("orig_state_tax")),
			CorrectStateIncomeTax:  parseCents(r.FormValue("corr_state_tax")),
			OriginalLocalWages:     parseCents(r.FormValue("orig_local_wages")),
			CorrectLocalWages:      parseCents(r.FormValue("corr_local_wages")),
			OriginalLocalIncomeTax: parseCents(r.FormValue("orig_local_tax")),
			CorrectLocalIncomeTax:  parseCents(r.FormValue("corr_local_tax")),
		},
		Box13: domain.Box13Flags{
			OrigStatutoryEmployee:    parseBoolPtr("orig_statutory_emp"),
			CorrectStatutoryEmployee: parseBoolPtr("corr_statutory_emp"),
			OrigRetirementPlan:       parseBoolPtr("orig_retirement_plan"),
			CorrectRetirementPlan:    parseBoolPtr("corr_retirement_plan"),
			OrigThirdPartySickPay:    parseBoolPtr("orig_third_party_sick"),
			CorrectThirdPartySickPay: parseBoolPtr("corr_third_party_sick"),
		},
	}
}

func (h *Handler) deleteEmployee(w http.ResponseWriter, r *http.Request) {
	empID, err := pathID(r, "id")
	if err != nil {
		http.Error(w, "invalid id", 400)
		return
	}
	subIDStr := r.URL.Query().Get("sub")
	subID, _ := strconv.ParseInt(subIDStr, 10, 64)

	if err := h.repo.DeleteEmployee(r.Context(), empID); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if subID > 0 {
		s, err := h.repo.GetSubmission(r.Context(), subID)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		render(w, r, templates.EmployeeList(s))
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) generateFile(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		http.Error(w, "invalid id", 400)
		return
	}
	s, err := h.repo.GetSubmission(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if len(s.Employees) == 0 {
		http.Error(w, "no employees in submission", 400)
		return
	}
	var buf bytes.Buffer
	if err := h.gen.Generate(context.Background(), s, &buf); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	filename := fmt.Sprintf("W2C_%s_%s.txt", s.Employer.EIN, time.Now().Format("20060102"))
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Write(buf.Bytes())
}

func (h *Handler) generatePDF(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		http.Error(w, "invalid id", 400)
		return
	}
	s, err := h.repo.GetSubmission(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if len(s.Employees) == 0 {
		http.Error(w, "no employees in submission", 400)
		return
	}
	var buf bytes.Buffer
	if err := pdf.GeneratePDF(s, &buf); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	filename := fmt.Sprintf("W2C_%s_%s_report.pdf", s.Employer.EIN, time.Now().Format("20060102"))
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Write(buf.Bytes())
}

// render writes a templ component to the response.
func render(w http.ResponseWriter, r *http.Request, c templ.Component) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := c.Render(r.Context(), w); err != nil {
		http.Error(w, err.Error(), 500)
	}
}

func pathID(r *http.Request, key string) (int64, error) {
	return strconv.ParseInt(r.PathValue(key), 10, 64)
}

func stripDashes(s string) string {
	return strings.ReplaceAll(s, "-", "")
}

func stripNonDigits(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func parseCents(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	parts := strings.SplitN(s, ".", 2)
	dollars, _ := strconv.ParseInt(parts[0], 10, 64)
	var cents int64
	if len(parts) == 2 {
		c := parts[1]
		if len(c) == 1 {
			c += "0"
		} else if len(c) > 2 {
			c = c[:2]
		}
		cents, _ = strconv.ParseInt(c, 10, 64)
	}
	return dollars*100 + cents
}
