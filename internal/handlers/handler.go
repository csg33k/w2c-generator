package handlers

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/csg33k/w2c-generator/internal/domain"
	"github.com/csg33k/w2c-generator/internal/ports"
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
	mux.HandleFunc("GET /submissions", h.listSubmissions)
	mux.HandleFunc("POST /submissions", h.createSubmission)
	mux.HandleFunc("GET /submissions/{id}", h.viewSubmission)
	mux.HandleFunc("DELETE /submissions/{id}", h.deleteSubmission)
	mux.HandleFunc("POST /submissions/{id}/employees", h.addEmployee)
	mux.HandleFunc("DELETE /employees/{id}", h.deleteEmployee)
	mux.HandleFunc("GET /submissions/{id}/generate", h.generateFile)
	return mux
}

func (h *Handler) index(w http.ResponseWriter, r *http.Request) {
	submissions, err := h.repo.ListSubmissions(r.Context())
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	renderIndex(w, submissions)
}

func (h *Handler) listSubmissions(w http.ResponseWriter, r *http.Request) {
	submissions, err := h.repo.ListSubmissions(r.Context())
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	renderSubmissionList(w, submissions)
}

func (h *Handler) createSubmission(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	s := &domain.Submission{
		Employer: domain.EmployerRecord{
			EIN:            stripDashes(r.FormValue("ein")),
			Name:           r.FormValue("employer_name"),
			AddressLine1:   r.FormValue("addr1"),
			AddressLine2:   r.FormValue("addr2"),
			City:           r.FormValue("city"),
			State:          r.FormValue("state"),
			ZIP:            r.FormValue("zip"),
			ZIPExtension:   r.FormValue("zip_ext"),
			AgentIndicator: "0",
			TaxYear:        domain.TaxYear2021,
		},
		Notes: r.FormValue("notes"),
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
	renderSubmissionDetail(w, s)
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
	// HTMX: redirect to home after delete
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
	e := &domain.EmployeeRecord{
		SSN:          stripDashes(r.FormValue("ssn")),
		OriginalSSN:  stripDashes(r.FormValue("original_ssn")),
		FirstName:    r.FormValue("first_name"),
		MiddleName:   r.FormValue("middle_name"),
		LastName:     r.FormValue("last_name"),
		Suffix:       r.FormValue("suffix"),
		AddressLine1: r.FormValue("addr1"),
		AddressLine2: r.FormValue("addr2"),
		City:         r.FormValue("city"),
		State:        r.FormValue("state"),
		ZIP:          r.FormValue("zip"),
		ZIPExtension: r.FormValue("zip_ext"),
		Amounts: domain.MonetaryAmounts{
			OriginalWagesTipsOther:      parseCents(r.FormValue("orig_wages")),
			CorrectWagesTipsOther:       parseCents(r.FormValue("corr_wages")),
			OriginalSocialSecurityWages: parseCents(r.FormValue("orig_ss_wages")),
			CorrectSocialSecurityWages:  parseCents(r.FormValue("corr_ss_wages")),
			OriginalMedicareWages:       parseCents(r.FormValue("orig_med_wages")),
			CorrectMedicareWages:        parseCents(r.FormValue("corr_med_wages")),
			OriginalFederalIncomeTax:    parseCents(r.FormValue("orig_fed_tax")),
			CorrectFederalIncomeTax:     parseCents(r.FormValue("corr_fed_tax")),
			OriginalSocialSecurityTax:   parseCents(r.FormValue("orig_ss_tax")),
			CorrectSocialSecurityTax:    parseCents(r.FormValue("corr_ss_tax")),
			OriginalMedicareTax:         parseCents(r.FormValue("orig_med_tax")),
			CorrectMedicareTax:          parseCents(r.FormValue("corr_med_tax")),
		},
	}
	if err := h.repo.AddEmployee(r.Context(), subID, e); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	// Re-fetch and re-render employee list fragment
	s, err := h.repo.GetSubmission(r.Context(), subID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	renderEmployeeList(w, s)
}

func (h *Handler) deleteEmployee(w http.ResponseWriter, r *http.Request) {
	empID, err := pathID(r, "id")
	if err != nil {
		http.Error(w, "invalid id", 400)
		return
	}
	// Get submission id from query param for re-render
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
		renderEmployeeList(w, s)
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

	filename := fmt.Sprintf("W2C_%s_%s.txt", stripDashes(s.Employer.EIN), time.Now().Format("20060102"))
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Write(buf.Bytes())
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func pathID(r *http.Request, key string) (int64, error) {
	return strconv.ParseInt(r.PathValue(key), 10, 64)
}

func stripDashes(s string) string {
	return strings.ReplaceAll(s, "-", "")
}

// parseCents converts a dollar string like "1234.56" to cents (123456).
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

func centsToDisplay(cents int64) string {
	return fmt.Sprintf("%.2f", float64(cents)/100)
}
