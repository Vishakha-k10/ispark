package controllers_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/iips-oss/ispark/api/config"
	"github.com/iips-oss/ispark/api/controllers"
	"github.com/iips-oss/ispark/api/models"
	"github.com/iips-oss/ispark/api/utils"
	"gorm.io/gorm"
)

func setupActivityMonitoringApp(t *testing.T) (*fiber.App, string, uint) {
	t.Helper()

	SetupTestDB(t)

	// AutoMigrate all models
	err := config.DB.AutoMigrate(
		&models.Admin{},
		&models.Student{},
		&models.Activity{},
		&models.Enrollment{},
		&models.Certificate{},
		&models.AdminNote{},
	)
	if err != nil {
		t.Fatalf("Failed to migrate test DB: %v", err)
	}

	// Clear test tables safely
	config.DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&models.AdminNote{})
	config.DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&models.Certificate{})
	config.DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&models.Enrollment{})
	config.DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&models.Activity{})
	config.DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&models.Student{})
	config.DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&models.Admin{})

	// Create test admin
	hashedPwd, _ := utils.HashPassword("AdminPass123!")
	admin := models.Admin{
		AdminID:  "ADM001",
		Name:     "Test Super Admin",
		Email:    "admin@test.com",
		Password: hashedPwd,
		Role:     "superadmin",
		Status:   "Active",
	}
	if err := config.DB.Create(&admin).Error; err != nil {
		t.Fatalf("Failed to create test admin: %v", err)
	}

	token, err := utils.GenerateAccessToken(admin.AdminID, admin.Email, admin.Role)
	if err != nil {
		t.Fatalf("Failed to generate access token: %v", err)
	}

	// Seed test student
	student := models.Student{
		RollNo:       "21CS0071",
		Name:         "Dev Mehta",
		CourseName:   "MCA 5yrs Integrated",
		Semester:     6,
		ContactNo:    "9876543210",
		EmailID:      "dev@test.com",
		EnrollmentNo: "21CS0071",
		IsVerified:   true,
		Status:       "Approved",
	}
	config.DB.Create(&student)

	// Seed test activities
	act1 := models.Activity{
		Name:         "Leadership Workshop",
		Category:     "Soft Skills",
		Credits:      10,
		Mode:         "Offline",
		Status:       "Completed",
		RegDeadline:  time.Now().AddDate(0, 0, -10),
		ActivityDate: time.Now().AddDate(0, 0, -5),
		Coordinator:  "Dr. Rajesh Kumar",
	}
	config.DB.Create(&act1)

	act2 := models.Activity{
		Name:         "Research Project",
		Category:     "Academic",
		Credits:      20,
		Mode:         "Hybrid",
		Status:       "Ongoing",
		RegDeadline:  time.Now(),
		ActivityDate: time.Now().AddDate(0, 1, 0),
		Coordinator:  "Dr. Anita Sharma",
	}
	config.DB.Create(&act2)

	// Seed enrollment
	config.DB.Create(&models.Enrollment{
		StudentRollNo: student.RollNo,
		ActivityID:    act1.ID,
		Status:        "Completed",
	})

	// Seed certificate
	config.DB.Create(&models.Certificate{
		StudentRollNo:    student.RollNo,
		ActivityName:     act1.Name,
		ActivityCategory: act1.Category,
		Credits:          10,
		Status:           "Pending",
	})

	app := fiber.New()

	// Auth Middleware helper for tests
	authMW := func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Missing token"})
		}
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token header"})
		}
		claims, err := utils.ValidateAccessToken(parts[1])
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token"})
		}
		c.Locals("roll_no", claims.RollNo)
		c.Locals("user_id", claims.RollNo)
		c.Locals("admin_id", claims.RollNo)
		c.Locals("role", claims.Role)
		return c.Next()
	}

	adminGroup := app.Group("/api/admin", authMW)
	adminGroup.Get("/monitoring/stats", controllers.GetActivityMonitoringStats)
	adminGroup.Get("/monitoring/activities", controllers.GetMonitoredActivities)
	adminGroup.Put("/monitoring/activities/:id", controllers.UpdateMonitoredActivity)
	adminGroup.Get("/monitoring/insights", controllers.GetMonitoringInsights)
	adminGroup.Get("/monitoring/attention-students", controllers.GetStudentsRequiringAttention)
	adminGroup.Post("/monitoring/send-reminder", controllers.SendActivityMonitoringReminder)

	return app, token, act1.ID
}

