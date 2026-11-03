package inventory

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	svc := &Service{db: db}
	svc.ensureSchemaAndSeed()
	return svc
}

func parseFlexibleDate(dateStr string) *time.Time {
	dateStr = strings.TrimSpace(dateStr)
	if dateStr == "" {
		return nil
	}
	layouts := []string{
		"2006-01-02",
		time.RFC3339,
		"2006-01-02T15:04:05",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, dateStr); err == nil {
			return &t
		}
	}
	return nil
}

func (s *Service) ensureSchemaAndSeed() {
	// 1. Add columns to items table if not exists
	_ = s.db.Exec(`ALTER TABLE items ADD COLUMN IF NOT EXISTS supplier_id UUID REFERENCES suppliers(id) ON DELETE SET NULL`).Error
	_ = s.db.Exec(`ALTER TABLE items ADD COLUMN IF NOT EXISTS lot_number VARCHAR(100)`).Error
	_ = s.db.Exec(`ALTER TABLE items ADD COLUMN IF NOT EXISTS expiry_date DATE`).Error

	// 2. Seed Suppliers if empty
	var supCount int64
	s.db.Model(&Supplier{}).Count(&supCount)
	if supCount == 0 {
		suppliers := []Supplier{
			{Code: "SUP-001", Name: "PT Dental Medika Utama", ContactPerson: "Bpk. Arif Rahman", Phone: "081122334455", Email: "sales@dentalmedika.co.id", Address: "Jl. Kesehatan Raya No. 18, Jakarta", IsActive: true},
			{Code: "SUP-002", Name: "CV Sentral Gigi Sejahtera", ContactPerson: "Ibu Diana Putri", Phone: "081299887766", Email: "order@sentralgigi.id", Address: "Jl. Diponegoro No. 45, Surabaya", IsActive: true},
			{Code: "SUP-003", Name: "PT Ivoclar Indo Dental", ContactPerson: "Bpk. Kevin Tan", Phone: "081355667788", Email: "info@ivoclar-indo.com", Address: "Kawasan Industri Pulogadung, Jakarta", IsActive: true},
			{Code: "SUP-004", Name: "PT Zirconia Lab Supply", ContactPerson: "drg. Hendri Wijaya", Phone: "081566778899", Email: "support@zirconialab.id", Address: "Jl. Pasteur No. 102, Bandung", IsActive: true},
		}
		for i := range suppliers {
			_ = s.db.Create(&suppliers[i]).Error
		}
	}

	// 3. Seed Master Items, Batches & Initial Movements if empty
	var itemCount int64
	s.db.Model(&Item{}).Count(&itemCount)
	if itemCount == 0 {
		var units []UnitOfMeasure
		s.db.Find(&units)
		unitMap := make(map[string]int)
		for _, u := range units {
			unitMap[u.Code] = u.ID
		}
		getUnit := func(code string) int {
			if id, ok := unitMap[code]; ok {
				return id
			}
			if len(units) > 0 {
				return units[0].ID
			}
			return 1
		}

		var sups []Supplier
		s.db.Order("code ASC").Find(&sups)
		getSup := func(idx int) *uuid.UUID {
			if idx < len(sups) {
				id := sups[idx].ID
				return &id
			}
			return nil
		}

		now := time.Now()
		strPtr := func(v string) *string { return &v }
		dateOffset := func(days int) *time.Time {
			t := now.AddDate(0, 0, days)
			return &t
		}

		seedItems := []struct {
			Item       Item
			UsedAmount float64
			UsageCount int
			UsageNote  string
		}{
			{
				Item: Item{
					Code:          "ITM-0001",
					Name:          "Akrilik Heat Cure Pink (Polymer & Monomer)",
					Category:      "RAW_MATERIAL",
					UnitID:        getUnit("box"),
					SupplierID:    getSup(0),
					MinStockLevel: 5,
					CurrentStock:  14,
					PurchasePrice: 385000,
					LotNumber:     strPtr("LOT-ACR-2611"),
					ExpiryDate:    dateOffset(240),
					Description:   "Bahan dasar plat gigi tiruan lepasan akrilik heat-cured warna pink natural",
					IsActive:      true,
				},
				UsedAmount: 6,
				UsageCount: 4,
				UsageNote:  "Pemakaian packing kuvet gigi tiruan lepasan akrilik",
			},
			{
				Item: Item{
					Code:          "ITM-0002",
					Name:          "Gips Tipe 4 Dental Stone (Biru Keras)",
					Category:      "RAW_MATERIAL",
					UnitID:        getUnit("kg"),
					SupplierID:    getSup(1),
					MinStockLevel: 10,
					CurrentStock:  28,
					PurchasePrice: 65000,
					LotNumber:     strPtr("LOT-GIP-2610"),
					ExpiryDate:    dateOffset(365),
					Description:   "Dental stone die presisi tinggi untuk pengecoran model kerja & master die",
					IsActive:      true,
				},
				UsedAmount: 17,
				UsageCount: 6,
				UsageNote:  "Pengecoran model kerja gips & preparasi die SPK harian",
			},
			{
				Item: Item{
					Code:          "ITM-0003",
					Name:          "Logam Alloy Co-Cr (Kobalt Krom) Non-Beryllium",
					Category:      "RAW_MATERIAL",
					UnitID:        getUnit("gr"),
					SupplierID:    getSup(2),
					MinStockLevel: 250,
					CurrentStock:  160, // Below min stock to trigger alert
					PurchasePrice: 4200,
					LotNumber:     strPtr("LOT-MET-2608"),
					ExpiryDate:    nil,
					Description:   "Ingot logam kobalt krom untuk casting koping PFM dan kerangka logam",
					IsActive:      true,
				},
				UsedAmount: 140,
				UsageCount: 5,
				UsageNote:  "Casting koping logam mahkota PFM",
			},
			{
				Item: Item{
					Code:          "ITM-0004",
					Name:          "Gigi Tiruan Resin Akrilik Set (Anterior & Posterior A2)",
					Category:      "PRODUCT",
					UnitID:        getUnit("set"),
					SupplierID:    getSup(0),
					MinStockLevel: 15,
					CurrentStock:  24,
					PurchasePrice: 95000,
					LotNumber:     strPtr("LOT-RES-2609"),
					ExpiryDate:    dateOffset(25), // Expiring soon (< 30 days)
					Description:   "Elemen gigi tiruan resin multi-layer shade VITA A2 untuk penyusunan gigi lepasan",
					IsActive:      true,
				},
				UsedAmount: 11,
				UsageCount: 5,
				UsageNote:  "Penyusunan elemen gigi tiruan lepasan pasien",
			},
			{
				Item: Item{
					Code:          "ITM-0005",
					Name:          "Disk Zirconia Multi-Layer CAD/CAM 98x16mm (VITA A2)",
					Category:      "RAW_MATERIAL",
					UnitID:        getUnit("pcs"),
					SupplierID:    getSup(3),
					MinStockLevel: 5,
					CurrentStock:  3, // Below min stock to trigger alert
					PurchasePrice: 1450000,
					LotNumber:     strPtr("LOT-ZIR-2611"),
					ExpiryDate:    dateOffset(540),
					Description:   "Blok disk Zirconia estetis translusen tinggi untuk mesin CNC milling",
					IsActive:      true,
				},
				UsedAmount: 4,
				UsageCount: 4,
				UsageNote:  "CAM Milling mahkota & bridge All-Ceramic Zirconia",
			},
			{
				Item: Item{
					Code:          "ITM-0006",
					Name:          "Bubuk Porselen Keramik Dentin & Enamel (Ivoclar/Ceramco)",
					Category:      "RAW_MATERIAL",
					UnitID:        getUnit("box"),
					SupplierID:    getSup(2),
					MinStockLevel: 3,
					CurrentStock:  8,
					PurchasePrice: 620000,
					LotNumber:     strPtr("LOT-PFM-2607"),
					ExpiryDate:    dateOffset(300),
					Description:   "Bubuk keramik pelapisan mahkota PFM (Opaque, Dentin, Enamel, Glaze)",
					IsActive:      true,
				},
				UsedAmount: 2,
				UsageCount: 2,
				UsageNote:  "Layering porselen mahkota PFM",
			},
			{
				Item: Item{
					Code:          "ITM-0007",
					Name:          "Valplast Flexible Nylon Resin Cartridge (Medium Pink)",
					Category:      "RAW_MATERIAL",
					UnitID:        getUnit("tube"),
					SupplierID:    getSup(1),
					MinStockLevel: 10,
					CurrentStock:  18,
					PurchasePrice: 175000,
					LotNumber:     strPtr("LOT-VAL-2610"),
					ExpiryDate:    dateOffset(45), // Expiring within 60 days
					Description:   "Cartridge nilon termoplastik fleksibel untuk injeksi gigi tiruan Valplast",
					IsActive:      true,
				},
				UsedAmount: 7,
				UsageCount: 3,
				UsageNote:  "Injeksi gigi tiruan fleksibel Valplast",
			},
			{
				Item: Item{
					Code:          "ITM-0008",
					Name:          "Kawat Ortodonti Stainless Steel 0.7mm & 0.8mm",
					Category:      "RAW_MATERIAL",
					UnitID:        getUnit("pcs"),
					SupplierID:    getSup(1),
					MinStockLevel: 5,
					CurrentStock:  12,
					PurchasePrice: 85000,
					LotNumber:     strPtr("LOT-ORT-2605"),
					ExpiryDate:    nil,
					Description:   "Kawat stainless steel medis untuk labial bow retainer & klamer cengkeram",
					IsActive:      true,
				},
				UsedAmount: 3,
				UsageCount: 2,
				UsageNote:  "Pembuatan labial bow & klamer retainer ortodonti",
			},
			{
				Item: Item{
					Code:          "ITM-0009",
					Name:          "Lilin Pola / Baseplate Modelling Wax Merah",
					Category:      "CONSUMABLE",
					UnitID:        getUnit("box"),
					SupplierID:    getSup(0),
					MinStockLevel: 4,
					CurrentStock:  9,
					PurchasePrice: 110000,
					LotNumber:     strPtr("LOT-WAX-2609"),
					ExpiryDate:    dateOffset(400),
					Description:   "Lilin malam pembuatan galangan gigit (bite rim) dan pola koping",
					IsActive:      true,
				},
				UsedAmount: 5,
				UsageCount: 3,
				UsageNote:  "Pembuatan galangan gigit & wax-up",
			},
			{
				Item: Item{
					Code:          "ITM-0010",
					Name:          "Mata Bur Karbida Tungsten & Karet Poles Silikon Set",
					Category:      "TOOL",
					UnitID:        getUnit("set"),
					SupplierID:    getSup(3),
					MinStockLevel: 4,
					CurrentStock:  6,
					PurchasePrice: 240000,
					LotNumber:     strPtr("LOT-BUR-2611"),
					ExpiryDate:    nil,
					Description:   "Instrumen trimming akrilik, pengasahan logam, dan pemolesan porselen/zirconia",
					IsActive:      true,
				},
				UsedAmount: 1,
				UsageCount: 1,
				UsageNote:  "Penggantian mata bur trimming meja teknisi",
			},
		}

		for _, entry := range seedItems {
			itm := entry.Item
			initialQty := itm.CurrentStock + entry.UsedAmount
			if err := s.db.Create(&itm).Error; err == nil {
				batch := InventoryBatch{
					ItemID:          itm.ID,
					LotNumber:       itm.LotNumber,
					ExpiryDate:      itm.ExpiryDate,
					InitialQuantity: initialQty,
					CurrentQuantity: itm.CurrentStock,
					UnitCost:        itm.PurchasePrice,
					ReceivedDate:    now.AddDate(0, 0, -15),
				}
				_ = s.db.Create(&batch).Error

				// Stock IN movement
				inMove := StockMovement{
					MovementType:  "RECEIPT",
					ItemID:        itm.ID,
					BatchID:       &batch.ID,
					Quantity:      initialQty,
					UnitCost:      itm.PurchasePrice,
					ReferenceType: "INITIAL_STOCK",
					Notes:         "Stok awal pembelian gudang dental lab",
					CreatedAt:     now.AddDate(0, 0, -15),
				}
				_ = s.db.Create(&inMove).Error

				// Stock OUT movement(s) for realistic usage history & top-used analytics
				if entry.UsedAmount > 0 && entry.UsageCount > 0 {
					perUse := entry.UsedAmount / float64(entry.UsageCount)
					for c := 0; c < entry.UsageCount; c++ {
						outMove := StockMovement{
							MovementType:  "ISSUE_PRODUCTION",
							ItemID:        itm.ID,
							BatchID:       &batch.ID,
							Quantity:      -perUse,
							UnitCost:      itm.PurchasePrice,
							ReferenceType: "PRODUCTION_ORDER",
							Notes:         entry.UsageNote,
							CreatedAt:     now.AddDate(0, 0, -(entry.UsageCount - c)),
						}
						_ = s.db.Create(&outMove).Error
					}
				}
			}
		}
	}
}

