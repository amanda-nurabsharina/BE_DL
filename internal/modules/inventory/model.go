package inventory

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UnitOfMeasure struct {
	ID   int    `gorm:"primaryKey" json:"id"`
	Code string `gorm:"uniqueIndex;not null" json:"code"`
	Name string `gorm:"not null" json:"name"`
}

func (UnitOfMeasure) TableName() string {
	return "units_of_measure"
}

type Supplier struct {
	ID            uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Code          string         `gorm:"uniqueIndex;not null" json:"code"`
	Name          string         `gorm:"not null" json:"name"`
	ContactPerson string         `json:"contact_person"`
	Phone         string         `json:"phone"`
	Email         string         `json:"email"`
	Address       string         `json:"address"`
	IsActive      bool           `gorm:"default:true" json:"is_active"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Supplier) TableName() string {
	return "suppliers"
}

type Item struct {
	ID            uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Code          string         `gorm:"uniqueIndex;not null" json:"code"`
	Name          string         `gorm:"not null" json:"name"`
	Category      string         `gorm:"default:'RAW_MATERIAL'" json:"category"` // RAW_MATERIAL, PRODUCT, CONSUMABLE, TOOL
	UnitID        int            `json:"unit_id"`
	Unit          UnitOfMeasure  `gorm:"foreignKey:UnitID" json:"unit"`
	SupplierID    *uuid.UUID     `json:"supplier_id"`
	Supplier      *Supplier      `gorm:"foreignKey:SupplierID" json:"supplier"`
	MinStockLevel float64        `gorm:"default:0" json:"min_stock_level"`
	CurrentStock  float64        `gorm:"default:0" json:"current_stock"`
	PurchasePrice float64        `gorm:"default:0" json:"purchase_price"`
	LotNumber     *string        `json:"lot_number"`
	ExpiryDate    *time.Time     `json:"expiry_date"`
	Description   string         `json:"description"`
	IsActive      bool           `gorm:"default:true" json:"is_active"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Item) TableName() string {
	return "items"
}

type InventoryBatch struct {
	ID              uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	ItemID          uuid.UUID  `json:"item_id"`
	Item            Item       `gorm:"foreignKey:ItemID" json:"item"`
	LotNumber       *string    `json:"lot_number"`  // Optional
	ExpiryDate      *time.Time `json:"expiry_date"` // Optional
	InitialQuantity float64    `json:"initial_quantity"`
	CurrentQuantity float64    `json:"current_quantity"`
	UnitCost        float64    `json:"unit_cost"`
	ReceivedDate    time.Time  `json:"received_date"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

func (InventoryBatch) TableName() string {
	return "inventory_batches"
}

type StockMovement struct {
	ID            uuid.UUID       `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	MovementType  string          `json:"movement_type"` // RECEIPT, ISSUE_PRODUCTION, ADJUSTMENT, RETURN
	ItemID        uuid.UUID       `json:"item_id"`
	Item          Item            `gorm:"foreignKey:ItemID" json:"item"`
	BatchID       *uuid.UUID      `json:"batch_id"`
	Batch         *InventoryBatch `gorm:"foreignKey:BatchID" json:"batch"`
	Quantity      float64         `json:"quantity"` // positive for IN, negative for OUT
	UnitCost      float64         `json:"unit_cost"`
	ReferenceType string          `json:"reference_type"`
	ReferenceID   *uuid.UUID      `json:"reference_id"`
	Notes         string          `json:"notes"`
	CreatedBy     *uuid.UUID      `json:"created_by"`
	CreatedAt     time.Time       `json:"created_at"`
}

func (StockMovement) TableName() string {
	return "stock_movements"
}

// Purchase Request with Owner Approval
type PurchaseRequest struct {
	ID              uuid.UUID             `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	PRNumber        string                `gorm:"uniqueIndex;not null" json:"pr_number"`
	SupplierID      *uuid.UUID            `json:"supplier_id"`
	Supplier        *Supplier             `gorm:"foreignKey:SupplierID" json:"supplier"`
	Status          string                `gorm:"default:'DRAFT'" json:"status"` // DRAFT, PENDING_APPROVAL, APPROVED_BY_OWNER, REJECTED, ORDERED, RECEIVED, CANCELLED
	RequestedBy     uuid.UUID             `json:"requested_by"`
	ApprovedBy      *uuid.UUID            `json:"approved_by"`
	ApprovedAt      *time.Time            `json:"approved_at"`
	RejectionReason string                `json:"rejection_reason"`
	TotalAmount     float64               `gorm:"default:0" json:"total_amount"`
	Notes           string                `json:"notes"`
	Items           []PurchaseRequestItem `gorm:"foreignKey:PurchaseRequestID" json:"items"`
	CreatedAt       time.Time             `json:"created_at"`
	UpdatedAt       time.Time             `json:"updated_at"`
}

func (PurchaseRequest) TableName() string {
	return "purchase_requests"
}

type PurchaseRequestItem struct {
	ID                 uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	PurchaseRequestID  uuid.UUID `json:"purchase_request_id"`
	ItemID             uuid.UUID `json:"item_id"`
	Item               Item      `gorm:"foreignKey:ItemID" json:"item"`
	Quantity           float64   `json:"quantity"`
	EstimatedUnitPrice float64   `json:"estimated_unit_price"`
	ActualUnitPrice    float64   `json:"actual_unit_price"`
	Subtotal           float64   `json:"subtotal"`
	Notes              string    `json:"notes"`
}

func (PurchaseRequestItem) TableName() string {
	return "purchase_request_items"
}

type TopUsedMaterial struct {
	ItemID     uuid.UUID `json:"item_id"`
	ItemCode   string    `json:"item_code"`
	ItemName   string    `json:"item_name"`
	Category   string    `json:"category"`
	UnitCode   string    `json:"unit_code"`
	TotalUsed  float64   `json:"total_used"`
	UsageCount int64     `json:"usage_count"`
	TotalValue float64   `json:"total_value"`
}

type InventoryReportSummary struct {
	TotalItems        int64             `json:"total_items"`
	TotalStockValue   float64           `json:"total_stock_value"`
	LowStockCount     int64             `json:"low_stock_count"`
	ExpiringSoonCount int64             `json:"expiring_soon_count"`
	ExpiredCount      int64             `json:"expired_count"`
	LowStockItems     []Item            `json:"low_stock_items"`
	TopUsedMaterials  []TopUsedMaterial `json:"top_used_materials"`
}
