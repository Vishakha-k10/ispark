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

func TestGetAdminProfile(t *testing.T) {
	t.Setenv("JWT_SECRET", strings.Repeat("test-jwt-", 4))
	t.Setenv("JWT_REFRESH_SECRET", strings.Repeat("test-refresh-jwt-", 4))

	SetupTestDB(t)

	app := fiber.New()
	routes.SetupRoutes(app)

	// Seed an admin
	hashedPassword, _ := utils.HashPassword("Password123")
	testAdmin := models.Admin{
		AdminID:       "testadmin",
		Name:          "Test Admin",
		Email:         "test.admin@isparc.dev",
		Password:      hashedPassword,
		Role:          "admin",
		AssignedBatch: "IT2K20",
	}
	config.DB.Create(&testAdmin)

	// 1. Unauthenticated Request
	t.Run("GetAdminProfile_Unauthenticated", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/admin/profile", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("Expected 401, got %d", resp.StatusCode)
		}
	})

	// 2. Authenticated Request
	t.Run("GetAdminProfile_Success", func(t *testing.T) {
		token, err := utils.GenerateAccessToken(testAdmin.AdminID, testAdmin.Email, testAdmin.Role)
		if err != nil {
			t.Fatalf("Failed to generate token: %v", err)
		}

		req := httptest.NewRequest("GET", "/api/admin/profile", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected 200, got %d", resp.StatusCode)
		}

		var respBody map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		adminMap, ok := respBody["admin"].(map[string]interface{})
		if !ok {
			t.Fatalf("Admin data missing in response")
		}

		if adminMap["admin_id"] != testAdmin.AdminID {
			t.Errorf("Expected admin_id %s, got %v", testAdmin.AdminID, adminMap["admin_id"])
		}
		if adminMap["name"] != testAdmin.Name {
			t.Errorf("Expected name %s, got %v", testAdmin.Name, adminMap["name"])
		}
		if adminMap["email"] != testAdmin.Email {
			t.Errorf("Expected email %s, got %v", testAdmin.Email, adminMap["email"])
		}
		if adminMap["role"] != testAdmin.Role {
			t.Errorf("Expected role %s, got %v", testAdmin.Role, adminMap["role"])
		}
		if adminMap["assigned_batch"] != testAdmin.AssignedBatch {
			t.Errorf("Expected assigned_batch %s, got %v", testAdmin.AssignedBatch, adminMap["assigned_batch"])
		}
	})
}