// Units
func (s *Service) ListUnits() ([]UnitOfMeasure, error) {
	var units []UnitOfMeasure
	err := s.db.Order("id ASC").Find(&units).Error
	return units, err
}

// Suppliers
func (s *Service) ListSuppliers(tenantID *uuid.UUID) ([]Supplier, error) {
	var suppliers []Supplier
	q := s.db.Where("deleted_at IS NULL")
	if tenantID != nil {
		q = q.Where("tenant_id = ?", *tenantID)
	}
	err := q.Order("name ASC").Find(&suppliers).Error
	return suppliers, err
}

func (s *Service) CreateSupplier(sup *Supplier) error {
	if sup.Name == "" {
		return errors.New("supplier name is required")
	}
	var count int64
	s.db.Model(&Supplier{}).Count(&count)
	if sup.Code == "" {
		sup.Code = fmt.Sprintf("SUP-%03d", count+1)
	}
	sup.IsActive = true
	return s.db.Create(sup).Error
}

// Items
type UpsertItemInput struct {
	TenantID      *uuid.UUID `json:"tenant_id"`
	Code          string     `json:"code"`
	Name          string     `json:"name"`
	Category      string     `json:"category"`
	UnitID        int        `json:"unit_id"`
	SupplierID    string     `json:"supplier_id"`
	MinStockLevel float64    `json:"min_stock_level"`
	CurrentStock  float64    `json:"current_stock"`
	PurchasePrice float64    `json:"purchase_price"`
	LotNumber     string     `json:"lot_number"`
	ExpiryDate    string     `json:"expiry_date"`
	Description   string     `json:"description"`
	UserID        uuid.UUID  `json:"-"`
}

