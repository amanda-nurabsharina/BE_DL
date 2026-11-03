package production

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"dental-lab-backend/internal/modules/auth"
	"dental-lab-backend/internal/modules/customers"
)

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	svc := &Service{db: db}
	svc.ensureStagesAndSeedOrders()
	return svc
}

func parseFlexDate(dateStr string, fallback time.Time) time.Time {
	dateStr = strings.TrimSpace(dateStr)
	if dateStr == "" {
		return fallback
	}
	layouts := []string{
		"2006-01-02",
		time.RFC3339,
		"2006-01-02T15:04:05",
	}
	for _, l := range layouts {
		if t, err := time.Parse(l, dateStr); err == nil {
			return t
		}
	}
	return fallback
}

func (s *Service) ensureStagesAndSeedOrders() {
	// Ensure PFM_CROWN has explicit Design & Wax-up -> Casting -> Finishing -> QC stages matching dental lab workflow
	var custCount int64
	s.db.Model(&customers.Customer{}).Count(&custCount)
	if custCount == 0 {
		sampleCustomers := []customers.Customer{
			{Code: "CUST-0001", Name: "drg. Budi Santoso, Sp.Pros", ClinicName: "Klinik Gigi Senyum Sehat", Phone: "081234567801", Email: "drg.budi@senyumsehat.id", Address: "Jl. Sudirman No. 45, Jakarta", Preferences: `{"shade_guide":"VITA A2"}`, IsActive: true},
			{Code: "CUST-0002", Name: "drg. Ratna Permata, Sp.KG", ClinicName: "RSIA Bunda Dental Center", Phone: "081298765432", Email: "ratna.drg@bundadental.com", Address: "Jl. Menteng Raya No. 12, Jakarta", Preferences: `{"shade_guide":"VITA A1"}`, IsActive: true},
			{Code: "CUST-0003", Name: "drg. Hendra Wijaya", ClinicName: "Praktek Mandiri drg. Hendra", Phone: "081345678912", Email: "hendra.dental@gmail.com", Address: "Jl. Gatot Subroto No. 88, Bandung", Preferences: `{"shade_guide":"VITA A3"}`, IsActive: true},
		}
		for i := range sampleCustomers {
			_ = s.db.Create(&sampleCustomers[i]).Error
		}
	}

	var orderCount int64
	s.db.Model(&ProductionOrder{}).Count(&orderCount)
	if orderCount == 0 {
		var custs []customers.Customer
		s.db.Order("code ASC").Find(&custs)
		if len(custs) == 0 {
			return
		}

		var techUser auth.User
		_ = s.db.Where("username = ?", "teknisi1").First(&techUser).Error
		var techID *uuid.UUID
		if techUser.ID != uuid.Nil {
			id := techUser.ID
			techID = &id
		}

		var pfmType, zircType, acrylicType DentureType
		_ = s.db.Preload("StageTemplates", func(db *gorm.DB) *gorm.DB { return db.Order("stage_order ASC") }).Where("code = ?", "PFM_CROWN").First(&pfmType).Error
		_ = s.db.Preload("StageTemplates", func(db *gorm.DB) *gorm.DB { return db.Order("stage_order ASC") }).Where("code = ?", "ZIRCONIA_CROWN").First(&zircType).Error
		_ = s.db.Preload("StageTemplates", func(db *gorm.DB) *gorm.DB { return db.Order("stage_order ASC") }).Where("code = ?", "ACRYLIC_DENTURE").First(&acrylicType).Error

		now := time.Now()

		// Sample Order 1: PFM Crown in Casting stage (shows Model Gips ✓ -> Design/Wax-up ✓ -> Casting [IN_PROGRESS] -> Porselen -> Finishing -> QC)
		if pfmType.ID != 0 {
			ord1 := ProductionOrder{
				OrderNumber:         "ORD-0001",
				CustomerID:          custs[0].ID,
				PatientName:         "Bpk. Agus Salim",
				DentureTypeID:       pfmType.ID,
				TeethNumbers:        `["11","21"]`,
				ShadeGuide:          "VITA A2",
				ReceivedDate:        now.AddDate(0, 0, -2),
				PromisedDueDate:     now.AddDate(0, 0, 2),
				Status:              "IN_PRODUCTION",
				CurrentStageCode:    "METAL_CASTING",
				CurrentStageName:    "Casting & Finishing Logam",
				CurrentTechnicianID: techID,
				Priority:            "URGENT",
				DoctorNotes:         "Mohon perhatikan kontak proksimal gigi 11-21 dan servikal fit rapat.",
				TotalPrice:          900000,
			}
			if err := s.db.Create(&ord1).Error; err == nil {
				for _, tmpl := range pfmType.StageTemplates {
					st := "PENDING"
					var startedAt, finishedAt *time.Time
					notes := ""
					if tmpl.StageOrder < 3 {
						st = "COMPLETED"
						sTime := now.AddDate(0, 0, -2)
						fTime := now.AddDate(0, 0, -1)
						startedAt = &sTime
						finishedAt = &fTime
						notes = "Tahap selesai sesuai instruksi"
					} else if tmpl.StageOrder == 3 {
						st = "IN_PROGRESS"
						sTime := now.AddDate(0, 0, -1)
						startedAt = &sTime
						notes = "Sedang proses pengecoran koping logam Co-Cr"
					}
					run := ProductionStageRun{
						ProductionOrderID: ord1.ID,
						StageCode:         tmpl.StageCode,
						StageName:         tmpl.StageName,
						StageOrder:        tmpl.StageOrder,
						Status:            st,
						TechnicianID:      techID,
						StartedAt:         startedAt,
						FinishedAt:        finishedAt,
						Notes:             notes,
					}
					_ = s.db.Create(&run).Error
				}
			}
		}

		// Sample Order 2: Zirconia Crown waiting for QC Review (QC_PENDING)
		if zircType.ID != 0 && len(custs) > 1 {
			ord2 := ProductionOrder{
				OrderNumber:         "ORD-0002",
				CustomerID:          custs[1].ID,
				PatientName:         "Ibu Siska Amelia",
				DentureTypeID:       zircType.ID,
				TeethNumbers:        `["16"]`,
				ShadeGuide:          "VITA A1",
				ReceivedDate:        now.AddDate(0, 0, -3),
				PromisedDueDate:     now.AddDate(0, 0, 1),
				Status:              "QC_PENDING",
				CurrentStageCode:    "QC_PENDING",
				CurrentStageName:    "Menunggu Quality Control (QC)",
				CurrentTechnicianID: techID,
				Priority:            "NORMAL",
				DoctorNotes:         "Mahkota molar 16 zirconia monolitik, oklusi anatomis.",
				TotalPrice:          750000,
			}
			if err := s.db.Create(&ord2).Error; err == nil {
				for _, tmpl := range zircType.StageTemplates {
					sTime := now.AddDate(0, 0, -3)
					fTime := now.AddDate(0, 0, -1)
					run := ProductionStageRun{
						ProductionOrderID: ord2.ID,
						StageCode:         tmpl.StageCode,
						StageName:         tmpl.StageName,
						StageOrder:        tmpl.StageOrder,
						Status:            "COMPLETED",
						TechnicianID:      techID,
						StartedAt:         &sTime,
						FinishedAt:        &fTime,
						Notes:             "Selesai dikerjakan teknisi",
					}
					_ = s.db.Create(&run).Error
				}
			}
		}

		// Sample Order 3: Acrylic Denture Completed & Ready to Deliver
		if acrylicType.ID != 0 && len(custs) > 2 {
			doneTime := now.Add(-4 * time.Hour)
			ord3 := ProductionOrder{
				OrderNumber:         "ORD-0003",
				CustomerID:          custs[2].ID,
				PatientName:         "Bpk. Darmawan",
				DentureTypeID:       acrylicType.ID,
				TeethNumbers:        `["35","36","37","45","46"]`,
				ShadeGuide:          "VITA A3",
				ReceivedDate:        now.AddDate(0, 0, -5),
				PromisedDueDate:     now,
				CompletedAt:         &doneTime,
				Status:              "COMPLETED",
				CurrentStageCode:    "QC_PASSED",
				CurrentStageName:    "Lolos QC (Selesai Produksi)",
				CurrentTechnicianID: techID,
				Priority:            "NORMAL",
				DoctorNotes:         "Gigi tiruan sebagian lepasan akrilik rahang bawah.",
				TotalPrice:          650000,
			}
			if err := s.db.Create(&ord3).Error; err == nil {
				for _, tmpl := range acrylicType.StageTemplates {
					sTime := now.AddDate(0, 0, -5)
					fTime := now.AddDate(0, 0, -1)
					run := ProductionStageRun{
						ProductionOrderID: ord3.ID,
						StageCode:         tmpl.StageCode,
						StageName:         tmpl.StageName,
						StageOrder:        tmpl.StageOrder,
						Status:            "COMPLETED",
						TechnicianID:      techID,
						StartedAt:         &sTime,
						FinishedAt:        &fTime,
						Notes:             "Tahap selesai & terverifikasi",
					}
					_ = s.db.Create(&run).Error
				}
				if techID != nil {
					qc := QualityCheck{
						ProductionOrderID: ord3.ID,
						InspectorID:       *techID,
						Passed:            true,
						ChecklistResults:  `{"oklusi":true,"marginal_fit":true,"kontak_proksimal":true,"shade_match":true,"polishing":true}`,
						DefectNotes:       "Semua parameter QC lolos, plat akrilik halus dan retensi baik.",
						CheckedAt:         doneTime,
					}
					_ = s.db.Create(&qc).Error
				}
			}
		}
	}
}