func TestUpdateAdminProfile(t *testing.T) {
	t.Setenv("JWT_SECRET", strings.Repeat("test-jwt-", 4))
	t.Setenv("JWT_REFRESH_SECRET", strings.Repeat("test-refresh-jwt-", 4))

	SetupTestDB(t)

	app := fiber.New()
	routes.SetupRoutes(app)

	// Seed admins for testing conflicts
	hashedPassword, _ := utils.HashPassword("Password123")
	testAdmin := models.Admin{
		AdminID:       "admin1",
		Name:          "Admin One",
		Email:         "admin1@isparc.dev",
		Password:      hashedPassword,
		Role:          "admin",
		AssignedBatch: "IT2K20",
	}
	config.DB.Create(&testAdmin)

	otherAdmin := models.Admin{
		AdminID:       "admin2",
		Name:          "Admin Two",
		Email:         "admin2@isparc.dev",
		Password:      hashedPassword,
		Role:          "admin",
		AssignedBatch: "IT2K20",
	}
	config.DB.Create(&otherAdmin)

	token, err := utils.GenerateAccessToken(testAdmin.AdminID, testAdmin.Email, testAdmin.Role)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// 1. Success case: Update own name and email
	t.Run("UpdateAdminProfile_Success", func(t *testing.T) {
		body := `{"name":"Admin One Updated","email":"new.admin1@isparc.dev"}`
		req := httptest.NewRequest("PUT", "/api/admin/profile", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected 200, got %d", resp.StatusCode)
		}

		// Verify database state
		var dbAdmin models.Admin
		config.DB.Where("admin_id = ?", testAdmin.AdminID).First(&dbAdmin)
		if dbAdmin.Name != "Admin One Updated" {
			t.Errorf("Expected name 'Admin One Updated', got %s", dbAdmin.Name)
		}
		if dbAdmin.Email != "new.admin1@isparc.dev" {
			t.Errorf("Expected email 'new.admin1@isparc.dev', got %s", dbAdmin.Email)
		}
	})

	// 2. Blank name after trim -> 400
	t.Run("UpdateAdminProfile_BlankName", func(t *testing.T) {
		body := `{"name":"   ","email":"valid@isparc.dev"}`
		req := httptest.NewRequest("PUT", "/api/admin/profile", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", resp.StatusCode)
		}
	})

	// 3. Invalid email format -> 400
	invalidEmails := []string{
		"not-an-email",
		"@@@",
		"admin@",
		"a@b",
		"  x  ",
	}
	for _, email := range invalidEmails {
		t.Run("UpdateAdminProfile_InvalidEmail_"+email, func(t *testing.T) {
			body := `{"name":"Valid Name","email":"` + email + `"}`
			req := httptest.NewRequest("PUT", "/api/admin/profile", strings.NewReader(body))
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("Failed to execute request: %v", err)
			}
			if resp.StatusCode != http.StatusBadRequest {
				t.Errorf("Expected 400 for email '%s', got %d", email, resp.StatusCode)
			}
		})
	}

	// 4. Duplicate email (exact case) -> 409
	t.Run("UpdateAdminProfile_DuplicateEmail_Exact", func(t *testing.T) {
		body := `{"name":"Valid Name","email":"admin2@isparc.dev"}`
		req := httptest.NewRequest("PUT", "/api/admin/profile", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}
		if resp.StatusCode != http.StatusConflict {
			t.Errorf("Expected 409, got %d", resp.StatusCode)
		}
	})

	// 5. Duplicate email (different case) -> 409
	t.Run("UpdateAdminProfile_DuplicateEmail_DifferentCase", func(t *testing.T) {
		body := `{"name":"Valid Name","email":"ADMIN2@isparc.dev"}`
		req := httptest.NewRequest("PUT", "/api/admin/profile", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}
		if resp.StatusCode != http.StatusConflict {
			t.Errorf("Expected 409, got %d", resp.StatusCode)
		}
	})

	// 6. Name overlong (> 100) -> 400
	t.Run("UpdateAdminProfile_NameOverlong", func(t *testing.T) {
		overlongName := strings.Repeat("A", 101)
		body := `{"name":"` + overlongName + `","email":"valid.name@isparc.dev"}`
		req := httptest.NewRequest("PUT", "/api/admin/profile", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", resp.StatusCode)
		}
	})

	// 7. Name boundary condition (exactly 100) -> 200
	t.Run("UpdateAdminProfile_NameBoundary", func(t *testing.T) {
		boundaryName := strings.Repeat("A", 100)
		body := `{"name":"` + boundaryName + `","email":"valid.name@isparc.dev"}`
		req := httptest.NewRequest("PUT", "/api/admin/profile", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected 200, got %d", resp.StatusCode)
		}
	})

	// 8. Email overlong (> 100) -> 400
	t.Run("UpdateAdminProfile_EmailOverlong", func(t *testing.T) {
		overlongEmail := strings.Repeat("a", 90) + "@isparc.dev"
		body := `{"name":"Valid Name","email":"` + overlongEmail + `"}`
		req := httptest.NewRequest("PUT", "/api/admin/profile", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", resp.StatusCode)
		}
	})

	// 9. Email boundary condition (exactly 100) -> 200
	t.Run("UpdateAdminProfile_EmailBoundary", func(t *testing.T) {
		boundaryEmail := strings.Repeat("a", 89) + "@isparc.dev"
		body := `{"name":"Valid Name","email":"` + boundaryEmail + `"}`
		req := httptest.NewRequest("PUT", "/api/admin/profile", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected 200, got %d", resp.StatusCode)
		}
	})
}

