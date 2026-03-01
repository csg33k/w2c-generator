package ports

import (
	"context"
	"io"

	"github.com/csg33k/w2c-generator/internal/domain"
)

// SubmissionRepository defines persistence operations.
type SubmissionRepository interface {
	CreateSubmission(ctx context.Context, s *domain.Submission) error
	GetSubmission(ctx context.Context, id int64) (*domain.Submission, error)
	ListSubmissions(ctx context.Context) ([]domain.Submission, error)
	UpdateSubmission(ctx context.Context, s *domain.Submission) error
	DeleteSubmission(ctx context.Context, id int64) error

	AddEmployee(ctx context.Context, submissionID int64, e *domain.EmployeeRecord) error
	GetEmployee(ctx context.Context, id int64) (*domain.EmployeeRecord, error)
	UpdateEmployee(ctx context.Context, e *domain.EmployeeRecord) error
	DeleteEmployee(ctx context.Context, id int64) error
}

// EFW2CGenerator defines the output generation port.
type EFW2CGenerator interface {
	// Generate writes a complete EFW2C file for the submission.
	// The spec version is selected from s.Employer.TaxYear automatically.
	Generate(ctx context.Context, s *domain.Submission, w io.Writer) error

	// SupportedYears returns the tax years this generator can produce files for,
	// in ascending order, each with its SSA publication URL.
	SupportedYears() []domain.TaxYearInfo
}
