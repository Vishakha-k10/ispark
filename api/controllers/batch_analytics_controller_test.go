package controllers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/iips-oss/ispark/api/config"
	"github.com/iips-oss/ispark/api/models"
	"github.com/iips-oss/ispark/api/routes"
	"github.com/iips-oss/ispark/api/utils"
)

func TestBatchAnalyticsOverview(t *testing.T) {
	t.Setenv("JWT_SECRET", strings.Repeat("test-jwt-", 4))
	t.Setenv("JWT_REFRESH_SECRET", strings.Repeat("test-refresh-jwt-", 4))

	SetupTestDB(t)

	app := fiber.New()
	routes.SetupRoutes(app)

	// Seed Admin
	hashedPassword, _ := utils.HashPassword("Password123!")
	testAdmin := models.Admin{
		AdminID:       "batchadmin",
		Name:          "Batch Admin",
		Email:         "batch.admin@isparc.dev",
		Password:      hashedPassword,
		Role:          "admin",
		AssignedBatch: "IT2K24",
	}
	config.DB.Create(&testAdmin)

	// Seed Students
	students := []models.Student{
		{RollNo: "IT2K24001", Name: "Alice", CourseName: "IT", Semester: 4, EmailID: "alice@test.dev", EnrollmentNo: "E001"},
		{RollNo: "IT2K24002", Name: "Bob", CourseName: "IT", Semester: 4, EmailID: "bob@test.dev", EnrollmentNo: "E002"},
	}
	for _, s := range students {
		config.DB.Create(&s)
	}

	// Seed Certificates
	certs := []models.Certificate{
		{StudentRollNo: "IT2K24001", ActivityName: "Workshop A", ActivityCategory: "Personality Development", Credits: 10, Status: "Approved"},
		{StudentRollNo: "IT2K24001", ActivityName: "Workshop B", ActivityCategory: "Skill Building", Credits: 15, Status: "Approved"},
		{StudentRollNo: "IT2K24002", ActivityName: "Workshop C", ActivityCategory: "Personality Development", Credits: 5, Status: "Pending"},
	}
	for _, c := range certs {
		config.DB.Create(&c)
	}

	token, err := utils.GenerateAccessToken(testAdmin.AdminID, testAdmin.Email, testAdmin.Role)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	t.Run("GetBatchAnalyticsOverview_Success", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/admin/batch-analytics", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200, got %d", resp.StatusCode)
		}

		var body map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			t.Fatalf("Failed to decode JSON: %v", err)
		}

		summary, ok := body["summary"].(map[string]interface{})
		if !ok {
			t.Fatal("Missing summary in response")
		}

		if int(summary["total_students"].(float64)) != 2 {
			t.Errorf("Expected total_students 2, got %v", summary["total_students"])
		}

		batches, ok := body["batches"].([]interface{})
		if !ok || len(batches) == 0 {
			t.Fatal("Expected non-empty batches list")
		}
	})

	t.Run("GetBatchDetail_Success", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/admin/batch-analytics/IT2K24", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200, got %d", resp.StatusCode)
		}

		var body map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			t.Fatalf("Failed to decode JSON: %v", err)
		}

		batchObj, ok := body["batch"].(map[string]interface{})
		if !ok {
			t.Fatal("Missing batch details in response")
		}

		if batchObj["name"] != "IT2K24" {
			t.Errorf("Expected batch name IT2K24, got %v", batchObj["name"])
		}
	})

	t.Run("UpdateBatchAnalytics_Success", func(t *testing.T) {
		bodyJSON := `{"status":"Excellent","notes":"Batch performing very well"}`
		req := httptest.NewRequest("PUT", "/api/admin/batch-analytics/IT2K24", strings.NewReader(bodyJSON))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200, got %d", resp.StatusCode)
		}
	})

	t.Run("ExportBatchReport_Success", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/admin/batch-analytics/reports/export?type=batch", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200, got %d", resp.StatusCode)
		}

		if !strings.Contains(resp.Header.Get("Content-Type"), "text/csv") {
			t.Errorf("Expected Content-Type text/csv, got %s", resp.Header.Get("Content-Type"))
		}
	})
}