func TestAdminChangePassword(t *testing.T) {
	t.Setenv("JWT_SECRET", strings.Repeat("test-jwt-", 4))
	t.Setenv("JWT_REFRESH_SECRET", strings.Repeat("test-refresh-jwt-", 4))

	SetupTestDB(t)

	app := fiber.New()
	routes.SetupRoutes(app)

	hashedPassword, _ := utils.HashPassword("Password123!")
	testAdmin := models.Admin{
		AdminID:       "passwordadmin",
		Name:          "Password Admin",
		Email:         "pw.admin@isparc.dev",
		Password:      hashedPassword,
		Role:          "admin",
		AssignedBatch: "IT2K20",
	}
	config.DB.Create(&testAdmin)

	token, err := utils.GenerateAccessToken(testAdmin.AdminID, testAdmin.Email, testAdmin.Role)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// 1. Valid password succeeds -> 200
	t.Run("ChangePassword_Success", func(t *testing.T) {
		body := `{"current_password":"Password123!","new_password":"NewPassword123!","confirm_password":"NewPassword123!"}`
		req := httptest.NewRequest("POST", "/api/admin/change-password", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected 200, got %d", resp.StatusCode)
		}
	})

	// 2. Mismatched passwords -> 400
	t.Run("ChangePassword_Mismatched", func(t *testing.T) {
		body := `{"current_password":"NewPassword123!","new_password":"ValidPassword123!","confirm_password":"DifferentPassword123!"}`
		req := httptest.NewRequest("POST", "/api/admin/change-password", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", resp.StatusCode)
		}
	})

	// 3. Incorrect current password -> 401
	t.Run("ChangePassword_WrongCurrentPassword", func(t *testing.T) {
		body := `{"current_password":"WrongCurrentPassword123","new_password":"ValidPassword123!","confirm_password":"ValidPassword123!"}`
		req := httptest.NewRequest("POST", "/api/admin/change-password", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("Expected 401, got %d", resp.StatusCode)
		}
	})

	// 4. Weak password: too short (< 8 chars) -> 400
	t.Run("ChangePassword_TooShort", func(t *testing.T) {
		body := `{"current_password":"NewPassword123!","new_password":"Ab1!","confirm_password":"Ab1!"}`
		req := httptest.NewRequest("POST", "/api/admin/change-password", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", resp.StatusCode)
		}
	})

	// 5. Weak password: no number -> 400
	t.Run("ChangePassword_NoNumber", func(t *testing.T) {
		body := `{"current_password":"NewPassword123!","new_password":"NoNumberPass!","confirm_password":"NoNumberPass!"}`
		req := httptest.NewRequest("POST", "/api/admin/change-password", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", resp.StatusCode)
		}
	})

	// 6. Weak password: no special char -> 400
	t.Run("ChangePassword_NoSpecialChar", func(t *testing.T) {
		body := `{"current_password":"NewPassword123!","new_password":"NoSpecialChar123","confirm_password":"NoSpecialChar123"}`
		req := httptest.NewRequest("POST", "/api/admin/change-password", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", resp.StatusCode)
		}
	})

	// 7. Weak password: no uppercase -> 400
	t.Run("ChangePassword_NoUppercase", func(t *testing.T) {
		body := `{"current_password":"NewPassword123!","new_password":"nouppercase123!","confirm_password":"nouppercase123!"}`
		req := httptest.NewRequest("POST", "/api/admin/change-password", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", resp.StatusCode)
		}
	})

	// 8. Weak password: no lowercase -> 400
	t.Run("ChangePassword_NoLowercase", func(t *testing.T) {
		body := `{"current_password":"NewPassword123!","new_password":"NOLOWERCASE123!","confirm_password":"NOLOWERCASE123!"}`
		req := httptest.NewRequest("POST", "/api/admin/change-password", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", resp.StatusCode)
		}
	})
}