func (s *Service) ListDentureTypes() ([]DentureType, error) {
	var list []DentureType
	err := s.db.Preload("StageTemplates", func(db *gorm.DB) *gorm.DB {
		return db.Order("stage_order ASC")
	}).Where("is_active = true").Order("id ASC").Find(&list).Error
	return list, err
}

func (s *Service) ListOrders(status, customerIDStr, search string, tenantID *uuid.UUID) ([]ProductionOrder, error) {
	var list []ProductionOrder
	q := s.db.Preload("Customer").
		Preload("Patient").
		Preload("DentureType").
		Preload("CurrentTechnician").
		Preload("StageRuns", func(db *gorm.DB) *gorm.DB {
			return db.Order("stage_order ASC")
		}).
		Preload("StageRuns.Technician").
		Preload("QualityChecks", func(db *gorm.DB) *gorm.DB {
			return db.Order("checked_at DESC")
		}).
		Preload("QualityChecks.Inspector").
		Preload("Attachments", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at DESC")
		}).
		Where("deleted_at IS NULL")

	if tenantID != nil {
		q = q.Where("tenant_id = ?", *tenantID)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if customerIDStr != "" {
		if custID, err := uuid.Parse(customerIDStr); err == nil {
			q = q.Where("customer_id = ?", custID)
		}
	}
	if search != "" {
		likeTerm := "%" + search + "%"
		q = q.Where("order_number ILIKE ? OR patient_name ILIKE ?", likeTerm, likeTerm)
	}

	err := q.Order("created_at DESC").Find(&list).Error
	return list, err
}

