package controllers

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/iips-oss/ispark/api/config"
	"github.com/iips-oss/ispark/api/models"
)

// Helper: classify an activity category as PD (Personality Development) vs SB (Skill Building)
func isPDCategory(category string) bool {
	cat := strings.ToLower(strings.TrimSpace(category))
	return strings.Contains(cat, "personality") ||
		strings.Contains(cat, "pd") ||
		strings.Contains(cat, "leadership") ||
		strings.Contains(cat, "speaking") ||
		strings.Contains(cat, "cultural") ||
		strings.Contains(cat, "social") ||
		strings.Contains(cat, "soft")
}

// Helper: format batch key (e.g. IT2K24 -> IT 2024 or IT2K24)
func canonicalBatchName(rollNo string) string {
	roll := strings.TrimSpace(rollNo)
	if len(roll) >= 6 {
		return roll[:6]
	}
	if len(roll) > 0 {
		return roll
	}
	return "Unassigned"
}

type BatchStat struct {
	Name         string `json:"name"`
	Students     int    `json:"students"`
	PD           int    `json:"pd"`
	SB           int    `json:"sb"`
	Compliance   int    `json:"compliance"`
	Defaulters   int    `json:"defaulters"`
	PendingCerts int    `json:"pending_certs"`
	Status       string `json:"status"`
	Notes        string `json:"notes,omitempty"`
}

type StudentBatchDetail struct {
	RollNo           string `json:"roll_no"`
	Name             string `json:"name"`
	CourseName       string `json:"course_name"`
	Semester         int    `json:"semester"`
	PDCredits        int    `json:"pd_credits"`
	SBCredits        int    `json:"sb_credits"`
	TotalCredits     int    `json:"total_credits"`
	PendingCerts     int    `json:"pending_certs"`
	ComplianceStatus string `json:"compliance_status"`
	IsDefaulter      bool   `json:"is_defaulter"`
	IsInactive       bool   `json:"is_inactive"`
}