func TestSupervisedActivitiesStableOnNameChange(t *testing.T) {
	t.Setenv("JWT_SECRET", strings.Repeat("test-jwt-", 4))
	t.Setenv("JWT_REFRESH_SECRET", strings.Repeat("test-refresh-jwt-", 4))

	SetupTestDB(t)

	app := fiber.New()
	routes.SetupRoutes(app)

	// 1. Seed an admin and their supervised activities
	hashedPassword, _ := utils.HashPassword("Password123!")
	testAdmin := models.Admin{
		AdminID:       "stableadmin",
		Name:          "Original Name",
		Email:         "stable.admin@isparc.dev",
		Password:      hashedPassword,
		Role:          "admin",
		AssignedBatch: "IT2K24",
	}
	config.DB.Create(&testAdmin)

	testActivity := models.Activity{
		Name:          "Stable Identifier Test Activity",
		Category:      "TECHNICAL",
		Credits:       10,
		Mode:          "Offline",
		Coordinator:   "Original Name",
		CoordinatorID: "stableadmin",
	}
	config.DB.Create(&testActivity)

	token, err := utils.GenerateAccessToken(testAdmin.AdminID, testAdmin.Email, testAdmin.Role)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// 2. Fetch profile initially and assert count is 1
	t.Run("InitialCountIsOne", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/admin/profile", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200, got %d", resp.StatusCode)
		}

		var respBody map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}
		stats := respBody["stats"].(map[string]interface{})
		supervisedCount := int64(stats["supervised_activities"].(float64))
		if supervisedCount != 1 {
			t.Errorf("Expected supervised_activities to be 1, got %d", supervisedCount)
		}
	})

	// 3. Update admin name
	t.Run("UpdateNameAndCheckCountIsStillOne", func(t *testing.T) {
		body := `{"name":"New Updated Name","email":"stable.admin@isparc.dev"}`
		req := httptest.NewRequest("PUT", "/api/admin/profile", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200, got %d", resp.StatusCode)
		}

		// Fetch profile again and assert stats supervised count is still 1
		reqGet := httptest.NewRequest("GET", "/api/admin/profile", nil)
		reqGet.Header.Set("Authorization", "Bearer "+token)

		respGet, err := app.Test(reqGet)
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}
		if respGet.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200, got %d", respGet.StatusCode)
		}

		var respBody map[string]interface{}
		if err := json.NewDecoder(respGet.Body).Decode(&respBody); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}
		stats := respBody["stats"].(map[string]interface{})
		supervisedCount := int64(stats["supervised_activities"].(float64))
		if supervisedCount != 1 {
			t.Errorf("Expected supervised_activities to still be 1, got %d", supervisedCount)
		}
	})

	// 4. Update admin name and verify the returned activity coordinator name is updated
	t.Run("UpdateNameAndCheckVisibleActivityCoordinatorName", func(t *testing.T) {
		// Generate student token
		studentToken, err := utils.GenerateAccessToken("student123", "student@example.com", "student")
		if err != nil {
			t.Fatalf("Failed to generate student token: %v", err)
		}

		reqGetActivities := httptest.NewRequest("GET", "/api/student/activities", nil)
		reqGetActivities.Header.Set("Authorization", "Bearer "+studentToken)

		respGetActivities, err := app.Test(reqGetActivities)
		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}
		if respGetActivities.StatusCode != http.StatusOK {
			t.Fatalf("Expected 200, got %d", respGetActivities.StatusCode)
		}

		var activitiesList []map[string]interface{}
		if err := json.NewDecoder(respGetActivities.Body).Decode(&activitiesList); err != nil {
			t.Fatalf("Failed to decode activities response: %v", err)
		}

		// Find the activity we created earlier and check its coordinator name
		found := false
		for _, act := range activitiesList {
			if act["name"] == "Stable Identifier Test Activity" {
				found = true
				if act["coordinator"] != "New Updated Name" {
					t.Errorf("Expected coordinator name to be 'New Updated Name', got %v", act["coordinator"])
				}
			}
		}
		if !found {
			t.Errorf("Expected activity 'Stable Identifier Test Activity' to be found in list")
		}
	})
}

