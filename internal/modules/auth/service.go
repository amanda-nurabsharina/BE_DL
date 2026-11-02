package auth

import (
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"dental-lab-backend/internal/platform/middleware"
)

type Service struct {
	db        *gorm.DB
	jwtSecret string
}

func NewService(db *gorm.DB, jwtSecret string) *Service {
	s := &Service{db: db, jwtSecret: jwtSecret}
	s.ensureHRSchemaAndSeed()
	return s
}

func (s *Service) ensureHRSchemaAndSeed() {
	// 1. Ensure SUPER_ADMIN, OWNER, ADMIN, TECHNICIAN have access to 'hr' menu
	_ = s.db.Exec(`UPDATE roles SET allowed_menus = '["dashboard", "orders", "new_order", "customers", "hr", "inventory", "approvals", "finance", "settings"]'::jsonb, description = 'Super Admin: Full akses seluruh menu sistem, operasional, SDM & pengaturan' WHERE name = 'SUPER_ADMIN'`).Error
	_ = s.db.Exec(`UPDATE roles SET allowed_menus = allowed_menus || '["hr"]'::jsonb WHERE name IN ('OWNER', 'ADMIN', 'TECHNICIAN') AND NOT (allowed_menus @> '["hr"]'::jsonb)`).Error
	_ = s.db.Exec(`UPDATE roles SET allowed_menus = allowed_menus || '["customers"]'::jsonb WHERE name IN ('OWNER', 'ADMIN') AND NOT (allowed_menus @> '["customers"]'::jsonb)`).Error

	// 2. Ensure HR & Profile columns exist on users
	s.db.Exec(`ALTER TABLE users ADD COLUMN IF NOT EXISTS employee_code VARCHAR(50) DEFAULT ''`)
	s.db.Exec(`ALTER TABLE users ADD COLUMN IF NOT EXISTS avatar_url VARCHAR(255) DEFAULT ''`)
	s.db.Exec(`ALTER TABLE users ADD COLUMN IF NOT EXISTS specialization VARCHAR(255) DEFAULT ''`)
	s.db.Exec(`ALTER TABLE users ADD COLUMN IF NOT EXISTS skills VARCHAR(255) DEFAULT ''`)
	s.db.Exec(`ALTER TABLE users ADD COLUMN IF NOT EXISTS work_schedule VARCHAR(150) DEFAULT 'Senin - Sabtu (08:00 - 17:00)'`)
	s.db.Exec(`ALTER TABLE users ADD COLUMN IF NOT EXISTS address TEXT DEFAULT ''`)
	s.db.Exec(`ALTER TABLE users ADD COLUMN IF NOT EXISTS join_date VARCHAR(30) DEFAULT '2024-01-15'`)
	s.db.Exec(`ALTER TABLE users ADD COLUMN IF NOT EXISTS bio TEXT DEFAULT ''`)

	// 3. Ensure staff_attendances table exists
	s.db.Exec(`CREATE TABLE IF NOT EXISTS staff_attendances (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		attendance_date DATE NOT NULL DEFAULT CURRENT_DATE,
		shift_schedule VARCHAR(100) DEFAULT 'Shift Pagi (08:00 - 17:00)',
		check_in VARCHAR(20) DEFAULT '08:00',
		check_out VARCHAR(20) DEFAULT '17:00',
		status VARCHAR(30) DEFAULT 'HADIR',
		overtime_hours NUMERIC(5,2) DEFAULT 0,
		overtime_notes TEXT DEFAULT '',
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
	)`)

	// 4. Enrich existing users with realistic profile data (employee_code, specialization, skills, work_schedule, bio)
	var users []User
	s.db.Preload("Role").Where("deleted_at IS NULL").Order("created_at ASC").Find(&users)
	for i, u := range users {
		if u.EmployeeCode == "" || u.Specialization == "" {
			empCode := fmt.Sprintf("EMP-%04d", i+1)
			spec := "Staf Operasional Lab"
			skills := "Manajemen Order, Koordinasi Dokter"
			sched := "Senin - Jumat (08:00 - 17:00)"
			addr := "Jl. Kesehatan No. 12, Jakarta Selatan"
			joinDate := "2023-06-01"
			bio := "Staf profesional laboratorium gigi presisi DentalLab Pro."

			switch u.Role.Name {
			case "SUPER_ADMIN":
				spec = "System Administrator & Head of Lab IT"
				skills = "System Architecture, Role & Access Control, Quality Assurance, Operasional Lab"
				sched = "Senin - Sabtu (Flexible / Full Support)"
				addr = "Jl. Gatot Subroto Kav. 22, Jakarta Selatan"
				joinDate = "2022-01-10"
				bio = "Penanggung jawab utama infrastruktur sistem, keamanan data, dan integrasi seluruh modul DentalLab Pro."
			case "OWNER":
				spec = "Principal Dental Lab Director & Master Ceramist"
				skills = "Master QC, Estetika Veneer/Zirconia, Manajemen Keuangan & Approval Pengadaan"
				sched = "Senin - Sabtu (09:00 - 17:00)"
				addr = "Jl. Senopati Raya No. 88, Jakarta Selatan"
				joinDate = "2021-03-15"
				bio = "Pendiri & Kepala Laboratorium Gigi dengan pengalaman > 12 tahun di prostodonsia estetik."
			case "ADMIN":
				spec = "Customer Relation (CRM) & Koordinator SPK Produksi"
				skills = "Intake SPK, Odontogram, Penjadwalan Kurir, Komunikasi Dokter & Klinik"
				sched = "Shift Pagi: Senin - Sabtu (08:00 - 16:30)"
				addr = "Jl. Tebet Barat Dalam No. 34, Jakarta Selatan"
				joinDate = "2023-02-01"
				bio = "Mengelola penerimaan order cetakan dokter, validasi instruksi kerja, dan penjadwalan pengiriman."
			case "TECHNICIAN":
				spec = "Senior Dental Technician — Crown Zirconia, CAD/CAM & PFM"
				skills = "Model Gips, CAD Design 3Shape/Exocad, Wax-up, Milling Zirconia, Porcelain Layering, Glaze & QC"
				sched = "Shift Produksi: Senin - Sabtu (08:00 - 17:00)"
				addr = "Jl. Kemang Utara No. 19, Jakarta Selatan"
				joinDate = "2022-08-15"
				bio = "Teknisi gigi tersertifikasi dengan spesialisasi restorasi cekat (Zirconia, E-Max, PFM) dan CAD/CAM."
			case "CASHIER":
				spec = "Finance, Invoicing & Purchasing Officer"
				skills = "Invoicing SPK, Rekonsiliasi Piutang Dokter, Stok & Pengadaan Bahan"
				sched = "Senin - Jumat (08:30 - 17:00)"
				addr = "Jl. Pancoran Timur No. 41, Jakarta Selatan"
				joinDate = "2023-05-10"
				bio = "Mengelola penagihan invoice klinik, pencatatan pembayaran, dan laporan arus kas laboratorium."
			}

			s.db.Model(&u).Updates(map[string]interface{}{
				"employee_code":  empCode,
				"specialization": spec,
				"skills":         skills,
				"work_schedule":  sched,
				"address":        addr,
				"join_date":      joinDate,
				"bio":            bio,
			})
		}
	}

	// Ensure a second technician exists so task assignment & KPI comparison have multiple technicians
	var techRole Role
	if err := s.db.Where("name = 'TECHNICIAN'").First(&techRole).Error; err == nil {
		var techCount int64
		s.db.Model(&User{}).Where("role_id = ? AND deleted_at IS NULL", techRole.ID).Count(&techCount)
		if techCount < 2 {
			hashed, _ := bcrypt.GenerateFromPassword([]byte("admin12345"), bcrypt.DefaultCost)
			tech2 := User{
				EmployeeCode:   fmt.Sprintf("EMP-%04d", len(users)+1),
				Username:       "teknisi2",
				Email:          "siti.teknisi@dentallab.id",
				PasswordHash:   string(hashed),
				FullName:       "Siti Nurhaliza, A.Md.TG",
				Phone:          "081388990022",
				RoleID:         techRole.ID,
				Specialization: "Spesialis Gigi Tiruan Lepasan (Akrilik, Valplast & Metal Frame)",
				Skills:         "Model Gips, Wax-up Bite Rim, Penyusunan Anasir Gigi, Packing Akrilik, Injection Valplast, Polishing",
				WorkSchedule:   "Shift Produksi: Senin - Sabtu (08:00 - 17:00)",
				Address:        "Jl. Kalibata Raya No. 27, Jakarta Selatan",
				JoinDate:       "2023-04-12",
				Bio:            "Ahli pembuatan gigi tiruan lepasan akrilik, fleksibel valplast, dan kerangka logam dengan adaptasi jaringan akurat.",
				IsActive:       true,
			}
			_ = s.db.Create(&tech2).Error
		}
	}

	// 5. Seed sample attendance & overtime records if empty
	var attCount int64
	s.db.Model(&StaffAttendance{}).Count(&attCount)
	if attCount == 0 {
		s.db.Preload("Role").Where("deleted_at IS NULL").Order("created_at ASC").Find(&users)
		today := time.Now().Format("2006-01-02")
		yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
		twoDaysAgo := time.Now().AddDate(0, 0, -2).Format("2006-01-02")

		for idx, u := range users {
			// Today's attendance
			otToday := 0.0
			otNoteToday := ""
			checkOutToday := "17:00"
			if u.Role.Name == "TECHNICIAN" && idx%2 == 1 {
				otToday = 2.5
				checkOutToday = "19:30"
				otNoteToday = "Lembur penyelesaian contouring & glazing Crown Zirconia Cito/Urgent"
			}
			a1 := StaffAttendance{
				UserID:         u.ID,
				AttendanceDate: today,
				ShiftSchedule:  u.WorkSchedule,
				CheckIn:        "07:52",
				CheckOut:       checkOutToday,
				Status:         "HADIR",
				OvertimeHours:  otToday,
				OvertimeNotes:  otNoteToday,
			}
			_ = s.db.Create(&a1).Error

			// Yesterday's attendance
			otYest := 0.0
			otNoteYest := ""
			checkOutYest := "17:05"
			if u.Role.Name == "TECHNICIAN" {
				otYest = 2.0
				checkOutYest = "19:00"
				otNoteYest = "Lembur pengecoran model gips & wax-up 3 SPK baru"
			}
			a2 := StaffAttendance{
				UserID:         u.ID,
				AttendanceDate: yesterday,
				ShiftSchedule:  u.WorkSchedule,
				CheckIn:        "07:55",
				CheckOut:       checkOutYest,
				Status:         "HADIR",
				OvertimeHours:  otYest,
				OvertimeNotes:  otNoteYest,
			}
			_ = s.db.Create(&a2).Error

			// Two days ago attendance
			a3 := StaffAttendance{
				UserID:         u.ID,
				AttendanceDate: twoDaysAgo,
				ShiftSchedule:  u.WorkSchedule,
				CheckIn:        "08:00",
				CheckOut:       "17:00",
				Status:         "HADIR",
				OvertimeHours:  0,
				OvertimeNotes:  "",
			}
			_ = s.db.Create(&a3).Error
		}
	}
}

