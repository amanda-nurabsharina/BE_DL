package customers

import (
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	s := &Service{db: db}
	s.ensureSchemaAndSeed()
	return s
}

func (s *Service) ensureSchemaAndSeed() {
	// 1. Ensure schema columns & patients table exist
	s.db.Exec(`ALTER TABLE customers ADD COLUMN IF NOT EXISTS favorite_shade VARCHAR(50) DEFAULT ''`)
	s.db.Exec(`ALTER TABLE customers ADD COLUMN IF NOT EXISTS favorite_denture_type VARCHAR(150) DEFAULT ''`)
	s.db.Exec(`ALTER TABLE customers ADD COLUMN IF NOT EXISTS loyalty_tier VARCHAR(30) DEFAULT 'REGULAR'`)
	s.db.Exec(`ALTER TABLE customers ADD COLUMN IF NOT EXISTS discount_percent NUMERIC(5,2) DEFAULT 0`)

	s.db.Exec(`CREATE TABLE IF NOT EXISTS patients (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		code VARCHAR(30) UNIQUE NOT NULL,
		customer_id UUID REFERENCES customers(id) ON DELETE SET NULL,
		name VARCHAR(150) NOT NULL,
		gender VARCHAR(10) DEFAULT 'L',
		age INT DEFAULT 0,
		phone VARCHAR(30) DEFAULT '',
		default_shade VARCHAR(50) DEFAULT 'VITA A2',
		last_teeth_numbers VARCHAR(100) DEFAULT '',
		allergies_or_notes TEXT DEFAULT '',
		is_active BOOLEAN DEFAULT TRUE,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
		deleted_at TIMESTAMP WITH TIME ZONE
	)`)

	s.db.Exec(`ALTER TABLE production_orders ADD COLUMN IF NOT EXISTS patient_id UUID REFERENCES patients(id) ON DELETE SET NULL`)
	s.db.Exec(`ALTER TABLE production_orders ADD COLUMN IF NOT EXISTS discount_percent NUMERIC(5,2) DEFAULT 0`)
	s.db.Exec(`ALTER TABLE production_orders ADD COLUMN IF NOT EXISTS discount_amount NUMERIC(14,2) DEFAULT 0`)
	s.db.Exec(`ALTER TABLE production_orders ADD COLUMN IF NOT EXISTS feedback_rating INT DEFAULT 0`)
	s.db.Exec(`ALTER TABLE production_orders ADD COLUMN IF NOT EXISTS feedback_notes TEXT DEFAULT ''`)

	// 2. Ensure sample doctors exist and have rich CRM preferences & loyalty tiers
	var custs []Customer
	s.db.Where("deleted_at IS NULL").Order("code ASC").Find(&custs)

	if len(custs) == 0 {
		seedDocs := []Customer{
			{
				Code:                "DOC-0001",
				Name:                "drg. Hendra Wijaya, Sp.Pros",
				ClinicName:          "Klinik Gigi Senyum Sehat",
				Phone:               "081234567801",
				Email:               "hendra.pros@senyumsehat.id",
				Address:             "Jl. Sudirman No. 45, Jakarta Selatan",
				Notes:               "Preferensi oklusi ringan, kontak proksimal rapat, margin servikal halus",
				FavoriteShade:       "VITA A2",
				FavoriteDentureType: "Zirconia Crown (Full Contour)",
				LoyaltyTier:         "VIP",
				DiscountPercent:     15,
				Preferences:         `{"shade":"VITA A2","contact":"tight"}`,
				IsActive:            true,
			},
			{
				Code:                "DOC-0002",
				Name:                "drg. Amanda Putri, Sp.KG",
				ClinicName:          "RS Premier Dental Center",
				Phone:               "081298765402",
				Email:               "amanda.kg@premierdental.id",
				Address:             "Jl. Boulevard Raya Blok B2, Jakarta Utara",
				Notes:               "Suka anatomi labial natural dengan tekstur mikro untuk gigi anterior",
				FavoriteShade:       "VITA A1",
				FavoriteDentureType: "All-Porcelain E-Max Veneer / Crown",
				LoyaltyTier:         "GOLD",
				DiscountPercent:     10,
				Preferences:         `{"shade":"VITA A1","anatomy":"natural"}`,
				IsActive:            true,
			},
			{
				Code:                "DOC-0003",
				Name:                "drg. Budi Santoso",
				ClinicName:          "Klinik Pratama Dental Care",
				Phone:               "081355667788",
				Email:               "drg.budi@dentalcare.id",
				Address:             "Jl. Tebet Raya No. 18, Jakarta Selatan",
				Notes:               "Gigi tiruan lepasan akrilik & valplast harus tipis dan nyaman di palatum",
				FavoriteShade:       "VITA A3",
				FavoriteDentureType: "Valplast Flexible Denture",
				LoyaltyTier:         "SILVER",
				DiscountPercent:     5,
				Preferences:         `{"shade":"VITA A3"}`,
				IsActive:            true,
			},
			{
				Code:                "DOC-0004",
				Name:                "drg. Ratna Sari, Sp.Pros",
				ClinicName:          "Menteng Aesthetic Dental Clinic",
				Phone:               "081188990011",
				Email:               "ratna@mentengdental.id",
				Address:             "Jl. HOS Cokroaminoto No. 88, Menteng, Jakarta Pusat",
				Notes:               "Sering order PFM & Metal Frame presisi tinggi, kirim kurir pagi sebelum jam 10",
				FavoriteShade:       "VITA A2",
				FavoriteDentureType: "PFM (Porcelain Fused to Metal)",
				LoyaltyTier:         "GOLD",
				DiscountPercent:     10,
				Preferences:         `{"shade":"VITA A2"}`,
				IsActive:            true,
			},
		}
		for _, d := range seedDocs {
			s.db.Create(&d)
		}
		s.db.Where("deleted_at IS NULL").Order("code ASC").Find(&custs)
	} else {
		// Enrich existing doctors if their preferences/loyalty are still empty
		for i, c := range custs {
			if c.FavoriteShade == "" && c.FavoriteDentureType == "" {
				switch i % 3 {
				case 0:
					s.db.Model(&c).Updates(map[string]interface{}{
						"favorite_shade":        "VITA A2",
						"favorite_denture_type": "Zirconia Crown (Full Contour)",
						"loyalty_tier":          "VIP",
						"discount_percent":      15,
						"notes":                 "Preferensi oklusi ringan, kontak proksimal rapat, margin servikal halus",
					})
				case 1:
					s.db.Model(&c).Updates(map[string]interface{}{
						"favorite_shade":        "VITA A1",
						"favorite_denture_type": "All-Porcelain E-Max Veneer / Crown",
						"loyalty_tier":          "GOLD",
						"discount_percent":      10,
						"notes":                 "Suka anatomi labial natural dengan tekstur mikro untuk gigi anterior",
					})
				case 2:
					s.db.Model(&c).Updates(map[string]interface{}{
						"favorite_shade":        "VITA A3",
						"favorite_denture_type": "Valplast Flexible Denture",
						"loyalty_tier":          "SILVER",
						"discount_percent":      5,
						"notes":                 "Gigi tiruan lepasan akrilik & valplast harus tipis dan nyaman di palatum",
					})
				}
			}
		}
		// Ensure at least 4 doctors so there is a doctor needing follow-up reminder
		if len(custs) < 4 {
			extra := Customer{
				Code:                fmt.Sprintf("DOC-%04d", len(custs)+1),
				Name:                "drg. Ratna Sari, Sp.Pros",
				ClinicName:          "Menteng Aesthetic Dental Clinic",
				Phone:               "081188990011",
				Email:               "ratna@mentengdental.id",
				Address:             "Jl. HOS Cokroaminoto No. 88, Menteng, Jakarta Pusat",
				Notes:               "Sering order PFM & Metal Frame presisi tinggi, minta konfirmasi WA sebelum kirim",
				FavoriteShade:       "VITA A2",
				FavoriteDentureType: "PFM (Porcelain Fused to Metal)",
				LoyaltyTier:         "GOLD",
				DiscountPercent:     10,
				Preferences:         `{"shade":"VITA A2"}`,
				IsActive:            true,
				CreatedAt:           time.Now().AddDate(0, -2, 0),
				UpdatedAt:           time.Now().AddDate(0, -1, 0),
			}
			s.db.Create(&extra)
			s.db.Where("deleted_at IS NULL").Order("code ASC").Find(&custs)
		}
	}

	// 3. Seed Master Patients if empty
	var patCount int64
	s.db.Model(&Patient{}).Count(&patCount)
	if patCount == 0 && len(custs) > 0 {
		c0 := custs[0].ID
		c1 := custs[0].ID
		if len(custs) > 1 {
			c1 = custs[1].ID
		}
		c2 := custs[0].ID
		if len(custs) > 2 {
			c2 = custs[2].ID
		}
		c3 := custs[0].ID
		if len(custs) > 3 {
			c3 = custs[3].ID
		}

		seedPatients := []Patient{
			{
				Code:             "PAS-0001",
				CustomerID:       &c0,
				Name:             "Bpk. Hendra Gunawan",
				Gender:           "L",
				Age:              46,
				Phone:            "081211223344",
				DefaultShade:     "VITA A2",
				LastTeethNumbers: `["11","21"]`,
				AllergiesOrNotes: "Gigi anterior atas, tidak ada alergi bahan logam/resin",
				IsActive:         true,
			},
			{
				Code:             "PAS-0002",
				CustomerID:       &c1,
				Name:             "Ibu Siti Rahmawati",
				Gender:           "P",
				Age:              38,
				Phone:            "081322334455",
				DefaultShade:     "VITA A3",
				LastTeethNumbers: `["36","37"]`,
				AllergiesOrNotes: "Riwayat bruxism ringan, butuh oklusi kuat (PFM/Zirconia)",
				IsActive:         true,
			},
			{
				Code:             "PAS-0003",
				CustomerID:       &c2,
				Name:             "Bpk. Ahmad Fauzi",
				Gender:           "L",
				Age:              58,
				Phone:            "081533445566",
				DefaultShade:     "VITA A3",
				LastTeethNumbers: `["14","15","16","24","25"]`,
				AllergiesOrNotes: "Alergi logam nikel — direkomendasikan Valplast Flexible atau Akrilik",
				IsActive:         true,
			},
			{
				Code:             "PAS-0004",
				CustomerID:       &c0,
				Name:             "Sdr. Rina Kartika",
				Gender:           "P",
				Age:              27,
				Phone:            "081744556677",
				DefaultShade:     "VITA A1",
				LastTeethNumbers: `["12","11","21","22"]`,
				AllergiesOrNotes: "Estetik tinggi (Veneer E-Max anterior rahang atas)",
				IsActive:         true,
			},
			{
				Code:             "PAS-0005",
				CustomerID:       &c1,
				Name:             "Ibu Dewi Lestari",
				Gender:           "P",
				Age:              52,
				Phone:            "081855667788",
				DefaultShade:     "VITA A2",
				LastTeethNumbers: `["46"]`,
				AllergiesOrNotes: "Crown molar rahang bawah kanan",
				IsActive:         true,
			},
			{
				Code:             "PAS-0006",
				CustomerID:       &c3,
				Name:             "Bpk. Bambang Sudibyo",
				Gender:           "L",
				Age:              63,
				Phone:            "081966778899",
				DefaultShade:     "VITA A3.5",
				LastTeethNumbers: `["11","12","13","21","22","23"]`,
				AllergiesOrNotes: "Full/Partial Denture rahang atas, linggir alveolar agak datar",
				IsActive:         true,
			},
		}

		for _, p := range seedPatients {
			s.db.Create(&p)
		}
	}

	// 4. Link existing production_orders to matching patients by patient_name & add sample feedback on completed orders
	s.db.Exec(`
		UPDATE production_orders po
		SET patient_id = p.id
		FROM patients p
		WHERE po.patient_id IS NULL AND LOWER(po.patient_name) = LOWER(p.name)
	`)
	s.db.Exec(`
		UPDATE production_orders
		SET feedback_rating = 5,
		    feedback_notes = 'Marginal fit sangat presisi, warna VITA A1 sesuai ekspektasi pasien, tanpa perlu banyak grinding saat insersi.'
		WHERE status IN ('COMPLETED', 'DELIVERED') AND feedback_rating = 0
	`)
}