func (s *Service) GetOrderByID(id uuid.UUID) (*ProductionOrder, error) {
	var order ProductionOrder
	err := s.db.Preload("Customer").
		Preload("Patient").
		Preload("DentureType").
		Preload("CurrentTechnician").
		Preload("StageRuns", func(db *gorm.DB) *gorm.DB {
			return db.Order("stage_order ASC")
		}).
		Preload("StageRuns.Technician").
		Preload("QualityChecks", func(db *gorm.DB) *gorm.DB {
			return db.Order("checked_at DESC")
		}).
		Preload("QualityChecks.Inspector").
		Preload("Attachments", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at DESC")
		}).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&order).Error

	if err != nil {
		return nil, err
	}
	return &order, nil
}

type CreateOrderInput struct {
	TenantID        *uuid.UUID `json:"tenant_id"`
	CustomerID      uuid.UUID  `json:"customer_id"`
	PatientID       *uuid.UUID `json:"patient_id"`
	PatientName     string     `json:"patient_name"`
	DentureTypeID   int        `json:"denture_type_id"`
	TeethNumbers    string     `json:"teeth_numbers"` // e.g. ["11", "21"]
	ShadeGuide      string     `json:"shade_guide"`
	ReceivedDate    string     `json:"received_date"`
	PromisedDueDate string     `json:"promised_due_date"`
	TechnicianID    *uuid.UUID `json:"technician_id"`
	Priority        string     `json:"priority"`
	DoctorNotes     string     `json:"doctor_notes"`
	InternalNotes   string     `json:"internal_notes"`
	DiscountPercent float64    `json:"discount_percent"`
	DiscountAmount  float64    `json:"discount_amount"`
	TotalPrice      float64    `json:"total_price"`
	CreatedBy       *uuid.UUID `json:"-"`
}