// GET /api/admin/batch-analytics
func GetBatchAnalyticsOverview(c *fiber.Ctx) error {
	admin, err := getAuthenticatedAdmin(c)
	if err != nil {
		return err
	}

	var students []models.Student
	query := config.DB.Model(&models.Student{}).Preload("Certificates").Preload("Enrollments")
	query, scoped := scopeToAssignedBatch(query, admin)

	if !scoped {
		return c.JSON(fiber.Map{
			"summary": fiber.Map{
				"assigned_batches":   0,
				"total_students":     0,
				"compliant_students": 0,
				"defaulters":         0,
			},
			"batches":      []BatchStat{},
			"alerts":       []fiber.Map{},
			"requirements": []fiber.Map{},
			"reports":      []fiber.Map{},
		})
	}

	if err := query.Find(&students).Error; err != nil {
		return errJSON(c, fiber.StatusInternalServerError, "Failed to retrieve student batch analytics")
	}

	// Fetch batch overrides
	var overrides []models.BatchOverride
	overrideMap := make(map[string]models.BatchOverride)
	if err := config.DB.Find(&overrides).Error; err == nil {
		for _, o := range overrides {
			overrideMap[strings.ToUpper(o.BatchName)] = o
		}
	}

	// Data structures for aggregation
	batchMap := make(map[string]*BatchStat)
	batchStudentsMap := make(map[string][]StudentBatchDetail)

	totalStudents := len(students)
	compliantCount := 0
	defaultersCount := 0
	inactiveCount := 0
	totalPendingReviews := 0

	completedBothCount := 0
	onlyPDCount := 0
	onlySBCount := 0
	neitherCount := 0

	for _, s := range students {
		batchKey := canonicalBatchName(s.RollNo)

		pdCredits := 0
		sbCredits := 0
		pendingCerts := 0

		for _, cert := range s.Certificates {
			switch cert.Status {
			case "Approved":
				if isPDCategory(cert.ActivityCategory) {
					pdCredits += cert.Credits
				} else {
					sbCredits += cert.Credits
				}
			case "Pending":
				pendingCerts++
				totalPendingReviews++
			}
		}

		totalCreds := pdCredits + sbCredits
		hasPD := pdCredits > 0
		hasSB := sbCredits > 0

		complianceStatus := "Neither Track Completed"
		if hasPD && hasSB {
			complianceStatus = "Completed Both Tracks"
			completedBothCount++
			compliantCount++
		} else if hasPD {
			complianceStatus = "Only Personality Development Completed"
			onlyPDCount++
		} else if hasSB {
			complianceStatus = "Only Skill Building Completed"
			onlySBCount++
		} else {
			neitherCount++
		}

		isDefaulter := !hasPD || !hasSB || totalCreds == 0
		if isDefaulter {
			defaultersCount++
		}

		isInactive := len(s.Enrollments) == 0 && len(s.Certificates) == 0
		if isInactive {
			inactiveCount++
		}

		studentDetail := StudentBatchDetail{
			RollNo:           s.RollNo,
			Name:             s.Name,
			CourseName:       s.CourseName,
			Semester:         s.Semester,
			PDCredits:        pdCredits,
			SBCredits:        sbCredits,
			TotalCredits:     totalCreds,
			PendingCerts:     pendingCerts,
			ComplianceStatus: complianceStatus,
			IsDefaulter:      isDefaulter,
			IsInactive:       isInactive,
		}

		batchStudentsMap[batchKey] = append(batchStudentsMap[batchKey], studentDetail)

		if b, exists := batchMap[batchKey]; exists {
			b.Students++
			b.PD += pdCredits
			b.SB += sbCredits
			b.PendingCerts += pendingCerts
			if isDefaulter {
				b.Defaulters++
			}
		} else {
			defCount := 0
			if isDefaulter {
				defCount = 1
			}
			batchMap[batchKey] = &BatchStat{
				Name:         batchKey,
				Students:     1,
				PD:           pdCredits,
				SB:           sbCredits,
				Defaulters:   defCount,
				PendingCerts: pendingCerts,
			}
		}
	}

	// Calculate compliance & status for each batch
	var batchList []BatchStat
	for key, b := range batchMap {
		compliantStudentsInBatch := 0
		for _, sd := range batchStudentsMap[key] {
			if sd.ComplianceStatus == "Completed Both Tracks" {
				compliantStudentsInBatch++
			}
		}

		if b.Students > 0 {
			b.Compliance = (compliantStudentsInBatch * 100) / b.Students
		} else {
			b.Compliance = 0
		}

		// Calculate status
		if b.Compliance >= 90 {
			b.Status = "Excellent"
		} else if b.Compliance >= 75 {
			b.Status = "Good"
		} else {
			b.Status = "At Risk"
		}

		// Apply override if present
		if ov, ok := overrideMap[strings.ToUpper(key)]; ok {
			if ov.Status != "" {
				b.Status = ov.Status
			}
			b.Notes = ov.Notes
		}

		batchList = append(batchList, *b)
	}

	// Ensure assigned batch is included if empty
	if len(batchList) == 0 && admin.Role == "admin" && admin.AssignedBatch != "" {
		batchList = append(batchList, BatchStat{
			Name:       admin.AssignedBatch,
			Students:   0,
			Compliance: 0,
			Status:     "At Risk",
		})
	}

	assignedBatchesCount := len(batchList)

	alerts := []fiber.Map{
		{
			"tone":        "rose",
			"title":       "Defaulters",
			"description": "Students who have not met compliance requirements",
			"value":       fmt.Sprintf("%d Students", defaultersCount),
			"count":       defaultersCount,
			"icon":        "M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126ZM12 15.75h.007v.008H12v-.008Z",
		},
		{
			"tone":        "amber",
			"title":       "Inactive Students",
			"description": "No activity recorded in the current semester",
			"value":       fmt.Sprintf("%d Students", inactiveCount),
			"count":       inactiveCount,
			"icon":        "M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126ZM12 15.75h.007v.008H12v-.008Z",
		},
		{
			"tone":        "blue",
			"title":       "Pending Certificate Reviews",
			"description": "Awaiting mentor approval and verification",
			"value":       fmt.Sprintf("%d Submissions", totalPendingReviews),
			"count":       totalPendingReviews,
			"icon":        "M12 6v6h4.5m4.5 0a9 9 0 11-18 0 9 9 0 0118 0z",
		},
	}

	requirements := []fiber.Map{
		{"tone": "emerald", "label": "Completed Both Tracks", "count": completedBothCount},
		{"tone": "amber", "label": "Only Personality Development Completed", "count": onlyPDCount},
		{"tone": "amber", "label": "Only Skill Building Completed", "count": onlySBCount},
		{"tone": "rose", "label": "Neither Track Completed", "count": neitherCount},
	}

	reports := []fiber.Map{
		{"title": "Batch Report", "type": "batch"},
		{"title": "Student Progress Report", "type": "student_progress"},
		{"title": "Compliance Report", "type": "compliance"},
		{"title": "Certificate Verification Report", "type": "certificate_verification"},
	}

	return c.JSON(fiber.Map{
		"summary": fiber.Map{
			"assigned_batches":   assignedBatchesCount,
			"total_students":     totalStudents,
			"compliant_students": compliantCount,
			"defaulters":         defaultersCount,
		},
		"batches":      batchList,
		"alerts":       alerts,
		"requirements": requirements,
		"reports":      reports,
	})
}

