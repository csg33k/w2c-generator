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
	UpdateEmployee(ctx context.Context, e *domain.EmployeeRecord) error
	DeleteEmployee(ctx context.Context, id int64) error
}

// EFW2CGenerator defines the output generation port.
type EFW2CGenerator interface {
	// Generate writes the full EFW2C fixed-width file to w.
	Generate(ctx context.Context, s *domain.Submission, w io.Writer) error
}
