package task_test

import (
	"testing"

	"github.com/AsaqeLee/taskflow/internal/domain"
	domaintask "github.com/AsaqeLee/taskflow/internal/domain/task"
)

func TestParseStatus_AcceptsKnownStatuses(t *testing.T) {
	cases := []domaintask.Status{
		domaintask.StatusOpen,
		domaintask.StatusAssigned,
		domaintask.StatusInProgress,
		domaintask.StatusSubmitted,
		domaintask.StatusApproved,
		domaintask.StatusCompleted,
		domaintask.StatusCancelled,
		domaintask.StatusDeleted,
	}

	for _, want := range cases {
		got, err := domaintask.ParseStatus(want.String())
		if err != nil {
			t.Fatalf("ParseStatus(%q): %v", want, err)
		}
		if got != want {
			t.Fatalf("ParseStatus(%q) = %q, want %q", want, got, want)
		}
	}
}

func TestParseStatus_RejectsUnknownStatus(t *testing.T) {
	_, err := domaintask.ParseStatus("not_a_status")
	if err != domain.ErrInvalidTaskStatus {
		t.Fatalf("expected ErrInvalidTaskStatus, got %v", err)
	}
}
