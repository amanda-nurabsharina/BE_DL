package finance

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"dental-lab-backend/internal/modules/production"
)

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	s := &Service{db: db}
	s.ensureFinanceSchemaAndSeed()
	return s
}

func (s *Service) ensureFinanceSchemaAndSeed() {
	// 1. Ensure standard expense categories exist (Biaya Bahan, Biaya Tenaga Kerja, Biaya Operasional)
	categories := []ExpenseCategory{
		{ID: 1, Code: "RAW_MATERIAL", Name: "Biaya Bahan Produksi (Akrilik, Gips, Zirkonia, Logam)"},
		{ID: 2, Code: "LABOR", Name: "Biaya Tenaga Kerja & Insentif Tekniker"},
		{ID: 3, Code: "OPERATIONAL", Name: "Biaya Operasional Lab (Listrik, Kurir, Perawatan Mesin)"},
	}
	for _, cat := range categories {
		var existing ExpenseCategory
		if err := s.db.Where("code = ? OR id = ?", cat.Code, cat.ID).First(&existing).Error; err != nil {
			_ = s.db.Create(&cat).Error
		} else {
			_ = s.db.Model(&existing).Updates(map[string]interface{}{
				"code": cat.Code,
				"name": cat.Name,
			}).Error
		}
	}

	// 2. Seed realistic sample expenses if none exist
	var expCount int64
	s.db.Model(&Expense{}).Count(&expCount)
	if expCount == 0 {
		var catMat, catLab, catOps ExpenseCategory
		s.db.Where("code = 'RAW_MATERIAL'").First(&catMat)
		s.db.Where("code = 'LABOR'").First(&catLab)
		s.db.Where("code = 'OPERATIONAL'").First(&catOps)

		now := time.Now()
		samples := []Expense{
			{
				ExpenseNumber: "EXP-0001",
				CategoryID:    catMat.ID,
				Amount:        850000,
				ExpenseDate:   now.AddDate(0, 0, -3),
				Description:   "Pembelian Blok Zirconia HT A2 & Gips Tipe IV Die Stone untuk produksi mahkota",
			},
			{
				ExpenseNumber: "EXP-0002",
				CategoryID:    catLab.ID,
				Amount:        650000,
				ExpenseDate:   now.AddDate(0, 0, -2),
				Description:   "Insentif produksi & uang lembur tekniker (Glazing & Finishing Crown Zirconia)",
			},
			{
				ExpenseNumber: "EXP-0003",
				CategoryID:    catOps.ID,
				Amount:        320000,
				ExpenseDate:   now.AddDate(0, 0, -1),
				Description:   "Biaya kurir antar-jemput cetakan dokter & token listrik mesin furnace porselen",
			},
		}
		for _, sp := range samples {
			if sp.CategoryID > 0 {
				_ = s.db.Create(&sp).Error
			}
		}
	}

	// 3. Seed at least one completed ProductionOrder if no COMPLETED/DELIVERED orders exist so Invoice & Hasil Order is immediately demonstrable
	var completedOrderCount int64
	s.db.Model(&production.ProductionOrder{}).Where("status IN ('COMPLETED', 'DELIVERED') AND deleted_at IS NULL").Count(&completedOrderCount)
	if completedOrderCount == 0 {
		var firstCustIDStr string
		s.db.Table("customers").Select("id").Where("deleted_at IS NULL").Limit(1).Scan(&firstCustIDStr)
		firstCustID, err := uuid.Parse(firstCustIDStr)
		if err == nil && firstCustID != uuid.Nil {
			var dtID int = 1
			s.db.Table("denture_types").Select("id").Limit(1).Scan(&dtID)
			var techIDStr string
			s.db.Table("users").Select("id").Where("deleted_at IS NULL").Limit(1).Scan(&techIDStr)
			techID, _ := uuid.Parse(techIDStr)

			var totalOrders int64
			s.db.Model(&production.ProductionOrder{}).Count(&totalOrders)
			now := time.Now()
			recDate := now.AddDate(0, 0, -3)
			dueDate := now.AddDate(0, 0, 1)

			var techPtr *uuid.UUID
			if techID != uuid.Nil {
				techPtr = &techID
			}

			sampleOrder := production.ProductionOrder{
				OrderNumber:         fmt.Sprintf("ORD-%04d", totalOrders+1),
				CustomerID:          firstCustID,
				PatientName:         "Bpk. Hendra Gunawan",
				DentureTypeID:       dtID,
				TeethNumbers:        `["11","21"]`,
				ShadeGuide:          "VITA A2",
				ReceivedDate:        recDate,
				PromisedDueDate:     dueDate,
				CompletedAt:         &now,
				Status:              "COMPLETED",
				CurrentStageCode:    "QC_PASSED",
				CurrentStageName:    "Lolos QC (Selesai Produksi)",
				CurrentTechnicianID: techPtr,
				Priority:            "NORMAL",
				DoctorNotes:         "Kontak proksimal rapat, anatomi natural incisal.",
				InternalNotes:       "Seluruh tahapan produksi & QC selesai tepat waktu.",
				DiscountPercent:     10,
				DiscountAmount:      180000,
				TotalPrice:          1620000,
				FeedbackRating:      5,
				FeedbackNotes:       "Marginal fit sangat presisi dan warna VITA A2 sesuai.",
			}
			if err := s.db.Create(&sampleOrder).Error; err == nil {
				if techID != uuid.Nil {
					_ = s.db.Create(&production.QualityCheck{
						ProductionOrderID: sampleOrder.ID,
						InspectorID:       techID,
						Passed:            true,
						ChecklistResults:  `{"oklusi":true,"marginal_fit":true,"kontak_proksimal":true,"shade_match":true,"polishing":true}`,
						DefectNotes:       "Lolos pemeriksaan mutu akhir (Oklusi, Marginal Fit, dan Estetika Glaze).",
						CheckedAt:         now,
					}).Error
				}
			}
		}
	}

	// 4. Automatically sync invoices for all COMPLETED / DELIVERED production orders
	_, _ = s.SyncInvoicesFromProduction(nil, nil)
}

