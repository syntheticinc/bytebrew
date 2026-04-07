package config_repo

import (
	"context"
	"fmt"

	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/models"
	"gorm.io/gorm"
)

// TenantRecord is an intermediate struct for DB <-> domain mapping.
type TenantRecord struct {
	ID        uint
	TenantUID string
	Email     string
	PlanType  string
}

// GORMTenantRepository implements tenant CRUD using GORM.
type GORMTenantRepository struct {
	db *gorm.DB
}

// NewGORMTenantRepository creates a new GORMTenantRepository.
func NewGORMTenantRepository(db *gorm.DB) *GORMTenantRepository {
	return &GORMTenantRepository{db: db}
}

// Create inserts a new tenant.
func (r *GORMTenantRepository) Create(ctx context.Context, record *TenantRecord) error {
	model := models.TenantModel{
		TenantUID: record.TenantUID,
		Email:     record.Email,
		PlanType:  record.PlanType,
	}
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return fmt.Errorf("create tenant: %w", err)
	}
	record.ID = model.ID
	return nil
}

// GetByUID returns a tenant by its external UID.
func (r *GORMTenantRepository) GetByUID(ctx context.Context, uid string) (*TenantRecord, error) {
	var tenant models.TenantModel
	if err := r.db.WithContext(ctx).Where("tenant_uid = ?", uid).First(&tenant).Error; err != nil {
		return nil, fmt.Errorf("get tenant %q: %w", uid, err)
	}
	return &TenantRecord{
		ID:        tenant.ID,
		TenantUID: tenant.TenantUID,
		Email:     tenant.Email,
		PlanType:  tenant.PlanType,
	}, nil
}

// GetByEmail returns a tenant by email.
func (r *GORMTenantRepository) GetByEmail(ctx context.Context, email string) (*TenantRecord, error) {
	var tenant models.TenantModel
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&tenant).Error; err != nil {
		return nil, fmt.Errorf("get tenant by email %q: %w", email, err)
	}
	return &TenantRecord{
		ID:        tenant.ID,
		TenantUID: tenant.TenantUID,
		Email:     tenant.Email,
		PlanType:  tenant.PlanType,
	}, nil
}

// Update updates a tenant's plan.
func (r *GORMTenantRepository) Update(ctx context.Context, uid string, planType string) error {
	result := r.db.WithContext(ctx).Model(&models.TenantModel{}).Where("tenant_uid = ?", uid).
		Update("plan_type", planType)
	if result.Error != nil {
		return fmt.Errorf("update tenant %q: %w", uid, result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