func (s *Service) ListItems(search string, category string, tenantID *uuid.UUID) ([]Item, error) {
	var items []Item
	q := s.db.Preload("Unit").Preload("Supplier").Where("deleted_at IS NULL")
	if tenantID != nil {
		q = q.Where("tenant_id = ?", *tenantID)
	}
	if search != "" {
		likeTerm := "%" + search + "%"
		q = q.Where("name ILIKE ? OR code ILIKE ? OR lot_number ILIKE ?", likeTerm, likeTerm, likeTerm)
	}
	if category != "" {
		q = q.Where("category = ?", category)
	}
	err := q.Order("code ASC").Find(&items).Error
	return items, err
}

func (s *Service) CreateItem(input UpsertItemInput) (*Item, error) {
	if input.Name == "" || input.UnitID == 0 {
		return nil, errors.New("nama bahan dan satuan wajib diisi")
	}

	var count int64
	s.db.Model(&Item{}).Count(&count)
	code := strings.TrimSpace(input.Code)
	if code == "" {
		code = fmt.Sprintf("ITM-%04d", count+1)
	}
	cat := input.Category
	if cat == "" {
		cat = "RAW_MATERIAL"
	}

	var supID *uuid.UUID
	if input.SupplierID != "" {
		if parsed, err := uuid.Parse(input.SupplierID); err == nil {
			supID = &parsed
		}
	}

	var lotPtr *string
	if strings.TrimSpace(input.LotNumber) != "" {
		l := strings.TrimSpace(input.LotNumber)
		lotPtr = &l
	}

	expPtr := parseFlexibleDate(input.ExpiryDate)

	item := Item{
		TenantID:      input.TenantID,
		Code:          code,
		Name:          input.Name,
		Category:      cat,
		UnitID:        input.UnitID,
		SupplierID:    supID,
		MinStockLevel: input.MinStockLevel,
		CurrentStock:  input.CurrentStock,
		PurchasePrice: input.PurchasePrice,
		LotNumber:     lotPtr,
		ExpiryDate:    expPtr,
		Description:   input.Description,
		IsActive:      true,
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&item).Error; err != nil {
			return err
		}

		// If initial stock > 0 or batch info is supplied, create initial batch & stock movement
		if item.CurrentStock > 0 || lotPtr != nil || expPtr != nil {
			batch := InventoryBatch{
				ItemID:          item.ID,
				LotNumber:       lotPtr,
				ExpiryDate:      expPtr,
				InitialQuantity: item.CurrentStock,
				CurrentQuantity: item.CurrentStock,
				UnitCost:        item.PurchasePrice,
				ReceivedDate:    time.Now(),
			}
			if err := tx.Create(&batch).Error; err != nil {
				return err
			}

			if item.CurrentStock > 0 {
				var createdBy *uuid.UUID
				if input.UserID != uuid.Nil {
					createdBy = &input.UserID
				}
				mv := StockMovement{
					MovementType:  "RECEIPT",
					ItemID:        item.ID,
					BatchID:       &batch.ID,
					Quantity:      item.CurrentStock,
					UnitCost:      item.PurchasePrice,
					ReferenceType: "INITIAL_MASTER",
					Notes:         "Saldo stok awal master bahan/produk",
					CreatedBy:     createdBy,
				}
				if err := tx.Create(&mv).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	var created Item
	_ = s.db.Preload("Unit").Preload("Supplier").First(&created, "id = ?", item.ID).Error
	return &created, nil
}

func (s *Service) UpdateItem(id uuid.UUID, input UpsertItemInput) (*Item, error) {
	var existing Item
	if err := s.db.First(&existing, "id = ? AND deleted_at IS NULL", id).Error; err != nil {
		return nil, errors.New("item not found")
	}

	updates := map[string]interface{}{
		"updated_at": time.Now(),
	}
	if input.Name != "" {
		updates["name"] = input.Name
	}
	if input.Category != "" {
		updates["category"] = input.Category
	}
	if input.UnitID != 0 {
		updates["unit_id"] = input.UnitID
	}
	if input.SupplierID != "" {
		if parsed, err := uuid.Parse(input.SupplierID); err == nil {
			updates["supplier_id"] = parsed
		}
	}
	updates["min_stock_level"] = input.MinStockLevel
	if input.PurchasePrice >= 0 {
		updates["purchase_price"] = input.PurchasePrice
	}
	if input.Description != "" {
		updates["description"] = input.Description
	}
	if strings.TrimSpace(input.LotNumber) != "" {
		updates["lot_number"] = strings.TrimSpace(input.LotNumber)
	}
	if exp := parseFlexibleDate(input.ExpiryDate); exp != nil {
		updates["expiry_date"] = exp
	}

	if err := s.db.Model(&Item{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return nil, err
	}

	var updated Item
	_ = s.db.Preload("Unit").Preload("Supplier").First(&updated, "id = ?", id).Error
	return &updated, nil
}

// Stock Movements & Batches
type ReceiveMaterialInput struct {
	ItemID     uuid.UUID `json:"item_id"`
	LotNumber  string    `json:"lot_number"`
	ExpiryDate string    `json:"expiry_date"`
	Quantity   float64   `json:"quantity"`
	UnitCost   float64   `json:"unit_cost"`
	SupplierID string    `json:"supplier_id"`
	Notes      string    `json:"notes"`
	UserID     uuid.UUID `json:"-"`
}

func (s *Service) ReceiveMaterial(input ReceiveMaterialInput) (*StockMovement, error) {
	if input.Quantity <= 0 {
		return nil, errors.New("jumlah stok masuk harus lebih dari 0")
	}

	var lotPtr *string
	if strings.TrimSpace(input.LotNumber) != "" {
		l := strings.TrimSpace(input.LotNumber)
		lotPtr = &l
	}
	expPtr := parseFlexibleDate(input.ExpiryDate)

	var movement StockMovement
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var item Item
		if err := tx.First(&item, "id = ?", input.ItemID).Error; err != nil {
			return errors.New("bahan tidak ditemukan")
		}

		unitCost := input.UnitCost
		if unitCost <= 0 {
			unitCost = item.PurchasePrice
		}

		// 1. Create batch
		batch := InventoryBatch{
			ItemID:          input.ItemID,
			LotNumber:       lotPtr,
			ExpiryDate:      expPtr,
			InitialQuantity: input.Quantity,
			CurrentQuantity: input.Quantity,
			UnitCost:        unitCost,
			ReceivedDate:    time.Now(),
		}
		if err := tx.Create(&batch).Error; err != nil {
			return err
		}

		// 2. Insert stock movement
		notes := input.Notes
		if notes == "" {
			notes = "Penerimaan stok bahan masuk (Pembelian / Restock)"
		}
		movement = StockMovement{
			MovementType:  "RECEIPT",
			ItemID:        input.ItemID,
			BatchID:       &batch.ID,
			Quantity:      input.Quantity,
			UnitCost:      unitCost,
			ReferenceType: "MANUAL_RECEIPT",
			Notes:         notes,
			CreatedBy:     &input.UserID,
		}
		if err := tx.Create(&movement).Error; err != nil {
			return err
		}

		// 3. Update Item stock, purchase price, and latest batch/expiry info
		itemUpdates := map[string]interface{}{
			"current_stock":  gorm.Expr("current_stock + ?", input.Quantity),
			"purchase_price": unitCost,
			"updated_at":     time.Now(),
		}
		if lotPtr != nil {
			itemUpdates["lot_number"] = *lotPtr
		}
		if expPtr != nil {
			itemUpdates["expiry_date"] = *expPtr
		}
		if input.SupplierID != "" {
			if supUUID, err := uuid.Parse(input.SupplierID); err == nil {
				itemUpdates["supplier_id"] = supUUID
			}
		}

		if err := tx.Model(&Item{}).Where("id = ?", input.ItemID).Updates(itemUpdates).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}
	return &movement, nil
}

type IssueMaterialInput struct {
	ItemID            uuid.UUID  `json:"item_id"`
	BatchID           *uuid.UUID `json:"batch_id"`
	Quantity          float64    `json:"quantity"`
	MovementType      string     `json:"movement_type"` // ISSUE_PRODUCTION, ADJUSTMENT
	ProductionOrderID *uuid.UUID `json:"production_order_id"`
	Notes             string     `json:"notes"`
	UserID            uuid.UUID  `json:"-"`
}

func (s *Service) IssueMaterial(input IssueMaterialInput) (*StockMovement, error) {
	if input.Quantity <= 0 {
		return nil, errors.New("jumlah pemakaian harus lebih dari 0")
	}

	var movement StockMovement
	err := s.db.Transaction(func(tx *gorm.DB) error {
		var item Item
		if err := tx.First(&item, "id = ?", input.ItemID).Error; err != nil {
			return errors.New("bahan tidak ditemukan")
		}
		if item.CurrentStock < input.Quantity {
			return fmt.Errorf("stok %s tidak mencukupi (tersedia: %.2f, dibutuhkan: %.2f)", item.Name, item.CurrentStock, input.Quantity)
		}

		var usedBatchID *uuid.UUID
		if input.BatchID != nil && *input.BatchID != uuid.Nil {
			usedBatchID = input.BatchID
			_ = tx.Model(&InventoryBatch{}).Where("id = ?", *input.BatchID).
				Update("current_quantity", gorm.Expr("GREATEST(current_quantity - ?, 0)", input.Quantity)).Error
		} else {
			// Deduct from earliest expiring active batch (FEFO / FIFO)
			var batches []InventoryBatch
			tx.Where("item_id = ? AND current_quantity > 0", input.ItemID).
				Order("expiry_date ASC NULLS LAST, received_date ASC").
				Find(&batches)

			remaining := input.Quantity
			for _, b := range batches {
				if remaining <= 0 {
					break
				}
				if usedBatchID == nil {
					bID := b.ID
					usedBatchID = &bID
				}
				deduct := remaining
				if b.CurrentQuantity < deduct {
					deduct = b.CurrentQuantity
				}
				_ = tx.Model(&InventoryBatch{}).Where("id = ?", b.ID).
					Update("current_quantity", gorm.Expr("GREATEST(current_quantity - ?, 0)", deduct)).Error
				remaining -= deduct
			}
		}

		mvType := input.MovementType
		if mvType == "" {
			mvType = "ISSUE_PRODUCTION"
		}
		refType := "PRODUCTION_USAGE"
		if input.ProductionOrderID != nil {
			refType = "PRODUCTION_ORDER"
		}
		notes := input.Notes
		if notes == "" {
			notes = "Pemakaian bahan produksi"
		}

		movement = StockMovement{
			MovementType:  mvType,
			ItemID:        input.ItemID,
			BatchID:       usedBatchID,
			Quantity:      -input.Quantity,
			UnitCost:      item.PurchasePrice,
			ReferenceType: refType,
			ReferenceID:   input.ProductionOrderID,
			Notes:         notes,
			CreatedBy:     &input.UserID,
		}
		if err := tx.Create(&movement).Error; err != nil {
			return err
		}

		// Record in production_material_consumptions if linked to an order
		if input.ProductionOrderID != nil && *input.ProductionOrderID != uuid.Nil {
			_ = tx.Exec(`INSERT INTO production_material_consumptions (production_order_id, item_id, batch_id, quantity, unit_cost, total_cost, recorded_by) VALUES (?, ?, ?, ?, ?, ?, ?)`,
				*input.ProductionOrderID, input.ItemID, usedBatchID, input.Quantity, item.PurchasePrice, input.Quantity*item.PurchasePrice, input.UserID).Error
		}

		// Update Item current_stock
		if err := tx.Model(&Item{}).Where("id = ?", input.ItemID).Updates(map[string]interface{}{
			"current_stock": gorm.Expr("current_stock - ?", input.Quantity),
			"updated_at":    time.Now(),
		}).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}
	return &movement, nil
}

func (s *Service) ListBatches(itemID string) ([]InventoryBatch, error) {
	var batches []InventoryBatch
	q := s.db.Preload("Item.Unit").Preload("Item.Supplier")
	if itemID != "" {
		q = q.Where("item_id = ?", itemID)
	}
	err := q.Order("expiry_date ASC NULLS LAST, received_date DESC").Find(&batches).Error
	return batches, err
}

func (s *Service) ListMovements(itemID string, movementType string) ([]StockMovement, error) {
	var movements []StockMovement
	q := s.db.Preload("Item.Unit").Preload("Batch")
	if itemID != "" {
		q = q.Where("item_id = ?", itemID)
	}
	if movementType == "IN" {
		q = q.Where("quantity > 0")
	} else if movementType == "OUT" {
		q = q.Where("quantity < 0")
	}
	err := q.Order("created_at DESC").Limit(100).Find(&movements).Error
	return movements, err
}

func (s *Service) GetInventoryReport(tenantID *uuid.UUID) (*InventoryReportSummary, error) {
	var items []Item
	q := s.db.Preload("Unit").Preload("Supplier").Where("deleted_at IS NULL")
	if tenantID != nil {
		q = q.Where("tenant_id = ?", *tenantID)
	}
	if err := q.Order("current_stock ASC").Find(&items).Error; err != nil {
		return nil, err
	}

	summary := &InventoryReportSummary{
		TotalItems:       int64(len(items)),
		LowStockItems:    []Item{},
		TopUsedMaterials: []TopUsedMaterial{},
	}

	now := time.Now()
	soonThreshold := now.AddDate(0, 0, 60)

	for _, itm := range items {
		summary.TotalStockValue += itm.CurrentStock * itm.PurchasePrice
		if itm.CurrentStock <= itm.MinStockLevel {
			summary.LowStockCount++
			summary.LowStockItems = append(summary.LowStockItems, itm)
		}
	}

	// Count expiring soon / expired from active batches (or items)
	var batches []InventoryBatch
	bQuery := s.db.Where("current_quantity > 0 AND expiry_date IS NOT NULL")
	if tenantID != nil {
		bQuery = bQuery.Joins("JOIN items i ON i.id = inventory_batches.item_id AND i.tenant_id = ?", *tenantID)
	}
	bQuery.Find(&batches)
	for _, b := range batches {
		if b.ExpiryDate != nil {
			if b.ExpiryDate.Before(now) {
				summary.ExpiredCount++
			} else if b.ExpiryDate.Before(soonThreshold) {
				summary.ExpiringSoonCount++
			}
		}
	}

	// Top used materials from negative stock_movements
	type usageRow struct {
		ItemID     uuid.UUID
		ItemCode   string
		ItemName   string
		Category   string
		UnitCode   string
		TotalUsed  float64
		UsageCount int64
		TotalValue float64
	}
	var rows []usageRow
	smQuery := s.db.Table("stock_movements sm").
		Select("sm.item_id, i.code as item_code, i.name as item_name, i.category, u.code as unit_code, SUM(ABS(sm.quantity)) as total_used, COUNT(sm.id) as usage_count, SUM(ABS(sm.quantity) * sm.unit_cost) as total_value").
		Joins("JOIN items i ON i.id = sm.item_id").
		Joins("LEFT JOIN units_of_measure u ON u.id = i.unit_id").
		Where("sm.quantity < 0 AND i.deleted_at IS NULL")
	if tenantID != nil {
		smQuery = smQuery.Where("i.tenant_id = ?", *tenantID)
	}
	smQuery.Group("sm.item_id, i.code, i.name, i.category, u.code").
		Order("total_used DESC, usage_count DESC").
		Limit(10).
		Scan(&rows)

	for _, r := range rows {
		summary.TopUsedMaterials = append(summary.TopUsedMaterials, TopUsedMaterial{
			ItemID:     r.ItemID,
			ItemCode:   r.ItemCode,
			ItemName:   r.ItemName,
			Category:   r.Category,
			UnitCode:   r.UnitCode,
			TotalUsed:  r.TotalUsed,
			UsageCount: r.UsageCount,
			TotalValue: r.TotalValue,
		})
	}

	return summary, nil
}

// Purchase Requests with Owner Approval
func (s *Service) ListPurchaseRequests(status string, tenantID *uuid.UUID) ([]PurchaseRequest, error) {
	var prs []PurchaseRequest
	q := s.db.Preload("Supplier").Preload("Items.Item.Unit")
	if tenantID != nil {
		q = q.Where("tenant_id = ?", *tenantID)
	}
	if status != "" {
		q = q.Where("status = ?", status)
	}
	err := q.Order("created_at DESC").Find(&prs).Error
	return prs, err
}

func (s *Service) CreatePurchaseRequest(req *PurchaseRequest) error {
	var count int64
	s.db.Model(&PurchaseRequest{}).Count(&count)
	req.PRNumber = fmt.Sprintf("PR-%04d", count+1)
	req.Status = "PENDING_APPROVAL" // Direct to Owner review

	var total float64
	for i := range req.Items {
		subtotal := req.Items[i].Quantity * req.Items[i].EstimatedUnitPrice
		req.Items[i].Subtotal = subtotal
		total += subtotal
	}
	req.TotalAmount = total

	return s.db.Create(req).Error
}

func (s *Service) ApprovePurchaseRequest(id uuid.UUID, ownerID uuid.UUID) error {
	now := time.Now()
	res := s.db.Model(&PurchaseRequest{}).
		Where("id = ? AND status = 'PENDING_APPROVAL'", id).
		Updates(map[string]interface{}{
			"status":      "APPROVED_BY_OWNER",
			"approved_by": ownerID,
			"approved_at": &now,
			"updated_at":  now,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errors.New("purchase request not found or not in PENDING_APPROVAL status")
	}
	return nil
}

func (s *Service) RejectPurchaseRequest(id uuid.UUID, ownerID uuid.UUID, reason string) error {
	now := time.Now()
	res := s.db.Model(&PurchaseRequest{}).
		Where("id = ? AND status = 'PENDING_APPROVAL'", id).
		Updates(map[string]interface{}{
			"status":           "REJECTED",
			"approved_by":      ownerID,
			"approved_at":      &now,
			"rejection_reason": reason,
			"updated_at":       now,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errors.New("purchase request not found or not in PENDING_APPROVAL status")
	}
	return nil
}

// Receive items from an approved PR
func (s *Service) ReceiveApprovedPR(prID uuid.UUID, userID uuid.UUID) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var pr PurchaseRequest
		if err := tx.Preload("Items").Where("id = ? AND status = 'APPROVED_BY_OWNER'", prID).First(&pr).Error; err != nil {
			return errors.New("PR must be APPROVED_BY_OWNER before receiving stock")
		}

		for _, item := range pr.Items {
			lotNum := fmt.Sprintf("LOT-%s", pr.PRNumber)
			// Create batch
			batch := InventoryBatch{
				ItemID:          item.ItemID,
				LotNumber:       &lotNum,
				InitialQuantity: item.Quantity,
				CurrentQuantity: item.Quantity,
				UnitCost:        item.ActualUnitPrice,
				ReceivedDate:    time.Now(),
			}
			if batch.UnitCost == 0 {
				batch.UnitCost = item.EstimatedUnitPrice
			}
			if err := tx.Create(&batch).Error; err != nil {
				return err
			}

			// Create movement
			movement := StockMovement{
				MovementType:  "RECEIPT",
				ItemID:        item.ItemID,
				BatchID:       &batch.ID,
				Quantity:      item.Quantity,
				UnitCost:      batch.UnitCost,
				ReferenceType: "PURCHASE_REQUEST",
				ReferenceID:   &pr.ID,
				Notes:         fmt.Sprintf("Penerimaan pembelian dari %s", pr.PRNumber),
				CreatedBy:     &userID,
			}
			if err := tx.Create(&movement).Error; err != nil {
				return err
			}

			// Update stock
			if err := tx.Model(&Item{}).Where("id = ?", item.ItemID).Updates(map[string]interface{}{
				"current_stock":  gorm.Expr("current_stock + ?", item.Quantity),
				"purchase_price": batch.UnitCost,
				"lot_number":     lotNum,
				"updated_at":     time.Now(),
			}).Error; err != nil {
				return err
			}
		}

		// Update PR status
		return tx.Model(&pr).Updates(map[string]interface{}{
			"status":     "RECEIVED",
			"updated_at": time.Now(),
		}).Error
	})
}