func (s *Service) CreateOrder(input CreateOrderInput) (*ProductionOrder, error) {
	if input.PatientName == "" {
		return nil, errors.New("patient name is required")
	}
	if input.DentureTypeID == 0 {
		return nil, errors.New("denture type is required")
	}
	if input.TeethNumbers == "" {
		input.TeethNumbers = "[]"
	}
	if input.Priority == "" {
		input.Priority = "NORMAL"
	}

	// Fetch denture type & its stage templates
	var dt DentureType
	if err := s.db.Preload("StageTemplates", func(db *gorm.DB) *gorm.DB {
		return db.Order("stage_order ASC")
	}).First(&dt, input.DentureTypeID).Error; err != nil {
		return nil, errors.New("invalid denture type selected")
	}

	if input.TotalPrice == 0 {
		input.TotalPrice = dt.BasePrice
	}

	now := time.Now()
	recDate := parseFlexDate(input.ReceivedDate, now)
	dueDate := parseFlexDate(input.PromisedDueDate, now.AddDate(0, 0, dt.EstimatedDays))

	var order ProductionOrder
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var count int64
		tx.Model(&ProductionOrder{}).Count(&count)
		orderNumber := fmt.Sprintf("ORD-%04d", count+1)

		firstStageCode := ""
		firstStageName := ""
		if len(dt.StageTemplates) > 0 {
			firstStageCode = dt.StageTemplates[0].StageCode
			firstStageName = dt.StageTemplates[0].StageName
		}

		initialStatus := "IN_PRODUCTION"
		if input.TechnicianID == nil {
			initialStatus = "INTAKE"
		}

		// Ensure patient record exists in Master Pasien (patients table) or update existing
		patientID := input.PatientID
		if patientID == nil {
			var existingPat customers.Patient
			if err := tx.Where("LOWER(name) = LOWER(?) AND deleted_at IS NULL", input.PatientName).First(&existingPat).Error; err == nil {
				patientID = &existingPat.ID
				tx.Model(&existingPat).Updates(map[string]interface{}{
					"default_shade":      input.ShadeGuide,
					"last_teeth_numbers": input.TeethNumbers,
					"updated_at":         now,
				})
			} else {
				var patCount int64
				tx.Unscoped().Model(&customers.Patient{}).Count(&patCount)
				newPat := customers.Patient{
					TenantID:         input.TenantID,
					Code:             fmt.Sprintf("PAS-%04d", patCount+1),
					CustomerID:       &input.CustomerID,
					Name:             input.PatientName,
					Gender:           "L",
					DefaultShade:     input.ShadeGuide,
					LastTeethNumbers: input.TeethNumbers,
					IsActive:         true,
					CreatedAt:        now,
					UpdatedAt:        now,
				}
				if err := tx.Create(&newPat).Error; err == nil {
					patientID = &newPat.ID
				}
			}
		} else {
			tx.Model(&customers.Patient{}).Where("id = ?", *patientID).Updates(map[string]interface{}{
				"default_shade":      input.ShadeGuide,
				"last_teeth_numbers": input.TeethNumbers,
				"updated_at":         now,
			})
		}

		order = ProductionOrder{
			TenantID:            input.TenantID,
			OrderNumber:         orderNumber,
			CustomerID:          input.CustomerID,
			PatientID:           patientID,
			PatientName:         input.PatientName,
			DentureTypeID:       input.DentureTypeID,
			TeethNumbers:        input.TeethNumbers,
			ShadeGuide:          input.ShadeGuide,
			ReceivedDate:        recDate,
			PromisedDueDate:     dueDate,
			Status:              initialStatus,
			CurrentStageCode:    firstStageCode,
			CurrentStageName:    firstStageName,
			CurrentTechnicianID: input.TechnicianID,
			Priority:            input.Priority,
			DoctorNotes:         input.DoctorNotes,
			InternalNotes:       input.InternalNotes,
			DiscountPercent:     input.DiscountPercent,
			DiscountAmount:      input.DiscountAmount,
			TotalPrice:          input.TotalPrice,
			CreatedBy:           input.CreatedBy,
		}

		if err := tx.Create(&order).Error; err != nil {
			return err
		}

		// Instantiate stage runs from templates
		for _, tmpl := range dt.StageTemplates {
			status := "PENDING"
			var startedAt *time.Time
			var stageTechID *uuid.UUID
			if tmpl.StageOrder == 1 {
				status = "IN_PROGRESS"
				startedAt = &now
				stageTechID = input.TechnicianID
			}
			run := ProductionStageRun{
				ProductionOrderID: order.ID,
				StageCode:         tmpl.StageCode,
				StageName:         tmpl.StageName,
				StageOrder:        tmpl.StageOrder,
				Status:            status,
				TechnicianID:      stageTechID,
				StartedAt:         startedAt,
				CreatedAt:         now,
			}
			if err := tx.Create(&run).Error; err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return s.GetOrderByID(order.ID)
}

func (s *Service) AssignTechnician(orderID uuid.UUID, techID uuid.UUID) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&ProductionOrder{}).Where("id = ?", orderID).Updates(map[string]interface{}{
			"current_technician_id": techID,
			"status":                "IN_PRODUCTION",
			"updated_at":            time.Now(),
		}).Error; err != nil {
			return err
		}
		// Also assign to current active stage run
		_ = tx.Model(&ProductionStageRun{}).
			Where("production_order_id = ? AND status = 'IN_PROGRESS'", orderID).
			Update("technician_id", techID).Error
		return nil
	})
}

