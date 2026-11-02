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
	return &Service{db: db}
}

func (s *Service) ListInvoices(status string, customerIDStr string, search string) ([]Invoice, error) {
	var list []Invoice
	q := s.db.Preload("Customer").
		Preload("ProductionOrder.DentureType").
		Preload("Payments").
		Where("deleted_at IS NULL")

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

		subtotal := order.TotalPrice
		total := subtotal - input.DiscountAmount
		if total < 0 {
			total = 0
		}

		if input.DueDate.IsZero() {
			input.DueDate = time.Now().AddDate(0, 0, 14) // 14 days credit terms default
		}

		inv = Invoice{
			InvoiceNumber:     invNum,
			ProductionOrderID: &order.ID,
			CustomerID:        order.CustomerID,
			IssueDate:         time.Now(),
			DueDate:           input.DueDate,
			Subtotal:          subtotal,
			DiscountAmount:    input.DiscountAmount,
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
			Discount:    input.DiscountAmount,
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

func (s *Service) ListExpenses() ([]Expense, error) {
	var expenses []Expense
	err := s.db.Preload("Category").Order("expense_date DESC, created_at DESC").Find(&expenses).Error
	return expenses, err
}

type CreateExpenseInput struct {
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

	return &exp, nil
}