type customerOrderAgg struct {
	CustomerID    uuid.UUID  `gorm:"column:customer_id"`
	TotalOrders   int        `gorm:"column:total_orders"`
	TotalSpent    float64    `gorm:"column:total_spent"`
	LastOrderDate *time.Time `gorm:"column:last_order_date"`
}

type customerPatientAgg struct {
	CustomerID    uuid.UUID `gorm:"column:customer_id"`
	PatientsCount int       `gorm:"column:patients_count"`
}

func (s *Service) enrichCustomers(list []Customer) []Customer {
	if len(list) == 0 {
		return list
	}

	var orderAggs []customerOrderAgg
	s.db.Raw(`
		SELECT customer_id,
		       COUNT(*) AS total_orders,
		       COALESCE(SUM(total_price), 0) AS total_spent,
		       MAX(received_date) AS last_order_date
		FROM production_orders
		WHERE deleted_at IS NULL
		GROUP BY customer_id
	`).Scan(&orderAggs)

	orderMap := make(map[uuid.UUID]customerOrderAgg)
	for _, a := range orderAggs {
		orderMap[a.CustomerID] = a
	}

	var patAggs []customerPatientAgg
	s.db.Raw(`
		SELECT customer_id, COUNT(*) AS patients_count
		FROM patients
		WHERE deleted_at IS NULL AND customer_id IS NOT NULL
		GROUP BY customer_id
	`).Scan(&patAggs)

	patMap := make(map[uuid.UUID]int)
	for _, p := range patAggs {
		patMap[p.CustomerID] = p.PatientsCount
	}

	now := time.Now()
	for i := range list {
		if agg, ok := orderMap[list[i].ID]; ok {
			list[i].TotalOrders = agg.TotalOrders
			list[i].TotalSpent = agg.TotalSpent
			list[i].LastOrderDate = agg.LastOrderDate
			if agg.LastOrderDate != nil {
				days := int(math.Floor(now.Sub(*agg.LastOrderDate).Hours() / 24))
				if days < 0 {
					days = 0
				}
				list[i].DaysSinceLastOrder = days
			} else {
				list[i].DaysSinceLastOrder = 999
			}
		} else {
			list[i].DaysSinceLastOrder = 999
		}
		list[i].PatientsCount = patMap[list[i].ID]
	}
	return list
}