func TestAssignedBatchGrouping(t *testing.T) {
	t.Setenv("JWT_SECRET", strings.Repeat("test-jwt-", 4))
	t.Setenv("JWT_REFRESH_SECRET", strings.Repeat("test-refresh-jwt-", 4))

	SetupTestDB(t)

	app := fiber.New()
	routes.SetupRoutes(app)

	// 1. Seed admin assigned to batch IT2K24
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

	// 2. Seed multiple students under batch IT2K24
	students := []models.Student{
		{RollNo: "IT2K24011", Name: "Student A", CourseName: "CS", Semester: 6, ContactNo: "123", EmailID: "a@e.com", EnrollmentNo: "E1"},
		{RollNo: "IT2K24012", Name: "Student B", CourseName: "CS", Semester: 6, ContactNo: "456", EmailID: "b@e.com", EnrollmentNo: "E2"},
		{RollNo: "IT2K24013", Name: "Student C", CourseName: "CS", Semester: 6, ContactNo: "789", EmailID: "c@e.com", EnrollmentNo: "E3"},
	}
	for _, s := range students {
		config.DB.Create(&s)
	}

	token, err := utils.GenerateAccessToken(testAdmin.AdminID, testAdmin.Email, testAdmin.Role)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// 3. Fetch students and verify they all return the same canonical batch "IT2K24"
	req := httptest.NewRequest("GET", "/api/admin/students", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}

	var respBody map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	studentsList := respBody["students"].([]interface{})
	if len(studentsList) != 3 {
		t.Fatalf("Expected 3 students, got %d", len(studentsList))
	}

	// Group using the backend-provided batch
	groupedBatches := make(map[string]int)
	for _, s := range studentsList {
		studentMap := s.(map[string]interface{})
		batch, ok := studentMap["batch"].(string)
		if !ok || batch == "" {
			t.Errorf("Expected batch field to be populated on student")
		}
		groupedBatches[batch]++
	}

	// Verify we got exactly one grouped batch "IT2K24" with count 3
	if len(groupedBatches) != 1 {
		t.Errorf("Expected exactly 1 grouped batch, got %d", len(groupedBatches))
	}
	if count, ok := groupedBatches["IT2K24"]; !ok || count != 3 {
		t.Errorf("Expected batch IT2K24 to have 3 students, got %d", count)
	}
}

// ---------------------------------------------------------------------------
// AdminLogin tests
// ---------------------------------------------------------------------------

