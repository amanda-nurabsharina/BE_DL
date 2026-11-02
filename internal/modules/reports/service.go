package reports

import (
	"time"

	"gorm.io/gorm"

	"dental-lab-backend/internal/modules/finance"
	"dental-lab-backend/internal/modules/inventory"
	"dental-lab-backend/internal/modules/production"
)

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

type DashboardSummary struct {
	ActiveOrdersCount    int64                        `json:"active_orders_count"`
	QCPendingCount       int64                        `json:"qc_pending_count"`
	CompletedThisMonth   int64                        `json:"completed_this_month"`
	TotalRevenueMonth    float64                      `json:"total_revenue_month"`
	TotalExpensesMonth   float64                      `json:"total_expenses_month"`
	NetOperatingIncome   float64                      `json:"net_operating_income"`
	TotalReceivables     float64                      `json:"total_receivables"`
	LowStockCount        int64                        `json:"low_stock_count"`
	RecentOrders         []production.ProductionOrder `json:"recent_orders"`
}

func (s *Service) GetDashboardSummary() (*DashboardSummary, error) {
	var summary DashboardSummary

	// Active orders
	s.db.Model(&production.ProductionOrder{}).
		Where("status IN ('INTAKE', 'IN_PRODUCTION', 'REOPENED_FOR_REWORK') AND deleted_at IS NULL").
		Count(&summary.ActiveOrdersCount)

	// QC Pending
	s.db.Model(&production.ProductionOrder{}).
		Where("status = 'QC_PENDING' AND deleted_at IS NULL").
		Count(&summary.QCPendingCount)

	// Completed this month
	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	s.db.Model(&production.ProductionOrder{}).
		Where("completed_at >= ? AND deleted_at IS NULL", startOfMonth).
		Count(&summary.CompletedThisMonth)

	// Revenue this month (paid payments)
	var revenueMonth float64
	s.db.Model(&finance.Payment{}).
		Where("payment_date >= ?", startOfMonth).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&revenueMonth)
	summary.TotalRevenueMonth = revenueMonth

	// Expenses this month
	var expensesMonth float64
	s.db.Model(&finance.Expense{}).
		Where("expense_date >= ?", startOfMonth).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&expensesMonth)
	summary.TotalExpensesMonth = expensesMonth

	summary.NetOperatingIncome = summary.TotalRevenueMonth - summary.TotalExpensesMonth

	// Total outstanding receivables (Piutang)
	var totalReceivables float64
	s.db.Model(&finance.Invoice{}).
		Where("status IN ('UNPAID', 'PARTIALLY_PAID') AND deleted_at IS NULL").
		Select("COALESCE(SUM(balance_due), 0)").
		Scan(&totalReceivables)
	summary.TotalReceivables = totalReceivables

	// Low stock items
	s.db.Model(&inventory.Item{}).
		Where("current_stock <= min_stock_level AND deleted_at IS NULL").
		Count(&summary.LowStockCount)

	// Recent 5 orders
	s.db.Preload("Customer").
		Preload("DentureType").
		Where("deleted_at IS NULL").
		Order("created_at DESC").
		Limit(5).
		Find(&summary.RecentOrders)

	return &summary, nil
}

type AgingReportItem struct {
	CustomerID   string  `json:"customer_id"`
	DoctorName   string  `json:"doctor_name"`
	ClinicName   string  `json:"clinic_name"`
	Phone        string  `json:"phone"`
	TotalInvoices int    `json:"total_invoices"`
	TotalReceivable float64 `json:"total_receivable"`
}

func (s *Service) GetReceivablesReport() ([]AgingReportItem, error) {
	var results []AgingReportItem
	type RawResult struct {
		CustomerID string
		DoctorName string
		ClinicName string
		Phone      string
		TotalInv   int
		TotalDue   float64
	}
	var raw []RawResult

	err := s.db.Table("invoices").
		Select("invoices.customer_id, customers.name as doctor_name, customers.clinic_name, customers.phone, count(invoices.id) as total_inv, sum(invoices.balance_due) as total_due").
		Joins("JOIN customers ON customers.id = invoices.customer_id").
		Where("invoices.status IN ('UNPAID', 'PARTIALLY_PAID') AND invoices.deleted_at IS NULL").
		Group("invoices.customer_id, customers.name, customers.clinic_name, customers.phone").
		Order("total_due DESC").
		Scan(&raw).Error

	if err != nil {
		return nil, err
	}

	for _, r := range raw {
		results = append(results, AgingReportItem{
			CustomerID:      r.CustomerID,
			DoctorName:      r.DoctorName,
			ClinicName:      r.ClinicName,
			Phone:           r.Phone,
			TotalInvoices:   r.TotalInv,
			TotalReceivable: r.TotalDue,
		})
	}

	return results, nil
}