// GET /api/admin/batch-analytics/:batch
func GetBatchDetail(c *fiber.Ctx) error {
	admin, err := getAuthenticatedAdmin(c)
	if err != nil {
		return err
	}

	batchParam := strings.TrimSpace(c.Params("batch"))
	if batchParam == "" {
		return errJSON(c, fiber.StatusBadRequest, "Batch name parameter is required")
	}

	var students []models.Student
	query := config.DB.Model(&models.Student{}).Preload("Certificates").Preload("Enrollments")
	query, scoped := scopeToAssignedBatch(query, admin)

	if !scoped {
		return errJSON(c, fiber.StatusNotFound, "Batch not found or unauthorized")
	}

	// Filter by batch name (e.g. IT2K24 or IT 2024)
	cleanBatch := strings.ReplaceAll(batchParam, " ", "")
	query = query.Where("roll_no LIKE ?", cleanBatch+"%")

	if err := query.Find(&students).Error; err != nil {
		return errJSON(c, fiber.StatusInternalServerError, "Failed to retrieve batch detail")
	}

	var override models.BatchOverride
	notes := ""
	overrideStatus := ""
	if err := config.DB.Where("LOWER(batch_name) = LOWER(?)", cleanBatch).First(&override).Error; err == nil {
		notes = override.Notes
		overrideStatus = override.Status
	}

	totalStudents := len(students)
	totalPD := 0
	totalSB := 0
	totalPending := 0
	defaulters := 0
	compliant := 0

	var studentDetails []StudentBatchDetail

	for _, s := range students {
		pdCredits := 0
		sbCredits := 0
		pendingCerts := 0

		for _, cert := range s.Certificates {
			switch cert.Status {
			case "Approved":
				if isPDCategory(cert.ActivityCategory) {
					pdCredits += cert.Credits
				} else {
					sbCredits += cert.Credits
				}
			case "Pending":
				pendingCerts++
				totalPending++
			}
		}

		totalCreds := pdCredits + sbCredits
		hasPD := pdCredits > 0
		hasSB := sbCredits > 0

		complianceStatus := "Neither Track Completed"
		if hasPD && hasSB {
			complianceStatus = "Completed Both Tracks"
			compliant++
		} else if hasPD {
			complianceStatus = "Only Personality Development Completed"
		} else if hasSB {
			complianceStatus = "Only Skill Building Completed"
		}

		isDefaulter := !hasPD || !hasSB || totalCreds == 0
		if isDefaulter {
			defaulters++
		}

		isInactive := len(s.Enrollments) == 0 && len(s.Certificates) == 0

		totalPD += pdCredits
		totalSB += sbCredits

		studentDetails = append(studentDetails, StudentBatchDetail{
			RollNo:           s.RollNo,
			Name:             s.Name,
			CourseName:       s.CourseName,
			Semester:         s.Semester,
			PDCredits:        pdCredits,
			SBCredits:        sbCredits,
			TotalCredits:     totalCreds,
			PendingCerts:     pendingCerts,
			ComplianceStatus: complianceStatus,
			IsDefaulter:      isDefaulter,
			IsInactive:       isInactive,
		})
	}

	compliancePct := 0
	if totalStudents > 0 {
		compliancePct = (compliant * 100) / totalStudents
	}

	calcStatus := "Good"
	if compliancePct >= 90 {
		calcStatus = "Excellent"
	} else if compliancePct < 75 {
		calcStatus = "At Risk"
	}

	if overrideStatus != "" {
		calcStatus = overrideStatus
	}

	return c.JSON(fiber.Map{
		"batch": BatchStat{
			Name:         batchParam,
			Students:     totalStudents,
			PD:           totalPD,
			SB:           totalSB,
			Compliance:   compliancePct,
			Defaulters:   defaulters,
			PendingCerts: totalPending,
			Status:       calcStatus,
			Notes:        notes,
		},
		"students": studentDetails,
	})
}