func (s *Service) ListInvoices(status string, customerIDStr string, search string, tenantID *uuid.UUID) ([]Invoice, error) {
	// Ensure any newly completed orders have their invoices generated automatically
	_, _ = s.SyncInvoicesFromProduction(nil, tenantID)

	var list []Invoice
	q := s.db.Preload("Customer").
		Preload("ProductionOrder.DentureType").
		Preload("ProductionOrder.Attachments").
		Preload("ProductionOrder.QualityChecks").
		Preload("Items").
		Preload("Payments").
		Where("deleted_at IS NULL")

	if tenantID != nil {
		q = q.Where("tenant_id = ?", *tenantID)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if customerIDStr != "" {
		if cid, err := uuid.Parse(customerIDStr); err == nil {
			q = q.Where("customer_id = ?", cid)
		}
	}
	if search != "" {
		likeTerm := "%" + search + "%"
		q = q.Where("invoice_number ILIKE ?", likeTerm)
	}

	err := q.Order("issue_date DESC, created_at DESC").Find(&list).Error
	return list, err
}

func (s *Service) GetInvoiceByID(id uuid.UUID) (*Invoice, error) {
	var inv Invoice
	err := s.db.Preload("Customer").
		Preload("ProductionOrder.DentureType").
		Preload("ProductionOrder.Attachments").
		Preload("ProductionOrder.QualityChecks").
		Preload("Items").
		Preload("Payments").
		Where("id = ? AND deleted_at IS NULL", id).
		First(&inv).Error

	if err != nil {
		return nil, err
	}
	return &inv, nil
}

type CreateInvoiceInput struct {
	ProductionOrderID uuid.UUID  `json:"production_order_id"`
	DueDate           time.Time  `json:"due_date"`
	DiscountAmount    float64    `json:"discount_amount"`
	Notes             string     `json:"notes"`
	CreatedBy         *uuid.UUID `json:"-"`
}

func (s *Service) CreateInvoiceFromOrder(input CreateInvoiceInput) (*Invoice, error) {
	var order production.ProductionOrder
	if err := s.db.Preload("DentureType").Where("id = ?", input.ProductionOrderID).First(&order).Error; err != nil {
		return nil, errors.New("production order not found")
	}

	// Invoicing rule check: Issued upon Production Completion
	if order.Status != "COMPLETED" && order.Status != "DELIVERED" {
		return nil, fmt.Errorf("invoice can only be issued after production completion (current status: %s)", order.Status)
	}

	var existingInv Invoice
	if err := s.db.Where("production_order_id = ? AND status != 'VOID'", input.ProductionOrderID).First(&existingInv).Error; err == nil {
		return nil, fmt.Errorf("an invoice (%s) has already been issued for this production order", existingInv.InvoiceNumber)
	}

	var inv Invoice
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var count int64
		tx.Model(&Invoice{}).Count(&count)
		invNum := fmt.Sprintf("INV-%04d", count+1)

		disc := input.DiscountAmount
		if disc == 0 && order.DiscountAmount > 0 {
			disc = order.DiscountAmount
		}
		subtotal := order.TotalPrice + order.DiscountAmount
		if subtotal <= 0 {
			subtotal = order.TotalPrice
		}
		total := subtotal - disc
		if total < 0 {
			total = 0
		}

		if input.DueDate.IsZero() {
			input.DueDate = time.Now().AddDate(0, 0, 14) // 14 days credit terms default
		}

		inv = Invoice{
			TenantID:          order.TenantID,
			InvoiceNumber:     invNum,
			ProductionOrderID: &order.ID,
			CustomerID:        order.CustomerID,
			IssueDate:         time.Now(),
			DueDate:           input.DueDate,
			Subtotal:          subtotal,
			DiscountAmount:    disc,
			TotalAmount:       total,
			PaidAmount:        0,
			BalanceDue:        total,
			Status:            "UNPAID",
			Notes:             input.Notes,
			CreatedBy:         input.CreatedBy,
		}

		if err := tx.Create(&inv).Error; err != nil {
			return err
		}

		// Item line
		item := InvoiceItem{
			InvoiceID:   inv.ID,
			Description: fmt.Sprintf("%s - Pasien: %s (Gigi: %s)", order.DentureType.Name, order.PatientName, order.TeethNumbers),
			Quantity:    1,
			UnitPrice:   subtotal,
			Discount:    disc,
			Subtotal:    total,
		}
		return tx.Create(&item).Error
	})

	if err != nil {
		return nil, err
	}

	return s.GetInvoiceByID(inv.ID)
}