type UpdateStageRunInput struct {
	TechnicianID *uuid.UUID `json:"technician_id"`
	Status       string     `json:"status"`
	Notes        *string    `json:"notes"`
}

func (s *Service) UpdateStageRun(orderID uuid.UUID, runID uuid.UUID, input UpdateStageRunInput) (*ProductionOrder, error) {
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var run ProductionStageRun
		if err := tx.Where("id = ? AND production_order_id = ?", runID, orderID).First(&run).Error; err != nil {
			return errors.New("stage run not found")
		}

		updates := map[string]interface{}{}
		if input.TechnicianID != nil {
			updates["technician_id"] = input.TechnicianID
			// If this is the active stage, also update order's current_technician_id
			if run.Status == "IN_PROGRESS" {
				_ = tx.Model(&ProductionOrder{}).Where("id = ?", orderID).Update("current_technician_id", input.TechnicianID).Error
			}
		}
		if input.Notes != nil {
			updates["notes"] = *input.Notes
		}
		if input.Status != "" {
			updates["status"] = input.Status
			now := time.Now()
			if input.Status == "IN_PROGRESS" && run.StartedAt == nil {
				updates["started_at"] = &now
			} else if input.Status == "COMPLETED" {
				updates["finished_at"] = &now
			}
		}

		if len(updates) > 0 {
			if err := tx.Model(&ProductionStageRun{}).Where("id = ?", runID).Updates(updates).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.GetOrderByID(orderID)
}

func (s *Service) UpdateOrderStatus(orderID uuid.UUID, status string, internalNotes string) (*ProductionOrder, error) {
	now := time.Now()
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": now,
	}
	if internalNotes != "" {
		updates["internal_notes"] = internalNotes
	}
	switch status {
	case "INTAKE":
		updates["current_stage_name"] = "Order Masuk (Intake)"
	case "IN_PRODUCTION":
		updates["current_stage_name"] = "Dalam Proses Produksi"
	case "QC_PENDING":
		updates["current_stage_code"] = "QC_PENDING"
		updates["current_stage_name"] = "Menunggu Quality Control (QC)"
	case "COMPLETED":
		updates["current_stage_code"] = "QC_PASSED"
		updates["current_stage_name"] = "Selesai Produksi (Siap Kirim)"
		updates["completed_at"] = &now
	case "DELIVERED":
		updates["current_stage_code"] = "DELIVERED"
		updates["current_stage_name"] = "Dikirim ke Dokter / Klinik"
		updates["delivered_at"] = &now
	case "CANCELLED":
		updates["current_stage_code"] = "CANCELLED"
		updates["current_stage_name"] = "Dibatalkan"
	}

	if err := s.db.Model(&ProductionOrder{}).Where("id = ?", orderID).Updates(updates).Error; err != nil {
		return nil, err
	}
	if status == "COMPLETED" || status == "DELIVERED" {
		s.autoCreateInvoiceIfNeeded(s.db, orderID)
	}
	return s.GetOrderByID(orderID)
}

