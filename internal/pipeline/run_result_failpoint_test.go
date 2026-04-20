package pipeline

import (
	"context"
	"errors"
	"testing"

	"salesradar/internal/domain"
)

func TestRunWithQualityGate_CoreFailpoint(t *testing.T) {
	t.Setenv("SALESRADAR_ENABLE_FAILPOINTS", "1")
	t.Setenv("SALESRADAR_FAILPOINT_CORE_PIPELINE", "error")

	_, _, err := RunWithQualityGate(context.Background(), domain.RunParams{})
	if !errors.Is(err, ErrCorePipelineFailpoint) {
		t.Fatalf("expected ErrCorePipelineFailpoint, got %v", err)
	}
}