func (s *Service) List(search string) ([]Customer, error) {
	var list []Customer
	query := s.db.Where("deleted_at IS NULL")
	if search != "" {
		likeTerm := "%" + search + "%"
		query = query.Where("name ILIKE ? OR clinic_name ILIKE ? OR phone ILIKE ? OR code ILIKE ?", likeTerm, likeTerm, likeTerm, likeTerm)
	}
	err := query.Order("code ASC").Find(&list).Error
	if err != nil {
		return nil, err
	}
	return s.enrichCustomers(list), nil
}

func (s *Service) GetByID(id uuid.UUID) (*Customer, error) {
	var c Customer
	err := s.db.Where("id = ? AND deleted_at IS NULL", id).First(&c).Error
	if err != nil {
		return nil, err
	}
	enriched := s.enrichCustomers([]Customer{c})
	return &enriched[0], nil
}

func (s *Service) Create(req CreateCustomerRequest) (*Customer, error) {
	var count int64
	s.db.Unscoped().Model(&Customer{}).Count(&count)
	code := fmt.Sprintf("DOC-%04d", count+1)

	pref := req.Preferences
	if pref == "" {
		pref = "{}"
	}
	tier := req.LoyaltyTier
	if tier == "" {
		tier = "REGULAR"
	}

	c := Customer{
		Code:                code,
		Name:                req.Name,
		ClinicName:          req.ClinicName,
		Phone:               req.Phone,
		Email:               req.Email,
		Address:             req.Address,
		Notes:               req.Notes,
		FavoriteShade:       req.FavoriteShade,
		FavoriteDentureType: req.FavoriteDentureType,
		LoyaltyTier:         tier,
		DiscountPercent:     req.DiscountPercent,
		Preferences:         pref,
		IsActive:            true,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}

	if err := s.db.Create(&c).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *Service) Update(id uuid.UUID, req UpdateCustomerRequest) (*Customer, error) {
	var c Customer
	if err := s.db.Where("id = ? AND deleted_at IS NULL", id).First(&c).Error; err != nil {
		return nil, err
	}

	if req.Name != nil {
		c.Name = *req.Name
	}
	if req.ClinicName != nil {
		c.ClinicName = *req.ClinicName
	}
	if req.Phone != nil {
		c.Phone = *req.Phone
	}
	if req.Email != nil {
		c.Email = *req.Email
	}
	if req.Address != nil {
		c.Address = *req.Address
	}
	if req.Notes != nil {
		c.Notes = *req.Notes
	}
	if req.FavoriteShade != nil {
		c.FavoriteShade = *req.FavoriteShade
	}
	if req.FavoriteDentureType != nil {
		c.FavoriteDentureType = *req.FavoriteDentureType
	}
	if req.LoyaltyTier != nil {
		c.LoyaltyTier = *req.LoyaltyTier
	}
	if req.DiscountPercent != nil {
		c.DiscountPercent = *req.DiscountPercent
	}
	if req.Preferences != nil {
		c.Preferences = *req.Preferences
	}
	if req.IsActive != nil {
		c.IsActive = *req.IsActive
	}
	c.UpdatedAt = time.Now()

	if err := s.db.Save(&c).Error; err != nil {
		return nil, err
	}
	return s.GetByID(id)
}