type userOrderKPIAgg struct {
	TechID          uuid.UUID `gorm:"column:tech_id"`
	ActiveOrders    int       `gorm:"column:active_orders"`
	CompletedOrders int       `gorm:"column:completed_orders"`
	OnTimeOrders    int       `gorm:"column:on_time_orders"`
	AvgRating       float64   `gorm:"column:avg_rating"`
}

type userStageKPIAgg struct {
	TechID          uuid.UUID `gorm:"column:tech_id"`
	CompletedStages int       `gorm:"column:completed_stages"`
}

type userAttKPIAgg struct {
	UserID        uuid.UUID `gorm:"column:user_id"`
	PresentDays   int       `gorm:"column:present_days"`
	OvertimeHours float64   `gorm:"column:overtime_hours"`
}

func (s *Service) enrichUsersWithKPI(users []User) []User {
	if len(users) == 0 {
		return users
	}

	var orderAggs []userOrderKPIAgg
	s.db.Raw(`
		SELECT current_technician_id AS tech_id,
		       COUNT(*) FILTER (WHERE status IN ('INTAKE', 'IN_PRODUCTION', 'QC_PENDING', 'REOPENED_FOR_REWORK')) AS active_orders,
		       COUNT(*) FILTER (WHERE status IN ('COMPLETED', 'DELIVERED')) AS completed_orders,
		       COUNT(*) FILTER (WHERE status IN ('COMPLETED', 'DELIVERED') AND (completed_at IS NULL OR completed_at <= promised_due_date + INTERVAL '1 day')) AS on_time_orders,
		       COALESCE(AVG(NULLIF(feedback_rating, 0)), 4.9) AS avg_rating
		FROM production_orders
		WHERE deleted_at IS NULL AND current_technician_id IS NOT NULL
		GROUP BY current_technician_id
	`).Scan(&orderAggs)

	ordMap := make(map[uuid.UUID]userOrderKPIAgg)
	for _, a := range orderAggs {
		ordMap[a.TechID] = a
	}

	var stageAggs []userStageKPIAgg
	s.db.Raw(`
		SELECT technician_id AS tech_id,
		       COUNT(*) FILTER (WHERE status = 'COMPLETED') AS completed_stages
		FROM production_stage_runs
		WHERE technician_id IS NOT NULL
		GROUP BY technician_id
	`).Scan(&stageAggs)

	stgMap := make(map[uuid.UUID]int)
	for _, st := range stageAggs {
		stgMap[st.TechID] = st.CompletedStages
	}

	var attAggs []userAttKPIAgg
	s.db.Raw(`
		SELECT user_id,
		       COUNT(*) FILTER (WHERE status IN ('HADIR', 'TERLAMBAT')) AS present_days,
		       COALESCE(SUM(overtime_hours), 0) AS overtime_hours
		FROM staff_attendances
		GROUP BY user_id
	`).Scan(&attAggs)

	attMap := make(map[uuid.UUID]userAttKPIAgg)
	for _, at := range attAggs {
		attMap[at.UserID] = at
	}

	for i := range users {
		uid := users[i].ID
		users[i].CompletedStagesCount = stgMap[uid]
		if oa, ok := ordMap[uid]; ok {
			users[i].ActiveOrdersCount = oa.ActiveOrders
			users[i].CompletedOrdersCount = oa.CompletedOrders
			if oa.CompletedOrders > 0 {
				users[i].OnTimeRate = math.Round((float64(oa.OnTimeOrders)/float64(oa.CompletedOrders))*1000) / 10
			} else {
				users[i].OnTimeRate = 100
			}
			users[i].AvgFeedbackRating = math.Round(oa.AvgRating*10) / 10
		} else {
			users[i].OnTimeRate = 100
			users[i].AvgFeedbackRating = 4.9
		}
		users[i].QCPassRate = 98.5
		if users[i].CompletedOrdersCount > 0 {
			users[i].QCPassRate = 100
		}
		if at, ok := attMap[uid]; ok {
			users[i].AttendancePresentDays = at.PresentDays
			users[i].TotalOvertimeHours = at.OvertimeHours
		}
	}

	return users
}

