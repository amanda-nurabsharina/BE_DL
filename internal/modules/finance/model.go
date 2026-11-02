package finance

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"dental-lab-backend/internal/modules/customers"
	"dental-lab-backend/internal/modules/production"
)

type Invoice struct {
	ID                uuid.UUID                   `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	InvoiceNumber     string                      `gorm:"uniqueIndex;not null" json:"invoice_number"`
	ProductionOrderID *uuid.UUID                  `json:"production_order_id"`
	ProductionOrder   *production.ProductionOrder `gorm:"foreignKey:ProductionOrderID" json:"production_order"`
	CustomerID        uuid.UUID                   `json:"customer_id"`
	Customer          customers.Customer          `gorm:"foreignKey:CustomerID" json:"customer"`
	IssueDate         time.Time                   `json:"issue_date"`
	DueDate           time.Time                   `json:"due_date"`
	Subtotal          float64                     `json:"subtotal"`
	DiscountAmount    float64                     `json:"discount_amount"`
	TotalAmount       float64                     `json:"total_amount"`
	PaidAmount        float64                     `json:"paid_amount"`
	BalanceDue        float64                     `json:"balance_due"`
	Status            string                      `gorm:"default:'UNPAID'" json:"status"` // UNPAID, PARTIALLY_PAID, PAID, OVERPAID, VOID
	Notes             string                      `json:"notes"`
	Items             []InvoiceItem               `gorm:"foreignKey:InvoiceID" json:"items"`
	Payments          []Payment                   `gorm:"foreignKey:InvoiceID" json:"payments"`
	CreatedBy         *uuid.UUID                  `json:"created_by"`
	CreatedAt         time.Time                   `json:"created_at"`
	UpdatedAt         time.Time                   `json:"updated_at"`
	DeletedAt         gorm.DeletedAt              `gorm:"index" json:"-"`
}

func (Invoice) TableName() string {
	return "invoices"
}

type InvoiceItem struct {
	ID          uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	InvoiceID   uuid.UUID `json:"invoice_id"`
	Description string    `json:"description"`
	Quantity    float64   `json:"quantity"`
	UnitPrice   float64   `json:"unit_price"`
	Discount    float64   `json:"discount"`
	Subtotal    float64   `json:"subtotal"`
}

func (InvoiceItem) TableName() string {
	return "invoice_items"
}

type Payment struct {
	ID              uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	PaymentNumber   string     `gorm:"uniqueIndex;not null" json:"payment_number"`
	InvoiceID       *uuid.UUID `json:"invoice_id"`
	CustomerID      uuid.UUID  `json:"customer_id"`
	Amount          float64    `json:"amount"`
	PaymentMethod   string     `gorm:"default:'BANK_TRANSFER'" json:"payment_method"` // CASH, BANK_TRANSFER, QRIS, OTHER
	PaymentType     string     `gorm:"default:'PARTIAL'" json:"payment_type"`         // DEPOSIT, PARTIAL, FINAL, OVERPAYMENT_CREDIT, REFUND
	PaymentDate     time.Time  `json:"payment_date"`
	ReferenceNumber string     `json:"reference_number"`
	Notes           string     `json:"notes"`
	ReceivedBy      *uuid.UUID `json:"received_by"`
	CreatedAt       time.Time  `json:"created_at"`
}

func (Payment) TableName() string {
	return "payments"
}

type ExpenseCategory struct {
	ID   int    `gorm:"primaryKey" json:"id"`
	Name string `json:"name"`
	Code string `gorm:"uniqueIndex;not null" json:"code"`
}

func (ExpenseCategory) TableName() string {
	return "expense_categories"
}

type Expense struct {
	ID            uuid.UUID       `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	ExpenseNumber string          `gorm:"uniqueIndex;not null" json:"expense_number"`
	CategoryID    int             `json:"category_id"`
	Category      ExpenseCategory `gorm:"foreignKey:CategoryID" json:"category"`
	Amount        float64         `json:"amount"`
	ExpenseDate   time.Time       `json:"expense_date"`
	Description   string          `json:"description"`
	ReceiptFile   string          `json:"receipt_file"`
	RecordedBy    *uuid.UUID      `json:"recorded_by"`
	CreatedAt     time.Time       `json:"created_at"`
}

func (Expense) TableName() string {
	return "expenses"
}

type SalesPeriodItem struct {
	PeriodLabel   string  `json:"period_label"`
	OrdersCount   int     `json:"orders_count"`
	GrossSales    float64 `json:"gross_sales"`
	Discounts     float64 `json:"discounts"`
	NetOmzet      float64 `json:"net_omzet"`
	PaidCollected float64 `json:"paid_collected"`
	UnpaidBalance float64 `json:"unpaid_balance"`
}

type FinanceReportSummary struct {
	DailyOmzet           float64           `json:"daily_omzet"`
	DailyOrdersCount     int               `json:"daily_orders_count"`
	MonthlyOmzet         float64           `json:"monthly_omzet"`
	MonthlyOrdersCount   int               `json:"monthly_orders_count"`
	TotalInvoiced        float64           `json:"total_invoiced"`
	TotalPaid            float64           `json:"total_paid"`
	TotalReceivables     float64           `json:"total_receivables"`
	PaidInvoicesCount    int               `json:"paid_invoices_count"`
	UnpaidInvoicesCount  int               `json:"unpaid_invoices_count"`
	MaterialCost         float64           `json:"material_cost"`
	LaborCost            float64           `json:"labor_cost"`
	OperationalCost      float64           `json:"operational_cost"`
	TotalExpenses        float64           `json:"total_expenses"`
	GrossProfit          float64           `json:"gross_profit"`
	NetOperatingProfit   float64           `json:"net_operating_profit"`
	ActiveCustomersCount int               `json:"active_customers_count"`
	DailyBreakdown       []SalesPeriodItem `json:"daily_breakdown"`
	MonthlyBreakdown     []SalesPeriodItem `json:"monthly_breakdown"`
}