// PUT /api/admin/batch-analytics/:batch
func UpdateBatchAnalytics(c *fiber.Ctx) error {
	admin, err := getAuthenticatedAdmin(c)
	if err != nil {
		return err
	}

	batchParam := strings.TrimSpace(c.Params("batch"))
	if batchParam == "" {
		return errJSON(c, fiber.StatusBadRequest, "Batch name parameter is required")
	}

	cleanBatch := strings.ReplaceAll(batchParam, " ", "")

	// Verify authorization if admin is batch-scoped
	if admin.Role == "admin" {
		if admin.AssignedBatch == "" || !strings.HasPrefix(strings.ToUpper(cleanBatch), strings.ToUpper(admin.AssignedBatch)) {
			return errJSON(c, fiber.StatusForbidden, "Unauthorized to edit this batch")
		}
	}

	var input struct {
		Status string `json:"status"`
		Notes  string `json:"notes"`
	}

	if err := c.BodyParser(&input); err != nil {
		return errJSON(c, fiber.StatusBadRequest, "Invalid request payload")
	}

	if input.Status != "" {
		validStatuses := map[string]bool{"Excellent": true, "Good": true, "At Risk": true}
		if !validStatuses[input.Status] {
			return errJSON(c, fiber.StatusBadRequest, "Invalid status. Must be Excellent, Good, or At Risk")
		}
	}

	var override models.BatchOverride
	err = config.DB.Where("LOWER(batch_name) = LOWER(?)", cleanBatch).First(&override).Error
	if err != nil {
		override = models.BatchOverride{
			BatchName: cleanBatch,
			Status:    input.Status,
			Notes:     input.Notes,
		}
		if err := config.DB.Create(&override).Error; err != nil {
			return errJSON(c, fiber.StatusInternalServerError, "Failed to create batch override")
		}
	} else {
		if input.Status != "" {
			override.Status = input.Status
		}
		override.Notes = input.Notes
		if err := config.DB.Save(&override).Error; err != nil {
			return errJSON(c, fiber.StatusInternalServerError, "Failed to update batch override")
		}
	}

	return c.JSON(fiber.Map{
		"message": "Batch parameters updated successfully",
		"batch": fiber.Map{
			"name":   batchParam,
			"status": override.Status,
			"notes":  override.Notes,
		},
	})
}

// Helper function to filter CSV report rows produced by reports_controller by admin's assigned batch
func filterReportRowsByBatch(rawRows [][]string, admin *models.Admin) [][]string {
	if len(rawRows) <= 1 {
		return rawRows
	}
	if admin.Role == "admin" {
		if admin.AssignedBatch == "" {
			return rawRows[:1] // Fail closed for admin with no assigned batch
		}

		header := rawRows[0]
		rollIndex := -1
		for i, col := range header {
			colLower := strings.ToLower(col)
			if colLower == "roll no" || colLower == "student" || colLower == "student roll no" || colLower == "rollno" {
				rollIndex = i
				break
			}
		}

		if rollIndex == -1 {
			return rawRows
		}

		filtered := [][]string{header}
		prefix := strings.ToUpper(admin.AssignedBatch)
		for _, row := range rawRows[1:] {
			if len(row) > rollIndex && strings.HasPrefix(strings.ToUpper(strings.TrimSpace(row[rollIndex])), prefix) {
				filtered = append(filtered, row)
			}
		}
		return filtered
	}
	return rawRows
}