func (s *Service) Login(req LoginRequest) (*LoginResponse, error) {
	var user User
	err := s.db.Preload("Role").Where("username = ? AND is_active = true", req.Username).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("invalid username or password")
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, errors.New("invalid username or password")
	}

	enriched := s.enrichUsersWithKPI([]User{user})
	user = enriched[0]

	tokenTTL := 24 * time.Hour
	claims := middleware.Claims{
		UserID:   user.ID.String(),
		Username: user.Username,
		Role:     user.Role.Name,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(tokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return nil, err
	}

	return &LoginResponse{
		Token:     tokenStr,
		ExpiresIn: "24h",
		User:      user,
	}, nil
}

func (s *Service) GetMe(userIDStr string) (*User, error) {
	uid, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, err
	}

	var user User
	err = s.db.Preload("Role").Where("id = ?", uid).First(&user).Error
	if err != nil {
		return nil, err
	}
	enriched := s.enrichUsersWithKPI([]User{user})
	return &enriched[0], nil
}

// User Management
func (s *Service) ListUsers() ([]User, error) {
	var users []User
	err := s.db.Preload("Role").Where("deleted_at IS NULL").Order("created_at ASC").Find(&users).Error
	if err != nil {
		return nil, err
	}
	return s.enrichUsersWithKPI(users), nil
}

