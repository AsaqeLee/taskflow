package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/AsaqeLee/taskflow/internal/database"
	"github.com/AsaqeLee/taskflow/internal/middleware"
	"github.com/AsaqeLee/taskflow/internal/model"
	"github.com/AsaqeLee/taskflow/internal/repository"
	"github.com/AsaqeLee/taskflow/internal/service"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const testMongoURIEnv = "TASKFLOW_MONGO_TEST_URI"
const testMongoDBEnv = "TASKFLOW_MONGO_TEST_DATABASE"

func TestHandler_MongoE2EWorkflow(t *testing.T) {
	gin.SetMode(gin.TestMode)

	uri := os.Getenv(testMongoURIEnv)
	if uri == "" {
		t.Skipf("%s not set; skipping Mongo E2E workflow integration test", testMongoURIEnv)
	}

	dbName := os.Getenv(testMongoDBEnv)
	if dbName == "" {
		dbName = "taskflow_e2e_test"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		t.Fatalf("connect mongo: %v", err)
	}

	defer func() {
		_ = client.Database(dbName).Collection("tasks").Drop(context.Background())
		_ = client.Database(dbName).Collection("task_records").Drop(context.Background())
		_ = client.Database(dbName).Collection("audit_logs").Drop(context.Background())
		_ = client.Database(dbName).Collection("users").Drop(context.Background())
		_ = client.Disconnect(context.Background())
	}()

	_ = client.Database(dbName).Collection("tasks").Drop(ctx)
	_ = client.Database(dbName).Collection("task_records").Drop(ctx)
	_ = client.Database(dbName).Collection("audit_logs").Drop(ctx)
	_ = client.Database(dbName).Collection("users").Drop(ctx)

	mongoDB := client.Database(dbName)
	taskRepo := repository.NewMongoTaskRepository(mongoDB.Collection("tasks"))
	recordRepo := repository.NewMongoTaskRecordRepository(mongoDB.Collection("task_records"))
	auditRepo := repository.NewMongoAuditLogRepository(mongoDB.Collection("audit_logs"))
	userRepo := repository.NewMongoUserRepository(mongoDB.Collection("users"))
	identityRepo := repository.NewMongoIdentityRepository(
		mongoDB.Collection("refresh_tokens"),
		mongoDB.Collection("password_reset_tokens"),
	)

	defaultUsers := []model.User{
		{
			ID:    "u_test_001",
			Name:  "Test Creator",
			Role:  "owner",
			Token: "token_creator",
		},
		{
			ID:    "u_test_002",
			Name:  "Test Assignee",
			Role:  "human",
			Token: "token_assignee",
		},
		{
			ID:    "u_agent_001",
			Name:  "Hermes Agent",
			Role:  "agent",
			Token: "token_agent",
		},
	}
	for _, u := range defaultUsers {
		if _, err := userRepo.Create(context.Background(), u); err != nil {
			t.Fatalf("failed to seed user %s: %v", u.ID, err)
		}
	}

	dbClient := &database.Client{Mongo: client, DBName: dbName}
	taskSvc := service.NewTaskService(taskRepo, recordRepo, auditRepo, dbClient)
	taskHandler := NewTaskHandler(taskSvc)
	identityHandler := NewIdentityHandler(userRepo, identityRepo, "test_secret", time.Hour, 24*time.Hour, time.Hour, true)

	r := gin.New()
	r.POST("/users", identityHandler.Register)

	authenticated := r.Group("/")
	authenticated.Use(middleware.UserAuth(userRepo, "test_secret", true))
	authenticated.GET("/me", identityHandler.Me)
	authenticated.POST("/tasks", taskHandler.Create)
	authenticated.GET("/tasks", taskHandler.List)
	authenticated.GET("/tasks/:id", taskHandler.GetByID)
	authenticated.GET("/tasks/:id/records", taskHandler.ListRecords)
	authenticated.PATCH("/tasks/:id", taskHandler.UpdateBasic)
	authenticated.POST("/tasks/:id/assign", taskHandler.Assign)
	authenticated.POST("/tasks/:id/start", taskHandler.Start)
	authenticated.POST("/tasks/:id/submit", taskHandler.Submit)
	authenticated.POST("/tasks/:id/reject", taskHandler.Reject)
	authenticated.POST("/tasks/:id/approve", taskHandler.Approve)
	authenticated.POST("/tasks/:id/close", taskHandler.Close)
	authenticated.POST("/tasks/:id/cancel", taskHandler.Cancel)
	authenticated.POST("/tasks/:id/reactivate", taskHandler.Reactivate)
	authenticated.DELETE("/tasks/:id", taskHandler.Delete)
	authenticated.GET("/tasks/:id/audit_logs", taskHandler.ListAuditLogs)

	sendRequest := func(method, path, body string, headers map[string]string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}

	hCreator := map[string]string{"X-User-ID": "u_test_001"}
	hAssignee := map[string]string{"X-User-ID": "u_test_002"}

	t.Log("Step 1: Create Task")
	w1 := sendRequest("POST", "/tasks", `{"title": "Mongo E2E Task", "description": "integration check"}`, hCreator)
	if w1.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", w1.Code, w1.Body.String())
	}
	var cResp struct {
		Task model.Task `json:"task"`
	}
	if err := json.Unmarshal(w1.Body.Bytes(), &cResp); err != nil {
		t.Fatalf("decode create resp: %v", err)
	}
	taskID := cResp.Task.ID

	t.Log("Step 2: Assign Task")
	w2 := sendRequest("POST", "/tasks/"+taskID+"/assign", `{"assignee_id": "u_test_002"}`, hCreator)
	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w2.Code, w2.Body.String())
	}

	t.Log("Step 2.5: Authorization check")
	w25 := sendRequest("POST", "/tasks/"+taskID+"/start", "", hCreator)
	if w25.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden, got %d", w25.Code)
	}

	t.Log("Step 3: Start Task")
	w3 := sendRequest("POST", "/tasks/"+taskID+"/start", "", hAssignee)
	if w3.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w3.Code)
	}

	t.Log("Step 4: Submit Task")
	w4 := sendRequest("POST", "/tasks/"+taskID+"/submit", `{"content": "initial draft"}`, hAssignee)
	if w4.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w4.Code)
	}

	t.Log("Step 5: Reject Task")
	w5 := sendRequest("POST", "/tasks/"+taskID+"/reject", `{"content": "need more tests"}`, hCreator)
	if w5.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w5.Code)
	}

	t.Log("Step 6: Re-start Task")
	w6 := sendRequest("POST", "/tasks/"+taskID+"/start", "", hAssignee)
	if w6.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w6.Code)
	}

	t.Log("Step 7: Re-submit Task")
	w7 := sendRequest("POST", "/tasks/"+taskID+"/submit", `{"content": "added tests"}`, hAssignee)
	if w7.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w7.Code)
	}

	t.Log("Step 8: Approve Task")
	w8 := sendRequest("POST", "/tasks/"+taskID+"/approve", `{"content": "perfect"}`, hCreator)
	if w8.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w8.Code)
	}

	t.Log("Step 9: Close Task")
	w9 := sendRequest("POST", "/tasks/"+taskID+"/close", "", hCreator)
	if w9.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w9.Code)
	}

	t.Log("Step 10: Verify records")
	w10 := sendRequest("GET", "/tasks/"+taskID+"/records", "", hCreator)
	if w10.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w10.Code)
	}
	var recResp struct {
		Records []model.TaskRecord `json:"records"`
	}
	if err := json.Unmarshal(w10.Body.Bytes(), &recResp); err != nil {
		t.Fatalf("decode records: %v", err)
	}
	if len(recResp.Records) != 4 {
		t.Fatalf("expected 4 records, got %d", len(recResp.Records))
	}
	expectedTypes := []string{"submit", "reject", "submit", "approve"}
	for i, tType := range expectedTypes {
		if recResp.Records[i].Type != tType {
			t.Fatalf("record type mismatch at %d: expected %q, got %q", i, tType, recResp.Records[i].Type)
		}
	}

	t.Log("Step 11: Reactivate Task")
	w11 := sendRequest("POST", "/tasks/"+taskID+"/reactivate", `{"content": "reactivating task"}`, hCreator)
	if w11.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w11.Code)
	}

	t.Log("Step 12: Cancel Task")
	w12 := sendRequest("POST", "/tasks/"+taskID+"/cancel", `{"content": "abort task"}`, hCreator)
	if w12.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w12.Code)
	}

	t.Log("Step 12.5: Verify audit logs")
	w125 := sendRequest("GET", "/tasks/"+taskID+"/audit_logs", "", hCreator)
	if w125.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w125.Code)
	}
	var auditResp struct {
		AuditLogs []model.AuditLog `json:"audit_logs"`
	}
	if err := json.Unmarshal(w125.Body.Bytes(), &auditResp); err != nil {
		t.Fatalf("decode audit logs: %v", err)
	}
	if len(auditResp.AuditLogs) != 11 {
		t.Fatalf("expected 11 audit logs, got %d", len(auditResp.AuditLogs))
	}
	expectedActions := []string{
		"task_created", "task_assigned", "task_started", "task_submitted",
		"task_rejected", "task_started", "task_submitted", "task_approved",
		"task_closed", "task_reopened", "task_cancelled",
	}
	for i, act := range expectedActions {
		if auditResp.AuditLogs[i].Action != act {
			t.Fatalf("audit log action mismatch at %d: expected %q, got %q", i, act, auditResp.AuditLogs[i].Action)
		}
	}

	t.Log("Step 13: Soft Delete Task")
	w13 := sendRequest("DELETE", "/tasks/"+taskID, "", hCreator)
	if w13.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w13.Code)
	}

	t.Log("Step 13.5: Verify Mongo task is retained with deleted_at")
	var deletedTaskDoc bson.M
	if err := mongoDB.Collection("tasks").FindOne(ctx, bson.M{"_id": taskID}).Decode(&deletedTaskDoc); err != nil {
		t.Fatalf("find deleted task: %v", err)
	}
	if _, ok := deletedTaskDoc["deleted_at"]; !ok {
		t.Fatalf("expected deleted_at field to be set")
	}

	t.Log("Step 14: Verify Task is gone")
	w14 := sendRequest("GET", "/tasks/"+taskID, "", hCreator)
	if w14.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w14.Code)
	}

	t.Log("Step 15: Verify Records are retained")
	w15 := sendRequest("GET", "/tasks/"+taskID+"/records", "", hCreator)
	if w15.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w15.Code, w15.Body.String())
	}
	var recRespAfterDelete struct {
		Records []model.TaskRecord `json:"records"`
	}
	if err := json.Unmarshal(w15.Body.Bytes(), &recRespAfterDelete); err != nil {
		t.Fatalf("decode records after delete: %v", err)
	}
	if len(recRespAfterDelete.Records) != 6 {
		t.Fatalf("expected 6 retained records, got %d", len(recRespAfterDelete.Records))
	}

	t.Log("Step 15.5: Verify AuditLogs are retained")
	w155 := sendRequest("GET", "/tasks/"+taskID+"/audit_logs", "", hCreator)
	if w155.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w155.Code, w155.Body.String())
	}
	var auditRespAfterDelete struct {
		AuditLogs []model.AuditLog `json:"audit_logs"`
	}
	if err := json.Unmarshal(w155.Body.Bytes(), &auditRespAfterDelete); err != nil {
		t.Fatalf("decode audit logs after delete: %v", err)
	}
	if len(auditRespAfterDelete.AuditLogs) != 12 {
		t.Fatalf("expected 12 retained audit logs, got %d", len(auditRespAfterDelete.AuditLogs))
	}
	if auditRespAfterDelete.AuditLogs[11].Action != model.AuditActionDeleted {
		t.Fatalf("expected final audit action %q, got %q", model.AuditActionDeleted, auditRespAfterDelete.AuditLogs[11].Action)
	}

	t.Log("All 15 steps of MongoDB E2E verified!")
}