func (s *Service) autoCreateInvoiceIfNeeded(tx *gorm.DB, orderID uuid.UUID) {
	var existingCount int64
	tx.Table("invoices").Where("production_order_id = ? AND status != 'VOID' AND deleted_at IS NULL", orderID).Count(&existingCount)
	if existingCount > 0 {
		return
	}

	var order ProductionOrder
	if err := tx.Preload("DentureType").Where("id = ?", orderID).First(&order).Error; err != nil {
		return
	}

	var totalCount int64
	tx.Table("invoices").Count(&totalCount)
	invNum := fmt.Sprintf("INV-%04d", totalCount+1)
	now := time.Now()
	dueDate := now.AddDate(0, 0, 14)

	subtotal := order.TotalPrice + order.DiscountAmount
	if subtotal <= 0 {
		subtotal = order.TotalPrice
	}
	total := order.TotalPrice
	invID := uuid.New()

	_ = tx.Exec(`
		INSERT INTO invoices (id, invoice_number, production_order_id, customer_id, issue_date, due_date, subtotal, discount_amount, total_amount, paid_amount, balance_due, status, notes, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 0, ?, 'UNPAID', ?, ?, ?)
	`, invID, invNum, order.ID, order.CustomerID, now, dueDate, subtotal, order.DiscountAmount, total, total,
		fmt.Sprintf("Invoice otomatis dari penyelesaian SPK %s (Pasien: %s)", order.OrderNumber, order.PatientName), now, now).Error

	desc := fmt.Sprintf("%s - Pasien: %s (Elemen Gigi: %s, Shade: %s)", order.DentureType.Name, order.PatientName, order.TeethNumbers, order.ShadeGuide)
	_ = tx.Exec(`
		INSERT INTO invoice_items (id, invoice_id, description, quantity, unit_price, discount, subtotal)
		VALUES (?, ?, ?, 1, ?, ?, ?)
	`, uuid.New(), invID, desc, subtotal, order.DiscountAmount, total).Error
}

func (s *Service) AdvanceStage(orderID uuid.UUID, currentRunID uuid.UUID, notes string, techID *uuid.UUID, nextTechID *uuid.UUID) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var currentRun ProductionStageRun
		if err := tx.Where("id = ? AND production_order_id = ?", currentRunID, orderID).First(&currentRun).Error; err != nil {
			return errors.New("stage run not found")
		}

		now := time.Now()
		currentRun.Status = "COMPLETED"
		if currentRun.StartedAt == nil {
			currentRun.StartedAt = &now
		}
		currentRun.FinishedAt = &now
		if notes != "" {
			currentRun.Notes = notes
		}
		if techID != nil && *techID != uuid.Nil {
			currentRun.TechnicianID = techID
		}
		if err := tx.Save(&currentRun).Error; err != nil {
			return err
		}

		// Find next stage
		var nextRun ProductionStageRun
		err := tx.Where("production_order_id = ? AND stage_order > ?", orderID, currentRun.StageOrder).
			Order("stage_order ASC").
			First(&nextRun).Error

		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// All production stages completed -> QC_PENDING
				return tx.Model(&ProductionOrder{}).Where("id = ?", orderID).Updates(map[string]interface{}{
					"status":             "QC_PENDING",
					"current_stage_code": "QC_PENDING",
					"current_stage_name": "Menunggu Quality Control (QC)",
					"updated_at":         now,
				}).Error
			}
			return err
		}

		// Advance to next stage
		nextRun.Status = "IN_PROGRESS"
		nextRun.StartedAt = &now
		if nextTechID != nil && *nextTechID != uuid.Nil {
			nextRun.TechnicianID = nextTechID
		} else if currentRun.TechnicianID != nil {
			nextRun.TechnicianID = currentRun.TechnicianID
		}
		if err := tx.Save(&nextRun).Error; err != nil {
			return err
		}

		updates := map[string]interface{}{
			"current_stage_code": nextRun.StageCode,
			"current_stage_name": nextRun.StageName,
			"status":             "IN_PRODUCTION",
			"updated_at":         now,
		}
		if nextRun.TechnicianID != nil {
			updates["current_technician_id"] = *nextRun.TechnicianID
		}
		return tx.Model(&ProductionOrder{}).Where("id = ?", orderID).Updates(updates).Error
	})
}