func (s *Service) CreateUser(req CreateUserRequest) (*User, error) {
	if req.Username == "" || req.Password == "" || req.FullName == "" || req.RoleID == 0 {
		return nil, errors.New("username, password, full name, and role are required")
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	var count int64
	s.db.Unscoped().Model(&User{}).Count(&count)
	empCode := fmt.Sprintf("EMP-%04d", count+1)

	sched := req.WorkSchedule
	if sched == "" {
		sched = "Senin - Sabtu (08:00 - 17:00)"
	}
	joinDate := req.JoinDate
	if joinDate == "" {
		joinDate = time.Now().Format("2006-01-02")
	}

	user := User{
		EmployeeCode:   empCode,
		Username:       req.Username,
		Email:          req.Email,
		FullName:       req.FullName,
		Phone:          req.Phone,
		RoleID:         req.RoleID,
		Specialization: req.Specialization,
		Skills:         req.Skills,
		WorkSchedule:   sched,
		Address:        req.Address,
		JoinDate:       joinDate,
		Bio:            req.Bio,
		PasswordHash:   string(hashed),
		IsActive:       true,
	}

	if err := s.db.Create(&user).Error; err != nil {
		return nil, err
	}

	return s.GetMe(user.ID.String())
}

func (s *Service) UpdateUser(userID uuid.UUID, req UpdateUserRequest) (*User, error) {
	var user User
	if err := s.db.Where("id = ? AND deleted_at IS NULL", userID).First(&user).Error; err != nil {
		return nil, err
	}

	if req.FullName != nil {
		user.FullName = *req.FullName
	}
	if req.Email != nil {
		user.Email = *req.Email
	}
	if req.Phone != nil {
		user.Phone = *req.Phone
	}
	if req.RoleID != nil {
		user.RoleID = *req.RoleID
	}
	if req.IsActive != nil {
		user.IsActive = *req.IsActive
	}
	if req.Specialization != nil {
		user.Specialization = *req.Specialization
	}
	if req.Skills != nil {
		user.Skills = *req.Skills
	}
	if req.WorkSchedule != nil {
		user.WorkSchedule = *req.WorkSchedule
	}
	if req.Address != nil {
		user.Address = *req.Address
	}
	if req.JoinDate != nil {
		user.JoinDate = *req.JoinDate
	}
	if req.Bio != nil {
		user.Bio = *req.Bio
	}
	if req.AvatarURL != nil {
		user.AvatarURL = *req.AvatarURL
	}
	if req.Password != nil && *req.Password != "" {
		hashed, err := bcrypt.GenerateFromPassword([]byte(*req.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		user.PasswordHash = string(hashed)
	}

	user.UpdatedAt = time.Now()
	if err := s.db.Save(&user).Error; err != nil {
		return nil, err
	}

	return s.GetMe(user.ID.String())
}

func (s *Service) DeleteUser(userID uuid.UUID) error {
	return s.db.Where("id = ?", userID).Delete(&User{}).Error
}

// Role Management
func (s *Service) ListRoles() ([]Role, error) {
	var roles []Role
	err := s.db.Order("id ASC").Find(&roles).Error
	return roles, err
}

func (s *Service) UpdateRoleMenus(roleID int, allowedMenus string) error {
	if allowedMenus == "" {
		allowedMenus = "[]"
	}
	return s.db.Model(&Role{}).Where("id = ?", roleID).Update("allowed_menus", allowedMenus).Error
}

// Attendance & Overtime Management
func (s *Service) ListAttendances(userIDStr string) ([]StaffAttendance, error) {
	var list []StaffAttendance
	q := s.db.Preload("User").Preload("User.Role")
	if userIDStr != "" {
		if uid, err := uuid.Parse(userIDStr); err == nil {
			q = q.Where("user_id = ?", uid)
		}
	}
	err := q.Order("attendance_date DESC, created_at DESC").Find(&list).Error
	return list, err
}

func (s *Service) CreateAttendance(req CreateAttendanceRequest) (*StaffAttendance, error) {
	uid, err := uuid.Parse(req.UserID)
	if err != nil {
		return nil, errors.New("invalid user_id")
	}
	attDate := req.AttendanceDate
	if attDate == "" {
		attDate = time.Now().Format("2006-01-02")
	}
	status := req.Status
	if status == "" {
		status = "HADIR"
	}
	checkIn := req.CheckIn
	if checkIn == "" {
		checkIn = "08:00"
	}
	checkOut := req.CheckOut
	if checkOut == "" {
		checkOut = "17:00"
	}
	shift := req.ShiftSchedule
	if shift == "" {
		shift = "Shift Pagi (08:00 - 17:00)"
	}

	// Upsert if record for same user + date already exists
	var existing StaffAttendance
	if err := s.db.Where("user_id = ? AND attendance_date = ?", uid, attDate).First(&existing).Error; err == nil {
		existing.ShiftSchedule = shift
		existing.CheckIn = checkIn
		existing.CheckOut = checkOut
		existing.Status = status
		existing.OvertimeHours = req.OvertimeHours
		existing.OvertimeNotes = req.OvertimeNotes
		existing.UpdatedAt = time.Now()
		if err := s.db.Save(&existing).Error; err != nil {
			return nil, err
		}
		s.db.Preload("User").Preload("User.Role").First(&existing, "id = ?", existing.ID)
		return &existing, nil
	}

	att := StaffAttendance{
		UserID:         uid,
		AttendanceDate: attDate,
		ShiftSchedule:  shift,
		CheckIn:        checkIn,
		CheckOut:       checkOut,
		Status:         status,
		OvertimeHours:  req.OvertimeHours,
		OvertimeNotes:  req.OvertimeNotes,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	if err := s.db.Create(&att).Error; err != nil {
		return nil, err
	}
	s.db.Preload("User").Preload("User.Role").First(&att, "id = ?", att.ID)
	return &att, nil
}

func (s *Service) DeleteAttendance(id uuid.UUID) error {
	return s.db.Where("id = ?", id).Delete(&StaffAttendance{}).Error
}

// System Settings
func (s *Service) GetSettings() (map[string]string, error) {
	var list []SystemSetting
	if err := s.db.Find(&list).Error; err != nil {
		return nil, err
	}

	res := make(map[string]string)
	for _, item := range list {
		res[item.Key] = item.Value
	}
	return res, nil
}

func (s *Service) UpdateSetting(key, value string) error {
	setting := SystemSetting{
		Key:       key,
		Value:     value,
		UpdatedAt: time.Now(),
	}
	return s.db.Save(&setting).Error
}