// TestAdminLogin covers all branches of the POST /api/admin/auth/login handler.
func TestAdminLogin(t *testing.T) {
	t.Setenv("JWT_SECRET", strings.Repeat("test-jwt-", 4))
	t.Setenv("JWT_REFRESH_SECRET", strings.Repeat("test-refresh-jwt-", 4))

	SetupTestDB(t)

	app := fiber.New()
	routes.SetupRoutes(app)

	const plainPassword = "Admin@1234"
	hashedPassword, _ := utils.HashPassword(plainPassword)

	// Seed an active admin
	activeAdmin := models.Admin{
		AdminID:            "loginadmin",
		Name:               "Login Admin",
		Email:              "login.admin@isparc.dev",
		Password:           hashedPassword,
		Role:               "admin",
		AssignedBatch:      "IT2K24",
		Status:             "Active",
		MustChangePassword: false,
	}
	config.DB.Create(&activeAdmin)

	// Seed an admin that must change their password on first login
	firstLoginAdmin := models.Admin{
		AdminID:            "firstlogin",
		Name:               "First Login Admin",
		Email:              "first.login@isparc.dev",
		Password:           hashedPassword,
		Role:               "admin",
		Status:             "Active",
		MustChangePassword: true,
	}
	config.DB.Create(&firstLoginAdmin)

	// Seed an inactive admin
	inactiveAdmin := models.Admin{
		AdminID:  "inactiveadmin",
		Name:     "Inactive Admin",
		Email:    "inactive@isparc.dev",
		Password: hashedPassword,
		Role:     "admin",
		Status:   "Inactive",
	}
	config.DB.Create(&inactiveAdmin)

	// Seed a superadmin
	superAdmin := models.Admin{
		AdminID:  "superlogin",
		Name:     "Super Admin",
		Email:    "super@isparc.dev",
		Password: hashedPassword,
		Role:     "superadmin",
		Status:   "Active",
	}
	config.DB.Create(&superAdmin)

	loginURL := "/api/admin/auth/login"

	makeLoginReq := func(body string) *http.Request {
		req := httptest.NewRequest(http.MethodPost, loginURL, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		return req
	}

	// 1. Success — correct credentials, active admin
	t.Run("Success_ActiveAdmin", func(t *testing.T) {
		body := `{"admin_id":"loginadmin","password":"Admin@1234"}`
		resp, err := app.Test(makeLoginReq(body))
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}

		var parsed map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
			t.Fatalf("decode body: %v", err)
		}

		if parsed["access_token"] == nil || parsed["access_token"] == "" {
			t.Error("expected non-empty access_token")
		}
		if parsed["message"] != "Admin logged in successfully" {
			t.Errorf("unexpected message: %v", parsed["message"])
		}
		if _, ok := parsed["must_change_password"]; !ok {
			t.Error("must_change_password field missing")
		}
		adminObj, ok := parsed["admin"].(map[string]interface{})
		if !ok {
			t.Fatal("admin object missing from response")
		}
		if adminObj["admin_id"] != "loginadmin" {
			t.Errorf("admin_id mismatch: got %v", adminObj["admin_id"])
		}
		if adminObj["role"] != "admin" {
			t.Errorf("role mismatch: got %v", adminObj["role"])
		}
	})

	// 2. Missing admin_id → 400
	t.Run("MissingAdminID", func(t *testing.T) {
		body := `{"password":"Admin@1234"}`
		resp, err := app.Test(makeLoginReq(body))
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", resp.StatusCode)
		}
	})

	// 3. Missing password → 400
	t.Run("MissingPassword", func(t *testing.T) {
		body := `{"admin_id":"loginadmin"}`
		resp, err := app.Test(makeLoginReq(body))
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", resp.StatusCode)
		}
	})

	// 4. Both fields missing → 400
	t.Run("MissingBothFields", func(t *testing.T) {
		body := `{}`
		resp, err := app.Test(makeLoginReq(body))
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", resp.StatusCode)
		}
	})

	// 5. Non-existent admin_id → 401
	t.Run("UnknownAdminID", func(t *testing.T) {
		body := `{"admin_id":"doesnotexist","password":"Admin@1234"}`
		resp, err := app.Test(makeLoginReq(body))
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", resp.StatusCode)
		}
		var parsed map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&parsed) //nolint:errcheck
		if parsed["error"] != "Invalid credentials" {
			t.Errorf("unexpected error: %v", parsed["error"])
		}
	})

	// 6. Correct admin_id but wrong password → 401
	t.Run("WrongPassword", func(t *testing.T) {
		body := `{"admin_id":"loginadmin","password":"WrongPass@99"}`
		resp, err := app.Test(makeLoginReq(body))
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", resp.StatusCode)
		}
		var parsed map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&parsed) //nolint:errcheck
		if parsed["error"] != "Invalid credentials" {
			t.Errorf("unexpected error: %v", parsed["error"])
		}
	})

	// 7. Correct credentials but account is inactive → 403
	t.Run("InactiveAccount", func(t *testing.T) {
		body := `{"admin_id":"inactiveadmin","password":"Admin@1234"}`
		resp, err := app.Test(makeLoginReq(body))
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("expected 403, got %d", resp.StatusCode)
		}
		var parsed map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&parsed) //nolint:errcheck
		if parsed["error"] == nil {
			t.Error("expected error message for inactive account")
		}
	})

	// 8. must_change_password flag is true in response when set on admin
	t.Run("MustChangePassword_True", func(t *testing.T) {
		body := `{"admin_id":"firstlogin","password":"Admin@1234"}`
		resp, err := app.Test(makeLoginReq(body))
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		var parsed map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if parsed["must_change_password"] != true {
			t.Errorf("expected must_change_password=true, got %v", parsed["must_change_password"])
		}
	})

	// 9. must_change_password flag is false when not set
	t.Run("MustChangePassword_False", func(t *testing.T) {
		body := `{"admin_id":"loginadmin","password":"Admin@1234"}`
		resp, err := app.Test(makeLoginReq(body))
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		var parsed map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if parsed["must_change_password"] != false {
			t.Errorf("expected must_change_password=false, got %v", parsed["must_change_password"])
		}
	})

	// 10. Superadmin role is reflected correctly in the response
	t.Run("SuperadminRole", func(t *testing.T) {
		body := `{"admin_id":"superlogin","password":"Admin@1234"}`
		resp, err := app.Test(makeLoginReq(body))
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
		var parsed map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		adminObj, ok := parsed["admin"].(map[string]interface{})
		if !ok {
			t.Fatal("admin object missing")
		}
		if adminObj["role"] != "superadmin" {
			t.Errorf("expected role=superadmin, got %v", adminObj["role"])
		}
	})

	// 11. Response must not expose the password field
	t.Run("PasswordNotExposed", func(t *testing.T) {
		body := `{"admin_id":"loginadmin","password":"Admin@1234"}`
		resp, err := app.Test(makeLoginReq(body))
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		var parsed map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		adminObj, ok := parsed["admin"].(map[string]interface{})
		if !ok {
			t.Fatal("admin object missing from response")
		}
		if _, hasPassword := adminObj["password"]; hasPassword {
			t.Error("password field must not be present in login response")
		}
	})
}

