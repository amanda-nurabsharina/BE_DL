package production

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"dental-lab-backend/internal/modules/auth"
	"dental-lab-backend/internal/modules/customers"
)

type DentureType struct {
	ID             int                     `gorm:"primaryKey" json:"id"`
	Code           string                  `gorm:"uniqueIndex;not null" json:"code"`
	Name           string                  `gorm:"not null" json:"name"`
	Category       string                  `json:"category"`
	BasePrice      float64                 `json:"base_price"`
	EstimatedDays  int                     `json:"estimated_days"`
	Description    string                  `json:"description"`
	IsActive       bool                    `json:"is_active"`
	StageTemplates []ProductionStageTemplate `gorm:"foreignKey:DentureTypeID" json:"stage_templates"`
	CreatedAt      time.Time               `json:"created_at"`
}

func (DentureType) TableName() string {
	return "denture_types"
}

type ProductionStageTemplate struct {
	ID            int    `gorm:"primaryKey" json:"id"`
	DentureTypeID int    `json:"denture_type_id"`
	StageOrder    int    `json:"stage_order"`
	StageCode     string `json:"stage_code"`
	StageName     string `json:"stage_name"`
	Description   string `json:"description"`
}

func (ProductionStageTemplate) TableName() string {
	return "production_stage_templates"
}

type ProductionOrder struct {
	ID                  uuid.UUID            `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	TenantID            *uuid.UUID           `gorm:"type:uuid" json:"tenant_id"`
	OrderNumber         string               `gorm:"uniqueIndex;not null" json:"order_number"`
	CustomerID          uuid.UUID            `json:"customer_id"`
	Customer            customers.Customer   `gorm:"foreignKey:CustomerID" json:"customer"`
	PatientID           *uuid.UUID           `json:"patient_id"`
	Patient             *customers.Patient   `gorm:"foreignKey:PatientID" json:"patient"`
	PatientName         string               `gorm:"not null" json:"patient_name"`
	DentureTypeID       int                  `json:"denture_type_id"`
	DentureType         DentureType          `gorm:"foreignKey:DentureTypeID" json:"denture_type"`
	TeethNumbers        string               `gorm:"type:jsonb" json:"teeth_numbers"` // e.g. ["11", "21"]
	ShadeGuide          string               `json:"shade_guide"`                     // e.g. "VITA A2"
	ReceivedDate        time.Time            `json:"received_date"`
	PromisedDueDate     time.Time            `json:"promised_due_date"`
	CompletedAt         *time.Time           `json:"completed_at"`
	DeliveredAt         *time.Time           `json:"delivered_at"`
	Status              string               `gorm:"default:'INTAKE'" json:"status"` // INTAKE, IN_PRODUCTION, QC_PENDING, QC_FAILED, REOPENED_FOR_REWORK, COMPLETED, DELIVERED, CANCELLED
	CurrentStageCode    string               `json:"current_stage_code"`
	CurrentStageName    string               `json:"current_stage_name"`
	CurrentTechnicianID *uuid.UUID           `json:"current_technician_id"`
	CurrentTechnician   *auth.User           `gorm:"foreignKey:CurrentTechnicianID" json:"current_technician"`
	Priority            string               `gorm:"default:'NORMAL'" json:"priority"` // NORMAL, URGENT
	DoctorNotes         string               `json:"doctor_notes"`
	InternalNotes       string               `json:"internal_notes"`
	DiscountPercent     float64              `gorm:"default:0" json:"discount_percent"`
	DiscountAmount      float64              `gorm:"default:0" json:"discount_amount"`
	TotalPrice          float64              `json:"total_price"`
	FeedbackRating      int                  `gorm:"default:0" json:"feedback_rating"`
	FeedbackNotes       string               `json:"feedback_notes"`
	StageRuns           []ProductionStageRun `gorm:"foreignKey:ProductionOrderID" json:"stage_runs"`
	QualityChecks       []QualityCheck       `gorm:"foreignKey:ProductionOrderID" json:"quality_checks"`
	Attachments         []OrderAttachment    `gorm:"foreignKey:ProductionOrderID" json:"attachments"`
	CreatedBy           *uuid.UUID           `json:"created_by"`
	CreatedAt           time.Time            `json:"created_at"`
	UpdatedAt           time.Time            `json:"updated_at"`
	DeletedAt           gorm.DeletedAt       `gorm:"index" json:"-"`
}

func (ProductionOrder) TableName() string {
	return "production_orders"
}

type ProductionStageRun struct {
	ID                uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	ProductionOrderID uuid.UUID  `json:"production_order_id"`
	StageCode         string     `json:"stage_code"`
	StageName         string     `json:"stage_name"`
	StageOrder        int        `json:"stage_order"`
	Status            string     `gorm:"default:'PENDING'" json:"status"` // PENDING, IN_PROGRESS, COMPLETED, REWORKED, SKIPPED
	TechnicianID      *uuid.UUID `json:"technician_id"`
	Technician        *auth.User `gorm:"foreignKey:TechnicianID" json:"technician"`
	StartedAt         *time.Time `json:"started_at"`
	FinishedAt        *time.Time `json:"finished_at"`
	Notes             string     `json:"notes"`
	CreatedAt         time.Time  `json:"created_at"`
}

func (ProductionStageRun) TableName() string {
	return "production_stage_runs"
}

type QualityCheck struct {
	ID                 uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	ProductionOrderID  uuid.UUID  `json:"production_order_id"`
	InspectorID        uuid.UUID  `json:"inspector_id"`
	Inspector          auth.User  `gorm:"foreignKey:InspectorID" json:"inspector"`
	Passed             bool       `json:"passed"`
	ChecklistResults   string     `gorm:"type:jsonb" json:"checklist_results"`
	DefectNotes        string     `json:"defect_notes"`
	ReworkTargetStage  string     `json:"rework_target_stage"`
	CheckedAt          time.Time  `json:"checked_at"`
	CreatedAt          time.Time  `json:"created_at"`
}

func (QualityCheck) TableName() string {
	return "quality_checks"
}

type OrderAttachment struct {
	ID                uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	ProductionOrderID uuid.UUID  `json:"production_order_id"`
	FileName          string     `json:"file_name"`
	StoredFilePath    string     `json:"stored_file_path"`
	FileSize          int64      `json:"file_size"`
	MimeType          string     `json:"mime_type"`
	AttachmentType    string     `json:"attachment_type"` // IMPRESSION_PHOTO, TEETH_PHOTO, STL_SCAN, QC_PHOTO
	Description       string     `json:"description"`
	UploadedBy        *uuid.UUID `json:"uploaded_by"`
	CreatedAt         time.Time  `json:"created_at"`
}

func (OrderAttachment) TableName() string {
	return "order_attachments"
}