func (s *Service) Delete(id uuid.UUID) error {
	return s.db.Where("id = ?", id).Delete(&Customer{}).Error
}

// --- Master Data Pasien ---

type patientOrderAgg struct {
	PatientName   string     `gorm:"column:patient_name"`
	TotalOrders   int        `gorm:"column:total_orders"`
	LastOrderDate *time.Time `gorm:"column:last_order_date"`
}

func (s *Service) ListPatients(search string, customerID string) ([]Patient, error) {
	var list []Patient
	query := s.db.Preload("Customer").Where("deleted_at IS NULL")
	if customerID != "" {
		if cid, err := uuid.Parse(customerID); err == nil {
			query = query.Where("customer_id = ?", cid)
		}
	}
	if search != "" {
		likeTerm := "%" + search + "%"
		query = query.Where("name ILIKE ? OR code ILIKE ? OR phone ILIKE ? OR allergies_or_notes ILIKE ?", likeTerm, likeTerm, likeTerm, likeTerm)
	}
	if err := query.Order("code ASC").Find(&list).Error; err != nil {
		return nil, err
	}

	var aggs []patientOrderAgg
	s.db.Raw(`
		SELECT LOWER(patient_name) AS patient_name,
		       COUNT(*) AS total_orders,
		       MAX(received_date) AS last_order_date
		FROM production_orders
		WHERE deleted_at IS NULL
		GROUP BY LOWER(patient_name)
	`).Scan(&aggs)

	aggMap := make(map[string]patientOrderAgg)
	for _, a := range aggs {
		aggMap[a.PatientName] = a
	}

	for i := range list {
		key := fmt.Sprintf("%s", list[i].Name)
		for k, v := range aggMap {
			if stringsEqualFold(k, key) {
				list[i].TotalOrders = v.TotalOrders
				list[i].LastOrderDate = v.LastOrderDate
				break
			}
		}
	}

	return list, nil
}

func stringsEqualFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca := a[i]
		cb := b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}

func (s *Service) CreatePatient(req CreatePatientRequest) (*Patient, error) {
	var count int64
	s.db.Unscoped().Model(&Patient{}).Count(&count)
	code := fmt.Sprintf("PAS-%04d", count+1)

	var custUUID *uuid.UUID
	if req.CustomerID != "" {
		if parsed, err := uuid.Parse(req.CustomerID); err == nil {
			custUUID = &parsed
		}
	}

	gender := req.Gender
	if gender == "" {
		gender = "L"
	}
	shade := req.DefaultShade
	if shade == "" {
		shade = "VITA A2"
	}

	p := Patient{
		Code:             code,
		CustomerID:       custUUID,
		Name:             req.Name,
		Gender:           gender,
		Age:              req.Age,
		Phone:            req.Phone,
		DefaultShade:     shade,
		LastTeethNumbers: req.LastTeethNumbers,
		AllergiesOrNotes: req.AllergiesOrNotes,
		IsActive:         true,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	if err := s.db.Create(&p).Error; err != nil {
		return nil, err
	}
	s.db.Preload("Customer").First(&p, "id = ?", p.ID)
	return &p, nil
}

func (s *Service) UpdatePatient(id uuid.UUID, req UpdatePatientRequest) (*Patient, error) {
	var p Patient
	if err := s.db.Where("id = ? AND deleted_at IS NULL", id).First(&p).Error; err != nil {
		return nil, err
	}

	if req.CustomerID != nil {
		if *req.CustomerID == "" {
			p.CustomerID = nil
		} else if parsed, err := uuid.Parse(*req.CustomerID); err == nil {
			p.CustomerID = &parsed
		}
	}
	if req.Name != nil {
		p.Name = *req.Name
	}
	if req.Gender != nil {
		p.Gender = *req.Gender
	}
	if req.Age != nil {
		p.Age = *req.Age
	}
	if req.Phone != nil {
		p.Phone = *req.Phone
	}
	if req.DefaultShade != nil {
		p.DefaultShade = *req.DefaultShade
	}
	if req.LastTeethNumbers != nil {
		p.LastTeethNumbers = *req.LastTeethNumbers
	}
	if req.AllergiesOrNotes != nil {
		p.AllergiesOrNotes = *req.AllergiesOrNotes
	}
	if req.IsActive != nil {
		p.IsActive = *req.IsActive
	}
	p.UpdatedAt = time.Now()

	if err := s.db.Save(&p).Error; err != nil {
		return nil, err
	}
	s.db.Preload("Customer").First(&p, "id = ?", p.ID)
	return &p, nil
}

func (s *Service) DeletePatient(id uuid.UUID) error {
	return s.db.Where("id = ?", id).Delete(&Patient{}).Error
}