type RecordPaymentInput struct {
	InvoiceID       uuid.UUID  `json:"invoice_id"`
	Amount          float64    `json:"amount"`
	PaymentMethod   string     `json:"payment_method"`
	PaymentType     string     `json:"payment_type"`
	PaymentDate     time.Time  `json:"payment_date"`
	ReferenceNumber string     `json:"reference_number"`
	Notes           string     `json:"notes"`
	ReceivedBy      *uuid.UUID `json:"-"`
}

func (s *Service) RecordPayment(input RecordPaymentInput) (*Payment, error) {
	if input.Amount <= 0 {
		return nil, errors.New("payment amount must be greater than zero")
	}

	var payment Payment
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var inv Invoice
		if err := tx.Where("id = ? AND status != 'VOID'", input.InvoiceID).First(&inv).Error; err != nil {
			return errors.New("invoice not found or void")
		}

		var count int64
		tx.Model(&Payment{}).Count(&count)
		payNum := fmt.Sprintf("PAY-%04d", count+1)

		payDate := input.PaymentDate
		if payDate.IsZero() {
			payDate = time.Now()
		}

		payment = Payment{
			TenantID:        inv.TenantID,
			PaymentNumber:   payNum,
			InvoiceID:       &inv.ID,
			CustomerID:      inv.CustomerID,
			Amount:          input.Amount,
			PaymentMethod:   input.PaymentMethod,
			PaymentType:     input.PaymentType,
			PaymentDate:     payDate,
			ReferenceNumber: input.ReferenceNumber,
			Notes:           input.Notes,
			ReceivedBy:      input.ReceivedBy,
		}

		if err := tx.Create(&payment).Error; err != nil {
			return err
		}

		// Update invoice balance
		newPaid := inv.PaidAmount + input.Amount
		newBalance := inv.TotalAmount - newPaid

		newStatus := "PARTIALLY_PAID"
		if newBalance <= 0 {
			if newBalance < 0 {
				newStatus = "OVERPAID"
			} else {
				newStatus = "PAID"
			}
		}

		return tx.Model(&inv).Updates(map[string]interface{}{
			"paid_amount": newPaid,
			"balance_due": newBalance,
			"status":      newStatus,
			"updated_at":  time.Now(),
		}).Error
	})

	if err != nil {
		return nil, err
	}
	return &payment, nil
}

