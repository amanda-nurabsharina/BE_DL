package auth

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Role struct {
	ID           int       `gorm:"primaryKey" json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	AllowedMenus string    `gorm:"type:jsonb" json:"allowed_menus"`
	CreatedAt    time.Time `json:"created_at"`
}

func (Role) TableName() string {
	return "roles"
}

type User struct {
	ID             uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	EmployeeCode   string         `json:"employee_code"`
	Username       string         `gorm:"uniqueIndex;not null" json:"username"`
	Email          string         `gorm:"uniqueIndex" json:"email"`
	PasswordHash   string         `json:"-"`
	FullName       string         `json:"full_name"`
	Phone          string         `json:"phone"`
	AvatarURL      string         `json:"avatar_url"`
	Specialization string         `json:"specialization"`
	Skills         string         `json:"skills"`
	WorkSchedule   string         `json:"work_schedule"`
	Address        string         `json:"address"`
	JoinDate       string         `json:"join_date"`
	Bio            string         `json:"bio"`
	RoleID         int            `json:"role_id"`
	Role           Role           `gorm:"foreignKey:RoleID" json:"role"`
	IsActive       bool           `json:"is_active"`
	// Computed HR & Production KPI fields
	ActiveOrdersCount     int     `gorm:"-" json:"active_orders_count"`
	CompletedOrdersCount  int     `gorm:"-" json:"completed_orders_count"`
	CompletedStagesCount  int     `gorm:"-" json:"completed_stages_count"`
	OnTimeRate            float64 `gorm:"-" json:"on_time_rate"`
	QCPassRate            float64 `gorm:"-" json:"qc_pass_rate"`
	AvgFeedbackRating     float64 `gorm:"-" json:"avg_feedback_rating"`
	TotalOvertimeHours    float64 `gorm:"-" json:"total_overtime_hours"`
	AttendancePresentDays int     `gorm:"-" json:"attendance_present_days"`
	CreatedAt             time.Time      `json:"created_at"`
	UpdatedAt             time.Time      `json:"updated_at"`
	DeletedAt             gorm.DeletedAt `gorm:"index" json:"-"`
}

func (User) TableName() string {
	return "users"
}

type StaffAttendance struct {
	ID             uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	UserID         uuid.UUID `gorm:"type:uuid;not null" json:"user_id"`
	User           User      `gorm:"foreignKey:UserID" json:"user"`
	AttendanceDate string    `gorm:"type:date;not null" json:"attendance_date"`
	ShiftSchedule  string    `json:"shift_schedule"`
	CheckIn        string    `json:"check_in"`
	CheckOut       string    `json:"check_out"`
	Status         string    `gorm:"default:'HADIR'" json:"status"` // HADIR, TERLAMBAT, IZIN, SAKIT, CUTI
	OvertimeHours  float64   `gorm:"default:0" json:"overtime_hours"`
	OvertimeNotes  string    `json:"overtime_notes"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (StaffAttendance) TableName() string {
	return "staff_attendances"
}

type SystemSetting struct {
	Key       string    `gorm:"primaryKey" json:"key"`
	Value     string    `json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (SystemSetting) TableName() string {
	return "system_settings"
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token     string `json:"token"`
	ExpiresIn string `json:"expires_in"`
	User      User   `json:"user"`
}

type CreateUserRequest struct {
	Username       string `json:"username"`
	Password       string `json:"password"`
	FullName       string `json:"full_name"`
	Email          string `json:"email"`
	Phone          string `json:"phone"`
	RoleID         int    `json:"role_id"`
	Specialization string `json:"specialization"`
	Skills         string `json:"skills"`
	WorkSchedule   string `json:"work_schedule"`
	Address        string `json:"address"`
	JoinDate       string `json:"join_date"`
	Bio            string `json:"bio"`
}

type UpdateUserRequest struct {
	FullName       *string `json:"full_name"`
	Email          *string `json:"email"`
	Phone          *string `json:"phone"`
	RoleID         *int    `json:"role_id"`
	IsActive       *bool   `json:"is_active"`
	Password       *string `json:"password"`
	Specialization *string `json:"specialization"`
	Skills         *string `json:"skills"`
	WorkSchedule   *string `json:"work_schedule"`
	Address        *string `json:"address"`
	JoinDate       *string `json:"join_date"`
	Bio            *string `json:"bio"`
	AvatarURL      *string `json:"avatar_url"`
}

type UpdateRoleMenusRequest struct {
	AllowedMenus string `json:"allowed_menus"` // JSON array string
}

type CreateAttendanceRequest struct {
	UserID         string  `json:"user_id"`
	AttendanceDate string  `json:"attendance_date"`
	ShiftSchedule  string  `json:"shift_schedule"`
	CheckIn        string  `json:"check_in"`
	CheckOut       string  `json:"check_out"`
	Status         string  `json:"status"`
	OvertimeHours  float64 `json:"overtime_hours"`
	OvertimeNotes  string  `json:"overtime_notes"`
}
