package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
	"time"

	domaintask "github.com/AsaqeLee/taskflow/internal/domain/task"
	"github.com/AsaqeLee/taskflow/internal/model"
)

type actionMatrixFile struct {
	Cases []actionMatrixCase `json:"cases"`
}

type actionMatrixCase struct {
	Name    string   `json:"name"`
	Status  string   `json:"status"`
	Actor   string   `json:"actor"`
	Actions []string `json:"actions"`
}

func TestAvailableActionsForUser_MatchesSharedContract(t *testing.T) {
	cases := loadActionMatrixContract(t)
	if len(cases) == 0 {
		t.Fatal("shared action matrix contract is empty")
	}

	now := time.Date(2026, 8, 13, 8, 0, 0, 0, time.UTC)
	const creatorID = "u_creator"
	const assigneeID = "u_assignee"
	const otherID = "u_other"

	for _, tc := range cases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			status, err := domaintask.ParseStatus(tc.Status)
			if err != nil {
				t.Fatalf("parse status %q: %v", tc.Status, err)
			}

			taskAssignee := assigneeID
			if status == domaintask.StatusOpen && tc.Actor != "assignee" && tc.Actor != "creator_and_assignee" {
				taskAssignee = ""
			}

			actorID := creatorID
			switch tc.Actor {
			case "creator":
				actorID = creatorID
			case "assignee":
				actorID = assigneeID
			case "other":
				actorID = otherID
			case "creator_and_assignee":
				actorID = creatorID
				taskAssignee = creatorID
			default:
				t.Fatalf("unknown actor %q", tc.Actor)
			}

			task := domaintask.Restore(
				"task_contract",
				"Contract task",
				"testdata",
				status,
				creatorID,
				taskAssignee,
				now,
				now,
				nil,
				"",
			)
			got := availableActionsForUser(task, model.User{ID: actorID})
			want := tc.Actions
			if want == nil {
				want = []string{}
			}
			if len(got) == 0 {
				got = []string{}
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("available_actions=%v, want %v", got, want)
			}
		})
	}
}

func loadActionMatrixContract(t *testing.T) []actionMatrixCase {
	t.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	path := filepath.Join(filepath.Dir(thisFile), "..", "..", "testdata", "task_action_matrix.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read contract %s: %v", path, err)
	}

	var file actionMatrixFile
	if err := json.Unmarshal(raw, &file); err != nil {
		t.Fatalf("parse contract: %v", err)
	}
	return file.Cases
}