func TestGetActivityMonitoringStats(t *testing.T) {
	app, token, _ := setupActivityMonitoringApp(t)

	req := httptest.NewRequest("GET", "/api/admin/monitoring/stats", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("Failed to decode response body: %v", err)
	}

	if body["total_monitored"] == nil {
		t.Errorf("Expected total_monitored in response")
	}
}

func TestGetMonitoredActivities(t *testing.T) {
	app, token, _ := setupActivityMonitoringApp(t)

	req := httptest.NewRequest("GET", "/api/admin/monitoring/activities?search=Leadership&sort_by=credits-desc", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var activities []controllers.MonitoredActivityResponse
	if err := json.NewDecoder(resp.Body).Decode(&activities); err != nil {
		t.Fatalf("Failed to decode response body: %v", err)
	}

	if len(activities) == 0 {
		t.Errorf("Expected at least one activity returned")
	}
	if activities[0].Name != "Leadership Workshop" {
		t.Errorf("Expected Leadership Workshop, got %s", activities[0].Name)
	}
}

func TestUpdateMonitoredActivity(t *testing.T) {
	app, token, actID := setupActivityMonitoringApp(t)

	newCredits := 15
	payload := controllers.UpdateMonitoredActivityInput{
		Name:        "Updated Leadership Workshop",
		Category:    "Soft Skills",
		Credits:     &newCredits,
		Status:      "Ongoing",
		StartDate:   "01 Jan 2025",
		EndDate:     "15 Jan 2025",
		Coordinator: "Dr. New Coordinator",
	}

	jsonBytes, _ := json.Marshal(payload)
	req := httptest.NewRequest("PUT", fmt.Sprintf("/api/admin/monitoring/activities/%d", actID), bytes.NewBuffer(jsonBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Test non-existent activity ID
	req404 := httptest.NewRequest("PUT", "/api/admin/monitoring/activities/9999", bytes.NewBuffer(jsonBytes))
	req404.Header.Set("Content-Type", "application/json")
	req404.Header.Set("Authorization", "Bearer "+token)
	resp404, _ := app.Test(req404)
	if resp404.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404 for non-existent ID, got %d", resp404.StatusCode)
	}
}

func TestGetMonitoringInsights(t *testing.T) {
	app, token, _ := setupActivityMonitoringApp(t)

	req := httptest.NewRequest("GET", "/api/admin/monitoring/insights", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestGetStudentsRequiringAttention(t *testing.T) {
	app, token, _ := setupActivityMonitoringApp(t)

	req := httptest.NewRequest("GET", "/api/admin/monitoring/attention-students", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var attentionList []controllers.AttentionStudentResponse
	if err := json.NewDecoder(resp.Body).Decode(&attentionList); err != nil {
		t.Fatalf("Failed to decode response body: %v", err)
	}

	if len(attentionList) == 0 {
		t.Errorf("Expected attention students returned")
	}
}

func TestSendActivityMonitoringReminder(t *testing.T) {
	app, token, _ := setupActivityMonitoringApp(t)

	payload := controllers.ActivityReminderInput{
		StudentEnrollment: "21CS0071",
		ActivityName:      "Technical Symposium",
		Issue:             "Certificate Not Uploaded",
		Message:           "Please upload your certificate as soon as possible.",
	}

	jsonBytes, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/admin/monitoring/send-reminder", bytes.NewBuffer(jsonBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Test missing student identifier
	badPayload := controllers.ActivityReminderInput{
		ActivityName: "Some Activity",
	}
	badBytes, _ := json.Marshal(badPayload)
	reqBad := httptest.NewRequest("POST", "/api/admin/monitoring/send-reminder", bytes.NewBuffer(badBytes))
	reqBad.Header.Set("Content-Type", "application/json")
	reqBad.Header.Set("Authorization", "Bearer "+token)

	respBad, _ := app.Test(reqBad)
	if respBad.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400 for missing student roll/enrollment, got %d", respBad.StatusCode)
	}
}