// QC Review
func (s *Service) SubmitQualityCheck(orderID uuid.UUID, inspectorID uuid.UUID, passed bool, checklistResults string, defectNotes string, reworkStageCode string) (*QualityCheck, error) {
	if checklistResults == "" {
		checklistResults = "{}"
	}
	var qc QualityCheck
	err := s.db.Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		qc = QualityCheck{
			ProductionOrderID: orderID,
			InspectorID:       inspectorID,
			Passed:            passed,
			ChecklistResults:  checklistResults,
			DefectNotes:       defectNotes,
			ReworkTargetStage: reworkStageCode,
			CheckedAt:         now,
		}
		if err := tx.Create(&qc).Error; err != nil {
			return err
		}

		if passed {
			// Passed QC -> COMPLETED (Now ready for invoicing & delivery!)
			if err := tx.Model(&ProductionOrder{}).Where("id = ?", orderID).Updates(map[string]interface{}{
				"status":             "COMPLETED",
				"current_stage_code": "QC_PASSED",
				"current_stage_name": "Lolos QC (Selesai Produksi)",
				"completed_at":       &now,
				"updated_at":         now,
			}).Error; err != nil {
				return err
			}
			s.autoCreateInvoiceIfNeeded(tx, orderID)
			return nil
		}

		// Failed QC -> Reopen for rework!
		if reworkStageCode == "" {
			reworkStageCode = "CAST_MODEL"
		}

		// Mark target stage run as IN_PROGRESS and subsequent stages as PENDING
		var targetRun ProductionStageRun
		if err := tx.Where("production_order_id = ? AND stage_code = ?", orderID, reworkStageCode).First(&targetRun).Error; err == nil {
			tx.Model(&ProductionStageRun{}).
				Where("production_order_id = ? AND stage_order > ?", orderID, targetRun.StageOrder).
				Update("status", "PENDING")
		}

		tx.Model(&ProductionStageRun{}).
			Where("production_order_id = ? AND stage_code = ?", orderID, reworkStageCode).
			Updates(map[string]interface{}{
				"status": "IN_PROGRESS",
				"notes":  fmt.Sprintf("Rework QC: %s", defectNotes),
			})

		return tx.Model(&ProductionOrder{}).Where("id = ?", orderID).Updates(map[string]interface{}{
			"status":             "REOPENED_FOR_REWORK",
			"current_stage_code": reworkStageCode,
			"current_stage_name": fmt.Sprintf("Revisi/Rework: %s", reworkStageCode),
			"updated_at":         now,
		}).Error
	})

	if err != nil {
		return nil, err
	}
	return &qc, nil
}

func (s *Service) UpdateOrderFeedback(orderID uuid.UUID, rating int, notes string) (*ProductionOrder, error) {
	if err := s.db.Model(&ProductionOrder{}).Where("id = ?", orderID).Updates(map[string]interface{}{
		"feedback_rating": rating,
		"feedback_notes":  notes,
		"updated_at":      time.Now(),
	}).Error; err != nil {
		return nil, err
	}
	return s.GetOrderByID(orderID)
}

