/*
Package service contains the application use-cases for the Commodore SDK.
This package imports only domain and ports - never Cobra or infrastructure.
*/
package service

import (
	"fmt"
	"strings"

	"github.com/OctalMesh/Commodore/internal/core/domain"
	"github.com/OctalMesh/Commodore/internal/core/ports"
)

// DoctorService checks that all required external tools are installed.
type DoctorService struct {
	checker ports.ToolChecker
}

// NewDoctorService constructs a DoctorService.
func NewDoctorService(checker ports.ToolChecker) *DoctorService {
	return &DoctorService{checker: checker}
}

/*
Check runs all tool checks and returns the full result list.
Returns a non-nil error alongside the results when any required tool is missing,
so callers can render the table before surfacing the error.
 */
func (s *DoctorService) Check() ([]domain.ToolResult, error) {
	results := s.checker.CheckAll()

	var missing []string
	for _, r := range results {
		if r.Tool.Required && !r.Found {
			missing = append(missing, r.Tool.Name)
		}
	}

	if len(missing) > 0 {
		return results, fmt.Errorf(
			"required tools are missing: %s\nInstall them and re-run: ow doctor",
			strings.Join(missing, ", "),
		)
	}

	return results, nil
}