// Expenses
func (s *Service) ListExpenseCategories() ([]ExpenseCategory, error) {
	var cats []ExpenseCategory
	err := s.db.Find(&cats).Error
	return cats, err
}

func (s *Service) ListExpenses(tenantID *uuid.UUID) ([]Expense, error) {
	var expenses []Expense
	q := s.db.Preload("Category")
	if tenantID != nil {
		q = q.Where("tenant_id = ?", *tenantID)
	}
	err := q.Order("expense_date DESC, created_at DESC").Find(&expenses).Error
	return expenses, err
}

type CreateExpenseInput struct {
	TenantID    *uuid.UUID `json:"tenant_id"`
	CategoryID  int        `json:"category_id"`
	Amount      float64    `json:"amount"`
	ExpenseDate time.Time  `json:"expense_date"`
	Description string     `json:"description"`
	RecordedBy  *uuid.UUID `json:"-"`
}

func (s *Service) CreateExpense(input CreateExpenseInput) (*Expense, error) {
	if input.Amount <= 0 {
		return nil, errors.New("amount must be greater than zero")
	}

	var count int64
	s.db.Model(&Expense{}).Count(&count)
	expNum := fmt.Sprintf("EXP-%04d", count+1)

	expDate := input.ExpenseDate
	if expDate.IsZero() {
		expDate = time.Now()
	}

	exp := Expense{
		TenantID:      input.TenantID,
		ExpenseNumber: expNum,
		CategoryID:    input.CategoryID,
		Amount:        input.Amount,
		ExpenseDate:   expDate,
		Description:   input.Description,
		RecordedBy:    input.RecordedBy,
	}

	if err := s.db.Create(&exp).Error; err != nil {
		return nil, err
	}

	s.db.Preload("Category").First(&exp, "id = ?", exp.ID)
	return &exp, nil
}

// SyncInvoicesFromProduction automatically creates invoices for all COMPLETED / DELIVERED production orders that don't have one yet
func (s *Service) SyncInvoicesFromProduction(createdBy *uuid.UUID, tenantID *uuid.UUID) (int, error) {
	var completedOrders []production.ProductionOrder
	q := s.db.Preload("DentureType").
		Where("status IN ('COMPLETED', 'DELIVERED') AND deleted_at IS NULL")
	if tenantID != nil {
		q = q.Where("tenant_id = ?", *tenantID)
	}
	err := q.Order("created_at ASC").Find(&completedOrders).Error
	if err != nil {
		return 0, err
	}

	createdCount := 0
	for _, order := range completedOrders {
		var existingCount int64
		s.db.Model(&Invoice{}).Where("production_order_id = ? AND status != 'VOID' AND deleted_at IS NULL", order.ID).Count(&existingCount)
		if existingCount > 0 {
			continue
		}

		subtotal := order.TotalPrice + order.DiscountAmount
		if subtotal <= 0 {
			subtotal = order.TotalPrice
		}
		_, err := s.CreateInvoiceFromOrder(CreateInvoiceInput{
			ProductionOrderID: order.ID,
			DueDate:           time.Now().AddDate(0, 0, 14),
			DiscountAmount:    order.DiscountAmount,
			Notes:             fmt.Sprintf("Invoice otomatis dari penyelesaian SPK %s (Pasien: %s)", order.OrderNumber, order.PatientName),
			CreatedBy:         createdBy,
		})
		if err == nil {
			createdCount++
		}
	}
	return createdCount, nil
}

