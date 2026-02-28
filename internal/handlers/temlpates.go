package handlers

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/csg33k/w2c-generator/internal/domain"
)

// NOTE: In a full project these would be .templ files compiled via `templ generate`.
// They are inlined here as html/template for zero-dependency portability.
// Swap to templ by moving each block to its own .templ file and calling component(data).Render(ctx, w).

var baseTmpl = template.Must(template.New("base").Funcs(template.FuncMap{
	"cents": centsToDisplay,
	"seq":   func(i int) int { return i + 1 },
}).Parse(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>W-2c EFW2C Generator · Tax Year 2021</title>
<script src="https://unpkg.com/htmx.org@1.9.12"></script>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=IBM+Plex+Mono:wght@400;500;600&family=IBM+Plex+Sans:wght@300;400;500;600&display=swap" rel="stylesheet">
<script src="https://cdn.tailwindcss.com"></script>
<style>
  :root {
    --ink: #0d1117;
    --paper: #f5f0e8;
    --ledger: #e8e0cc;
    --accent: #c0392b;
    --accent2: #2c6e49;
    --muted: #6b5e4e;
    --rule: #b8a898;
  }
  * { box-sizing: border-box; }
  body {
    background: var(--paper);
    color: var(--ink);
    font-family: 'IBM Plex Sans', sans-serif;
    background-image:
      repeating-linear-gradient(0deg, transparent, transparent 27px, var(--rule) 27px, var(--rule) 28px);
    min-height: 100vh;
  }
  .mono { font-family: 'IBM Plex Mono', monospace; }
  .stamp {
    display: inline-block;
    border: 3px solid var(--accent);
    color: var(--accent);
    font-family: 'IBM Plex Mono', monospace;
    font-weight: 600;
    letter-spacing: 0.15em;
    padding: 2px 10px;
    transform: rotate(-2deg);
    font-size: 0.7rem;
  }
  .card {
    background: rgba(255,255,255,0.7);
    border: 1px solid var(--ledger);
    border-left: 4px solid var(--ink);
  }
  .field-label {
    font-family: 'IBM Plex Mono', monospace;
    font-size: 0.6rem;
    font-weight: 600;
    letter-spacing: 0.1em;
    text-transform: uppercase;
    color: var(--muted);
    display: block;
    margin-bottom: 2px;
  }
  input, select, textarea {
    background: white;
    border: 1px solid var(--rule);
    border-bottom: 2px solid var(--ink);
    padding: 6px 8px;
    font-family: 'IBM Plex Mono', monospace;
    font-size: 0.85rem;
    width: 100%;
    outline: none;
    transition: border-color 0.15s;
  }
  input:focus, select:focus { border-bottom-color: var(--accent); }
  .btn {
    font-family: 'IBM Plex Mono', monospace;
    font-weight: 600;
    font-size: 0.8rem;
    letter-spacing: 0.08em;
    padding: 8px 18px;
    border: 2px solid var(--ink);
    cursor: pointer;
    transition: all 0.15s;
    text-transform: uppercase;
  }
  .btn-primary { background: var(--ink); color: white; }
  .btn-primary:hover { background: var(--accent); border-color: var(--accent); }
  .btn-danger { background: white; color: var(--accent); border-color: var(--accent); }
  .btn-danger:hover { background: var(--accent); color: white; }
  .btn-success { background: var(--accent2); color: white; border-color: var(--accent2); }
  .btn-success:hover { filter: brightness(1.1); }
  .divider {
    border: none; border-top: 2px solid var(--ink);
    margin: 24px 0;
  }
  .section-header {
    font-family: 'IBM Plex Mono', monospace;
    font-size: 0.7rem;
    font-weight: 600;
    letter-spacing: 0.18em;
    text-transform: uppercase;
    color: var(--muted);
    border-bottom: 1px solid var(--rule);
    padding-bottom: 4px;
    margin-bottom: 16px;
  }
  .emp-row { border-bottom: 1px solid var(--ledger); }
  .emp-row:last-child { border-bottom: none; }
  .amount-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 4px 16px;
  }
  .amount-row {
    display: contents;
  }
  .box-label {
    font-family: 'IBM Plex Mono', monospace;
    font-size: 0.65rem;
    font-weight: 600;
    color: var(--muted);
    padding: 2px 4px;
    background: var(--ledger);
    display: inline-block;
    margin-bottom: 2px;
  }
  .htmx-indicator { opacity: 0; transition: opacity 0.2s; }
  .htmx-request .htmx-indicator { opacity: 1; }
</style>
</head>
<body>
<div style="max-width:1100px;margin:0 auto;padding:32px 24px;">

<!-- Header -->
<div style="display:flex;align-items:flex-start;justify-content:space-between;margin-bottom:32px;">
  <div>
    <div style="font-family:'IBM Plex Mono',monospace;font-size:0.65rem;letter-spacing:0.2em;color:var(--muted);margin-bottom:4px;">
      INTERNAL REVENUE SERVICE · SOCIAL SECURITY ADMINISTRATION
    </div>
    <h1 style="font-family:'IBM Plex Mono',monospace;font-size:1.6rem;font-weight:600;letter-spacing:-0.02em;margin:0;">
      W‑2c Correction Generator
    </h1>
    <div style="font-size:0.85rem;color:var(--muted);margin-top:4px;">
      EFW2C Fixed-Width Submission Format · Tax Year <strong>2021</strong>
    </div>
  </div>
  <div style="text-align:right;">
    <div class="stamp">TY 2021</div>
    <div style="font-family:'IBM Plex Mono',monospace;font-size:0.65rem;color:var(--muted);margin-top:8px;">SSA Pub. 42-007</div>
  </div>
</div>

{{template "content" .}}

<div style="margin-top:48px;padding-top:16px;border-top:1px solid var(--rule);font-family:'IBM Plex Mono',monospace;font-size:0.6rem;color:var(--muted);text-align:center;">
  W-2c EFW2C GENERATOR · FOR INTERNAL USE · TAX YEAR 2021 ONLY
</div>
</div>
</body>
</html>`))

// Index page
var indexTmpl = template.Must(template.Must(baseTmpl.Clone()).Parse(`
{{define "content"}}
<div style="display:grid;grid-template-columns:1fr 1fr;gap:32px;align-items:start;">

<!-- New Submission Form -->
<div class="card" style="padding:24px;">
  <div class="section-header">New Submission</div>
  <form hx-post="/submissions" hx-target="body" hx-push-url="true">

    <div class="section-header" style="margin-top:0;">Employer Information</div>
    <div style="display:grid;grid-template-columns:1fr 1fr;gap:12px;">
      <div style="grid-column:1/-1;">
        <label class="field-label">Employer EIN *</label>
        <input type="text" name="ein" placeholder="12-3456789" required maxlength="10" class="mono">
      </div>
      <div style="grid-column:1/-1;">
        <label class="field-label">Employer Name *</label>
        <input type="text" name="employer_name" placeholder="ACME CORPORATION" required maxlength="39">
      </div>
      <div style="grid-column:1/-1;">
        <label class="field-label">Address Line 1</label>
        <input type="text" name="addr1" placeholder="123 MAIN ST" maxlength="39">
      </div>
      <div style="grid-column:1/-1;">
        <label class="field-label">Address Line 2</label>
        <input type="text" name="addr2" placeholder="SUITE 400" maxlength="39">
      </div>
      <div style="grid-column:1/-1;">
        <label class="field-label">City</label>
        <input type="text" name="city" placeholder="SPRINGFIELD" maxlength="39">
      </div>
      <div>
        <label class="field-label">State</label>
        <input type="text" name="state" placeholder="IL" maxlength="2">
      </div>
      <div>
        <label class="field-label">ZIP</label>
        <input type="text" name="zip" placeholder="62701" maxlength="5">
      </div>
    </div>

    <hr class="divider">
    <div>
      <label class="field-label">Notes (internal)</label>
      <textarea name="notes" rows="2" style="resize:vertical;" placeholder="Optional internal notes..."></textarea>
    </div>

    <div style="margin-top:16px;display:flex;justify-content:flex-end;">
      <button type="submit" class="btn btn-primary">
        CREATE SUBMISSION →
      </button>
    </div>
  </form>
</div>

<!-- Existing Submissions -->
<div>
  <div class="section-header">Existing Submissions</div>
  <div id="submission-list">
    {{template "submission-list-items" .Submissions}}
  </div>
</div>

</div>
{{end}}`))

var submissionListFrag = template.Must(template.New("sub-list-frag").Funcs(template.FuncMap{
	"cents": centsToDisplay,
}).Parse(`{{define "submission-list-items"}}
{{if not .}}
<div style="font-family:'IBM Plex Mono',monospace;font-size:0.8rem;color:var(--muted);padding:16px;text-align:center;">
  No submissions yet.
</div>
{{else}}
{{range .}}
<div class="card emp-row" style="padding:14px 18px;margin-bottom:8px;display:flex;justify-content:space-between;align-items:center;">
  <div>
    <div style="font-family:'IBM Plex Mono',monospace;font-weight:600;font-size:0.9rem;">{{.Employer.Name}}</div>
    <div style="font-size:0.75rem;color:var(--muted);margin-top:2px;">EIN: {{.Employer.EIN}} · {{.CreatedAt.Format "Jan 02, 2006"}}</div>
    {{if .Notes}}<div style="font-size:0.72rem;color:var(--muted);margin-top:2px;font-style:italic;">{{.Notes}}</div>{{end}}
  </div>
  <a href="/submissions/{{.ID}}" style="text-decoration:none;">
    <button class="btn btn-primary" style="padding:6px 14px;font-size:0.7rem;">OPEN →</button>
  </a>
</div>
{{end}}
{{end}}
{{end}}`))

var detailTmpl = template.Must(template.New("detail").Funcs(template.FuncMap{
	"cents": centsToDisplay,
}).Parse(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>W-2c · {{.Employer.Name}}</title>
<script src="https://unpkg.com/htmx.org@1.9.12"></script>
<link href="https://fonts.googleapis.com/css2?family=IBM+Plex+Mono:wght@400;500;600&family=IBM+Plex+Sans:wght@300;400;500;600&display=swap" rel="stylesheet">
<script src="https://cdn.tailwindcss.com"></script>
<style>
  :root{--ink:#0d1117;--paper:#f5f0e8;--ledger:#e8e0cc;--accent:#c0392b;--accent2:#2c6e49;--muted:#6b5e4e;--rule:#b8a898;}
  *{box-sizing:border-box;}
  body{background:var(--paper);color:var(--ink);font-family:'IBM Plex Sans',sans-serif;background-image:repeating-linear-gradient(0deg,transparent,transparent 27px,var(--rule) 27px,var(--rule) 28px);min-height:100vh;}
  .mono{font-family:'IBM Plex Mono',monospace;}
  .card{background:rgba(255,255,255,0.7);border:1px solid var(--ledger);border-left:4px solid var(--ink);}
  .field-label{font-family:'IBM Plex Mono',monospace;font-size:0.6rem;font-weight:600;letter-spacing:0.1em;text-transform:uppercase;color:var(--muted);display:block;margin-bottom:2px;}
  input,select,textarea{background:white;border:1px solid var(--rule);border-bottom:2px solid var(--ink);padding:6px 8px;font-family:'IBM Plex Mono',monospace;font-size:0.85rem;width:100%;outline:none;transition:border-color 0.15s;}
  input:focus{border-bottom-color:var(--accent);}
  .btn{font-family:'IBM Plex Mono',monospace;font-weight:600;font-size:0.8rem;letter-spacing:0.08em;padding:8px 18px;border:2px solid var(--ink);cursor:pointer;transition:all 0.15s;text-transform:uppercase;}
  .btn-primary{background:var(--ink);color:white;}
  .btn-primary:hover{background:var(--accent);border-color:var(--accent);}
  .btn-danger{background:white;color:var(--accent);border-color:var(--accent);}
  .btn-danger:hover{background:var(--accent);color:white;}
  .btn-success{background:var(--accent2);color:white;border-color:var(--accent2);}
  .btn-success:hover{filter:brightness(1.1);}
  .divider{border:none;border-top:2px solid var(--ink);margin:20px 0;}
  .section-header{font-family:'IBM Plex Mono',monospace;font-size:0.7rem;font-weight:600;letter-spacing:0.18em;text-transform:uppercase;color:var(--muted);border-bottom:1px solid var(--rule);padding-bottom:4px;margin-bottom:16px;}
  .box-label{font-family:'IBM Plex Mono',monospace;font-size:0.6rem;font-weight:600;color:var(--muted);padding:2px 4px;background:var(--ledger);display:inline-block;margin-bottom:3px;}
  .stamp{display:inline-block;border:3px solid var(--accent);color:var(--accent);font-family:'IBM Plex Mono',monospace;font-weight:600;letter-spacing:0.15em;padding:2px 10px;transform:rotate(-2deg);font-size:0.7rem;}
</style>
</head>
<body>
<div style="max-width:1200px;margin:0 auto;padding:32px 24px;">

<div style="display:flex;align-items:center;gap:16px;margin-bottom:24px;">
  <a href="/" style="font-family:'IBM Plex Mono',monospace;font-size:0.75rem;color:var(--muted);text-decoration:none;">← ALL SUBMISSIONS</a>
  <div class="stamp">TY 2021</div>
</div>

<div style="display:flex;justify-content:space-between;align-items:flex-start;margin-bottom:24px;">
  <div>
    <h1 style="font-family:'IBM Plex Mono',monospace;font-size:1.4rem;font-weight:600;margin:0;">{{.Employer.Name}}</h1>
    <div style="font-size:0.85rem;color:var(--muted);margin-top:4px;">EIN: <span class="mono">{{.Employer.EIN}}</span> · {{len .Employees}} employee correction(s)</div>
  </div>
  <div style="display:flex;gap:10px;">
    <button class="btn btn-success"
      hx-get="/submissions/{{.ID}}/generate"
      hx-swap="none"
      onclick="downloadFile({{.ID}})"
      style="padding:10px 20px;">
      ⬇ GENERATE EFW2C FILE
    </button>
    <button class="btn btn-danger"
      hx-delete="/submissions/{{.ID}}"
      hx-confirm="Delete this entire submission?"
      style="padding:10px 16px;">
      DELETE
    </button>
  </div>
</div>

<div style="display:grid;grid-template-columns:350px 1fr;gap:28px;align-items:start;">

<!-- Add Employee Form -->
<div class="card" style="padding:22px;">
  <div class="section-header">Add Employee Correction</div>
  <form hx-post="/submissions/{{.ID}}/employees"
        hx-target="#employee-list"
        hx-swap="innerHTML"
        hx-on::after-request="this.reset()">

    <div class="section-header">Identity</div>
    <div style="display:grid;gap:10px;">
      <div>
        <label class="field-label">Correct SSN *</label>
        <input type="text" name="ssn" placeholder="123-45-6789" required maxlength="11" class="mono">
      </div>
      <div>
        <label class="field-label">Originally Reported SSN (if different)</label>
        <input type="text" name="original_ssn" placeholder="Leave blank if unchanged" maxlength="11" class="mono">
      </div>
      <div style="display:grid;grid-template-columns:1fr 1fr;gap:8px;">
        <div>
          <label class="field-label">First Name</label>
          <input type="text" name="first_name" maxlength="12">
        </div>
        <div>
          <label class="field-label">Last Name</label>
          <input type="text" name="last_name" maxlength="15">
        </div>
      </div>
    </div>

    <hr class="divider">
    <div class="section-header">Wages &amp; Compensation</div>
    <div style="display:grid;gap:8px;">
      <div style="display:grid;grid-template-columns:1fr 1fr;gap:8px;">
        <div>
          <div class="box-label">Box 1 — Original</div>
          <label class="field-label">Wages, Tips (orig.)</label>
          <input type="number" name="orig_wages" step="0.01" min="0" placeholder="0.00" class="mono">
        </div>
        <div>
          <div class="box-label">Box 1 — Correct</div>
          <label class="field-label">Wages, Tips (corr.)</label>
          <input type="number" name="corr_wages" step="0.01" min="0" placeholder="0.00" class="mono">
        </div>
      </div>
      <div style="display:grid;grid-template-columns:1fr 1fr;gap:8px;">
        <div>
          <div class="box-label">Box 3 — Original</div>
          <label class="field-label">SS Wages (orig.)</label>
          <input type="number" name="orig_ss_wages" step="0.01" min="0" placeholder="0.00" class="mono">
        </div>
        <div>
          <div class="box-label">Box 3 — Correct</div>
          <label class="field-label">SS Wages (corr.)</label>
          <input type="number" name="corr_ss_wages" step="0.01" min="0" placeholder="0.00" class="mono">
        </div>
      </div>
      <div style="display:grid;grid-template-columns:1fr 1fr;gap:8px;">
        <div>
          <div class="box-label">Box 5 — Original</div>
          <label class="field-label">Medicare Wages (orig.)</label>
          <input type="number" name="orig_med_wages" step="0.01" min="0" placeholder="0.00" class="mono">
        </div>
        <div>
          <div class="box-label">Box 5 — Correct</div>
          <label class="field-label">Medicare Wages (corr.)</label>
          <input type="number" name="corr_med_wages" step="0.01" min="0" placeholder="0.00" class="mono">
        </div>
      </div>
    </div>

    <hr class="divider">
    <div class="section-header">Tax Withholdings</div>
    <div style="display:grid;gap:8px;">
      <div style="display:grid;grid-template-columns:1fr 1fr;gap:8px;">
        <div>
          <div class="box-label">Box 2 — Original</div>
          <label class="field-label">Fed. Income Tax (orig.)</label>
          <input type="number" name="orig_fed_tax" step="0.01" min="0" placeholder="0.00" class="mono">
        </div>
        <div>
          <div class="box-label">Box 2 — Correct</div>
          <label class="field-label">Fed. Income Tax (corr.)</label>
          <input type="number" name="corr_fed_tax" step="0.01" min="0" placeholder="0.00" class="mono">
        </div>
      </div>
      <div style="display:grid;grid-template-columns:1fr 1fr;gap:8px;">
        <div>
          <div class="box-label">Box 4 — Original</div>
          <label class="field-label">SS Tax (orig.)</label>
          <input type="number" name="orig_ss_tax" step="0.01" min="0" placeholder="0.00" class="mono">
        </div>
        <div>
          <div class="box-label">Box 4 — Correct</div>
          <label class="field-label">SS Tax (corr.)</label>
          <input type="number" name="corr_ss_tax" step="0.01" min="0" placeholder="0.00" class="mono">
        </div>
      </div>
      <div style="display:grid;grid-template-columns:1fr 1fr;gap:8px;">
        <div>
          <div class="box-label">Box 6 — Original</div>
          <label class="field-label">Medicare Tax (orig.)</label>
          <input type="number" name="orig_med_tax" step="0.01" min="0" placeholder="0.00" class="mono">
        </div>
        <div>
          <div class="box-label">Box 6 — Correct</div>
          <label class="field-label">Medicare Tax (corr.)</label>
          <input type="number" name="corr_med_tax" step="0.01" min="0" placeholder="0.00" class="mono">
        </div>
      </div>
    </div>

    <div style="margin-top:16px;display:flex;justify-content:flex-end;">
      <button type="submit" class="btn btn-primary">ADD EMPLOYEE +</button>
    </div>
  </form>
</div>

<!-- Employee List -->
<div>
  <div class="section-header">Employee Corrections ({{len .Employees}})</div>
  <div id="employee-list">
    {{template "employee-list-rows" .}}
  </div>
</div>

</div>
</div>

<script>
function downloadFile(id) {
  window.location.href = '/submissions/' + id + '/generate';
}
</script>
</body>
</html>
{{define "employee-list-rows"}}
{{if not .Employees}}
<div class="card" style="padding:20px;text-align:center;font-family:'IBM Plex Mono',monospace;font-size:0.8rem;color:var(--muted);">
  No employees added yet. Use the form to add corrections.
</div>
{{else}}
{{range .Employees}}
<div class="card" style="padding:16px 20px;margin-bottom:10px;">
  <div style="display:flex;justify-content:space-between;align-items:flex-start;">
    <div>
      <div style="font-family:'IBM Plex Mono',monospace;font-weight:600;font-size:1rem;">
        {{.LastName}}, {{.FirstName}}
        {{if .MiddleName}}<span style="color:var(--muted);font-size:0.85rem;"> {{.MiddleName}}</span>{{end}}
      </div>
      <div style="font-size:0.75rem;color:var(--muted);margin-top:2px;">
        SSN: <span class="mono">{{.SSN}}</span>
        {{if .OriginalSSN}} · Orig SSN: <span class="mono">{{.OriginalSSN}}</span>{{end}}
      </div>
    </div>
    <button class="btn btn-danger" style="padding:4px 12px;font-size:0.7rem;"
      hx-delete="/employees/{{.ID}}?sub={{.SubmissionID}}"
      hx-target="#employee-list"
      hx-swap="innerHTML"
      hx-confirm="Remove this employee from the submission?">
      REMOVE
    </button>
  </div>
  <div style="margin-top:12px;display:grid;grid-template-columns:repeat(3,1fr);gap:8px;font-size:0.75rem;">
    <div style="background:var(--ledger);padding:8px;border-left:3px solid var(--ink);">
      <div class="box-label">BOX 1 — WAGES/TIPS</div>
      <div>Orig: <span class="mono">${{cents .Amounts.OriginalWagesTipsOther}}</span></div>
      <div>Corr: <span class="mono" style="color:var(--accent2);font-weight:600;">${{cents .Amounts.CorrectWagesTipsOther}}</span></div>
    </div>
    <div style="background:var(--ledger);padding:8px;border-left:3px solid var(--ink);">
      <div class="box-label">BOX 2 — FED INCOME TAX</div>
      <div>Orig: <span class="mono">${{cents .Amounts.OriginalFederalIncomeTax}}</span></div>
      <div>Corr: <span class="mono" style="color:var(--accent2);font-weight:600;">${{cents .Amounts.CorrectFederalIncomeTax}}</span></div>
    </div>
    <div style="background:var(--ledger);padding:8px;border-left:3px solid var(--ink);">
      <div class="box-label">BOX 3 — SS WAGES</div>
      <div>Orig: <span class="mono">${{cents .Amounts.OriginalSocialSecurityWages}}</span></div>
      <div>Corr: <span class="mono" style="color:var(--accent2);font-weight:600;">${{cents .Amounts.CorrectSocialSecurityWages}}</span></div>
    </div>
    <div style="background:var(--ledger);padding:8px;border-left:3px solid var(--ink);">
      <div class="box-label">BOX 4 — SS TAX</div>
      <div>Orig: <span class="mono">${{cents .Amounts.OriginalSocialSecurityTax}}</span></div>
      <div>Corr: <span class="mono" style="color:var(--accent2);font-weight:600;">${{cents .Amounts.CorrectSocialSecurityTax}}</span></div>
    </div>
    <div style="background:var(--ledger);padding:8px;border-left:3px solid var(--ink);">
      <div class="box-label">BOX 5 — MEDICARE WAGES</div>
      <div>Orig: <span class="mono">${{cents .Amounts.OriginalMedicareWages}}</span></div>
      <div>Corr: <span class="mono" style="color:var(--accent2);font-weight:600;">${{cents .Amounts.CorrectMedicareWages}}</span></div>
    </div>
    <div style="background:var(--ledger);padding:8px;border-left:3px solid var(--ink);">
      <div class="box-label">BOX 6 — MEDICARE TAX</div>
      <div>Orig: <span class="mono">${{cents .Amounts.OriginalMedicareTax}}</span></div>
      <div>Corr: <span class="mono" style="color:var(--accent2);font-weight:600;">${{cents .Amounts.CorrectMedicareTax}}</span></div>
    </div>
  </div>
</div>
{{end}}
{{end}}
{{end}}`))

// ---------------------------------------------------------------------------
// Render helpers
// ---------------------------------------------------------------------------

type indexData struct {
	Submissions []domain.Submission
}

func renderIndex(w http.ResponseWriter, submissions []domain.Submission) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	indexTmpl.ExecuteTemplate(w, "base", indexData{Submissions: submissions})
}

func renderSubmissionList(w http.ResponseWriter, submissions []domain.Submission) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	submissionListFrag.ExecuteTemplate(w, "submission-list-items", submissions)
}

func renderSubmissionDetail(w http.ResponseWriter, s *domain.Submission) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := detailTmpl.Execute(w, s); err != nil {
		fmt.Fprintf(w, "template error: %v", err)
	}
}

func renderEmployeeList(w http.ResponseWriter, s *domain.Submission) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	detailTmpl.ExecuteTemplate(w, "employee-list-rows", s)
}