func filterMentorRowsByBatch(rawRows [][]string, admin *models.Admin) [][]string {
	if len(rawRows) <= 1 {
		return rawRows
	}
	if admin.Role == "admin" {
		if admin.AssignedBatch == "" {
			return rawRows[:1] // Fail closed
		}

		header := rawRows[0]
		batchIndex := -1
		for i, col := range header {
			if strings.ToLower(col) == "assigned batch" {
				batchIndex = i
				break
			}
		}

		if batchIndex == -1 {
			return rawRows
		}

		filtered := [][]string{header}
		prefix := strings.ToUpper(admin.AssignedBatch)
		for _, row := range rawRows[1:] {
			if len(row) > batchIndex {
				val := strings.ToUpper(strings.TrimSpace(row[batchIndex]))
				if strings.HasPrefix(val, prefix) || val == "ALL BATCHES" {
					filtered = append(filtered, row)
				}
			}
		}
		return filtered
	}
	return rawRows
}

func filterActivityRowsByBatch(rawRows [][]string, admin *models.Admin) [][]string {
	if len(rawRows) <= 1 {
		return rawRows
	}
	if admin.Role == "admin" {
		if admin.AssignedBatch == "" {
			return rawRows[:1] // Fail closed
		}

		header := rawRows[0]
		rollIndex := -1
		for i, col := range header {
			colLower := strings.ToLower(col)
			if colLower == "roll no" || colLower == "student" || colLower == "student roll no" || colLower == "rollno" || colLower == "batch" {
				rollIndex = i
				break
			}
		}

		if rollIndex == -1 {
			return rawRows
		}

		filtered := [][]string{header}
		prefix := strings.ToUpper(admin.AssignedBatch)
		for _, row := range rawRows[1:] {
			if len(row) > rollIndex && (strings.HasPrefix(strings.ToUpper(strings.TrimSpace(row[rollIndex])), prefix) || strings.EqualFold(strings.TrimSpace(row[rollIndex]), prefix)) {
				filtered = append(filtered, row)
			}
		}
		return filtered
	}
	return rawRows
}