// GetFinanceSummary computes daily/monthly sales omzet, gross profit, expenses by category, and active customers
func (s *Service) GetFinanceSummary(tenantID *uuid.UUID) (*FinanceReportSummary, error) {
	_, _ = s.SyncInvoicesFromProduction(nil, tenantID)

	var invoices []Invoice
	invQ := s.db.Preload("Customer").Where("status != 'VOID' AND deleted_at IS NULL")
	if tenantID != nil {
		invQ = invQ.Where("tenant_id = ?", *tenantID)
	}
	invQ.Order("issue_date DESC").Find(&invoices)

	var expenses []Expense
	expQ := s.db.Preload("Category")
	if tenantID != nil {
		expQ = expQ.Where("tenant_id = ?", *tenantID)
	}
	expQ.Order("expense_date DESC").Find(&expenses)

	now := time.Now()
	todayStr := now.Format("2006-01-02")
	monthStr := now.Format("2006-01")

	summary := &FinanceReportSummary{}
	activeCustMap := map[string]bool{}
	dailyMap := map[string]*SalesPeriodItem{}
	monthlyMap := map[string]*SalesPeriodItem{}
	var dailyKeys []string
	var monthlyKeys []string

	for _, inv := range invoices {
		invDay := inv.IssueDate.Format("2006-01-02")
		invMonth := inv.IssueDate.Format("2006-01")
		activeCustMap[inv.CustomerID.String()] = true

		summary.TotalInvoiced += inv.TotalAmount
		summary.TotalPaid += inv.PaidAmount
		summary.TotalReceivables += inv.BalanceDue

		if inv.Status == "PAID" || inv.BalanceDue <= 0 {
			summary.PaidInvoicesCount++
		} else {
			summary.UnpaidInvoicesCount++
		}

		if invDay == todayStr {
			summary.DailyOmzet += inv.TotalAmount
			summary.DailyOrdersCount++
		}
		if invMonth == monthStr {
			summary.MonthlyOmzet += inv.TotalAmount
			summary.MonthlyOrdersCount++
		}

		// Group daily
		if _, ok := dailyMap[invDay]; !ok {
			dailyKeys = append(dailyKeys, invDay)
			dailyMap[invDay] = &SalesPeriodItem{PeriodLabel: invDay}
		}
		dItem := dailyMap[invDay]
		dItem.OrdersCount++
		dItem.GrossSales += inv.Subtotal
		dItem.Discounts += inv.DiscountAmount
		dItem.NetOmzet += inv.TotalAmount
		dItem.PaidCollected += inv.PaidAmount
		dItem.UnpaidBalance += inv.BalanceDue

		// Group monthly
		if _, ok := monthlyMap[invMonth]; !ok {
			monthlyKeys = append(monthlyKeys, invMonth)
			monthlyMap[invMonth] = &SalesPeriodItem{PeriodLabel: invMonth}
		}
		mItem := monthlyMap[invMonth]
		mItem.OrdersCount++
		mItem.GrossSales += inv.Subtotal
		mItem.Discounts += inv.DiscountAmount
		mItem.NetOmzet += inv.TotalAmount
		mItem.PaidCollected += inv.PaidAmount
		mItem.UnpaidBalance += inv.BalanceDue
	}

	//Also count active customers from production orders
	var orderCustIDs []string
	custQ := s.db.Table("production_orders").Where("deleted_at IS NULL")
	if tenantID != nil {
		custQ = custQ.Where("tenant_id = ?", *tenantID)
	}
	custQ.Distinct("customer_id").Pluck("customer_id", &orderCustIDs)
	for _, cid := range orderCustIDs {
		activeCustMap[cid] = true
	}
	summary.ActiveCustomersCount = len(activeCustMap)

	// Calculate expenses by category (Biaya Bahan, Tenaga Kerja, Operasional)
	for _, exp := range expenses {
		summary.TotalExpenses += exp.Amount
		switch exp.Category.Code {
		case "RAW_MATERIAL", "MATERIAL":
			summary.MaterialCost += exp.Amount
		case "LABOR", "SALARY", "PAYROLL":
			summary.LaborCost += exp.Amount
		default:
			summary.OperationalCost += exp.Amount
		}
	}

	// Laba Kotor (Gross Profit) = Total Omzet Penjualan - (Biaya Bahan + Biaya Tenaga Kerja Produksi)
	summary.GrossProfit = summary.TotalInvoiced - (summary.MaterialCost + summary.LaborCost)
	// Laba Bersih Operasional (Net Operating Profit) = Laba Kotor - Biaya Operasional
	summary.NetOperatingProfit = summary.GrossProfit - summary.OperationalCost

	for _, k := range dailyKeys {
		summary.DailyBreakdown = append(summary.DailyBreakdown, *dailyMap[k])
	}
	for _, k := range monthlyKeys {
		summary.MonthlyBreakdown = append(summary.MonthlyBreakdown, *monthlyMap[k])
	}

	return summary, nil
}