func TestAdminDownloadCertificate_MissingFile(t *testing.T) {
	t.Setenv("JWT_SECRET", strings.Repeat("test-jwt-", 4))
	t.Setenv("JWT_REFRESH_SECRET", strings.Repeat("test-refresh-jwt-", 4))

	SetupTestDB(t)

	app := fiber.New()
	routes.SetupRoutes(app)

	// Seed Admin
	hashedPassword, _ := utils.HashPassword("Password123!")
	admin := models.Admin{
		AdminID:       "certadmin",
		Name:          "Cert Admin",
		Email:         "certadmin@isparc.dev",
		Password:      hashedPassword,
		Role:          "admin",
		AssignedBatch: "IT2K24",
	}
	config.DB.Create(&admin)

	// Seed Student in batch
	student := models.Student{
		RollNo:       "IT2K24999",
		Name:         "Cert Student",
		EmailID:      "certstudent@isparc.dev",
		CourseName:   "IT",
		Semester:     4,
		EnrollmentNo: "EN999",
	}
	config.DB.Create(&student)

	// Seed Certificate with missing physical file path
	cert := models.Certificate{
		ID:               9999,
		StudentRollNo:    "IT2K24999",
		ActivityName:     "Missing File Cert",
		ActivityCategory: "Skill Building",
		Credits:          10,
		Status:           "Approved",
		FileName:         "missing_file.pdf",
		FilePath:         "./uploads/certificates/non_existent_file_12345.pdf",
	}
	config.DB.Create(&cert)

	token, err := utils.GenerateAccessToken(admin.AdminID, admin.Email, admin.Role)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	t.Run("MissingFile_Returns404", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/admin/certificates/9999/download", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := app.Test(req, 10000)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Expected status 404 Not Found for missing certificate file, got %d", resp.StatusCode)
		}
	})
}