// GET /api/admin/batch-analytics/export
func ExportBatchReport(c *fiber.Ctx) error {
	admin, err := getAuthenticatedAdmin(c)
	if err != nil {
		return err
	}

	reportType := strings.ToLower(strings.TrimSpace(c.Query("type")))
	if reportType == "" {
		reportType = "batch"
	}

	format := strings.ToLower(strings.TrimSpace(c.Query("format")))

	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	var fileName string
	var rows [][]string

	switch reportType {
	case "certificate_verification", "certificate verification report", "certificate_verification_report":
		fileName = "Certificate_Verification_Report.csv"
		rawRows, err := certificateReportData(models.GenerateReportInput{})
		if err != nil {
			return errJSON(c, fiber.StatusInternalServerError, "Failed to generate report")
		}
		rows = filterReportRowsByBatch(rawRows, admin)

	case "student_progress", "student progress report", "student_progress_report":
		fileName = "Student_Progress_Report.csv"
		rawRows, err := buildReportData(models.GenerateReportInput{Type: "Student Performance"})
		if err != nil {
			return errJSON(c, fiber.StatusInternalServerError, "Failed to generate report")
		}
		rows = filterReportRowsByBatch(rawRows, admin)

	case "activity_participation", "activity participation report", "activity_participation_report":
		fileName = "Activity_Participation_Report.csv"
		rawRows, err := activityReportData(models.GenerateReportInput{})
		if err != nil {
			return errJSON(c, fiber.StatusInternalServerError, "Failed to generate report")
		}
		rows = filterActivityRowsByBatch(rawRows, admin)

	case "mentor_analytics", "mentor analytics report", "mentor_analytics_report":
		fileName = "Mentor_Analytics_Report.csv"
		rawRows, err := mentorReportData()
		if err != nil {
			return errJSON(c, fiber.StatusInternalServerError, "Failed to generate report")
		}
		rows = filterMentorRowsByBatch(rawRows, admin)

	case "batch", "batch report", "batch_report":
		fileName = "Batch_Performance_Report.csv"
		var students []models.Student
		query := config.DB.Model(&models.Student{}).Preload("Certificates").Preload("Enrollments")
		query, scoped := scopeToAssignedBatch(query, admin)
		if !scoped {
			return errJSON(c, fiber.StatusForbidden, "Unauthorized to access reports")
		}
		if err := query.Find(&students).Error; err != nil {
			return errJSON(c, fiber.StatusInternalServerError, "Failed to gather report data")
		}

		rows = [][]string{{"Batch Name", "Students", "PD Credits", "SB Credits", "Compliance %", "Defaulters", "Pending Certs", "Status"}}
		batchMap := make(map[string]*BatchStat)
		for _, s := range students {
			batchKey := canonicalBatchName(s.RollNo)
			pdCredits, sbCredits, pendingCerts := 0, 0, 0
			for _, cert := range s.Certificates {
				switch cert.Status {
				case "Approved":
					if isPDCategory(cert.ActivityCategory) {
						pdCredits += cert.Credits
					} else {
						sbCredits += cert.Credits
					}
				case "Pending":
					pendingCerts++
				}
			}

			isDefaulter := pdCredits == 0 || sbCredits == 0
			if b, ok := batchMap[batchKey]; ok {
				b.Students++
				b.PD += pdCredits
				b.SB += sbCredits
				b.PendingCerts += pendingCerts
				if isDefaulter {
					b.Defaulters++
				}
			} else {
				defCount := 0
				if isDefaulter {
					defCount = 1
				}
				batchMap[batchKey] = &BatchStat{
					Name:         batchKey,
					Students:     1,
					PD:           pdCredits,
					SB:           sbCredits,
					Defaulters:   defCount,
					PendingCerts: pendingCerts,
				}
			}
		}

		for _, b := range batchMap {
			compliance := 0
			if b.Students > 0 {
				compliance = ((b.Students - b.Defaulters) * 100) / b.Students
			}
			status := "Good"
			if compliance >= 90 {
				status = "Excellent"
			} else if compliance < 75 {
				status = "At Risk"
			}

			rows = append(rows, []string{
				b.Name,
				strconv.Itoa(b.Students),
				strconv.Itoa(b.PD),
				strconv.Itoa(b.SB),
				strconv.Itoa(compliance) + "%",
				strconv.Itoa(b.Defaulters),
				strconv.Itoa(b.PendingCerts),
				status,
			})
		}

	case "compliance", "compliance report", "compliance_report":
		fileName = "Compliance_Report.csv"
		var students []models.Student
		query := config.DB.Model(&models.Student{}).Preload("Certificates")
		query, scoped := scopeToAssignedBatch(query, admin)
		if !scoped {
			return errJSON(c, fiber.StatusForbidden, "Unauthorized to access reports")
		}
		if err := query.Find(&students).Error; err != nil {
			return errJSON(c, fiber.StatusInternalServerError, "Failed to gather report data")
		}

		rows = [][]string{{"Roll No", "Name", "Batch", "Completed Both Tracks", "Only PD Completed", "Only SB Completed", "Neither Track Completed", "Compliance Status"}}
		for _, s := range students {
			batchKey := canonicalBatchName(s.RollNo)
			pdCredits, sbCredits := 0, 0
			for _, cert := range s.Certificates {
				if cert.Status == "Approved" {
					if isPDCategory(cert.ActivityCategory) {
						pdCredits += cert.Credits
					} else {
						sbCredits += cert.Credits
					}
				}
			}
			hasPD := pdCredits > 0
			hasSB := sbCredits > 0

			both, onlyPD, onlySB, neither := "No", "No", "No", "No"
			status := "Defaulter"
			if hasPD && hasSB {
				both = "Yes"
				status = "Compliant"
			} else if hasPD {
				onlyPD = "Yes"
			} else if hasSB {
				onlySB = "Yes"
			} else {
				neither = "Yes"
			}

			rows = append(rows, []string{
				s.RollNo,
				s.Name,
				batchKey,
				both,
				onlyPD,
				onlySB,
				neither,
				status,
			})
		}

	default:
		rawRows, err := buildReportData(models.GenerateReportInput{Type: reportType})
		if err != nil {
			return errJSON(c, fiber.StatusBadRequest, "Invalid report type requested")
		}
		fileName = fmt.Sprintf("%s_Report.csv", strings.ReplaceAll(reportType, " ", "_"))
		rows = filterReportRowsByBatch(rawRows, admin)
	}

	if err := writer.WriteAll(rows); err != nil {
		return errJSON(c, fiber.StatusInternalServerError, "Failed to write CSV report")
	}
	writer.Flush()

	if format == "json" {
		return c.JSON(fiber.Map{
			"message":   "Report generated successfully",
			"file_name": fileName,
			"type":      reportType,
		})
	}

	c.Set("Content-Type", "text/csv")
	c.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fileName))
	return c.Send(buf.Bytes())
}
