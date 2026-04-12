package persistence

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/syntheticinc/bytebrew/engine/internal/domain"
	"github.com/syntheticinc/bytebrew/engine/internal/infrastructure/persistence/models"
	"gorm.io/gorm"
)

// MemoryStorage implements memory persistence using GORM (PostgreSQL).
type MemoryStorage struct {
	db *gorm.DB
}

// NewMemoryStorage creates a new memory storage.
func NewMemoryStorage(db *gorm.DB) *MemoryStorage {
	slog.Info("memory storage initialized (PostgreSQL)")
	return &MemoryStorage{db: db}
}

// Store persists a memory entry. If max_entries is reached, evicts the oldest (FIFO).
func (s *MemoryStorage) Store(ctx context.Context, mem *domain.Memory, maxEntries int) error {
	if maxEntries > 0 {
		if err := s.evictIfNeeded(ctx, mem.SchemaID, mem.UserID, maxEntries); err != nil {
			return fmt.Errorf("evict memories: %w", err)
		}
	}

	m := memoryToModel(mem)
	if m.ID == "" {
		m.ID = uuid.New().String()
	}
	if err := s.db.WithContext(ctx).Create(&m).Error; err != nil {
		return fmt.Errorf("insert memory: %w", err)
	}

	mem.ID = m.ID
	slog.DebugContext(ctx, "memory stored", "id", mem.ID, "schema_id", mem.SchemaID, "user_id", mem.UserID)
	return nil
}

// ListBySchema retrieves all memories for a schema, ordered by most recent first.
func (s *MemoryStorage) ListBySchema(ctx context.Context, schemaID string) ([]*domain.Memory, error) {
	var ms []models.MemoryModel
	err := s.db.WithContext(ctx).
		Where("schema_id = ?", schemaID).
		Order("created_at DESC").
		Find(&ms).Error
	if err != nil {
		return nil, fmt.Errorf("list memories by schema: %w", err)
	}
	return modelsToMemories(ms), nil
}

// ListBySchemaAndUser retrieves memories for a schema+user pair.
func (s *MemoryStorage) ListBySchemaAndUser(ctx context.Context, schemaID, userID string) ([]*domain.Memory, error) {
	var ms []models.MemoryModel
	err := s.db.WithContext(ctx).
		Where("schema_id = ? AND user_id = ?", schemaID, userID).
		Order("created_at DESC").
		Find(&ms).Error
	if err != nil {
		return nil, fmt.Errorf("list memories by schema+user: %w", err)
	}
	return modelsToMemories(ms), nil
}

// DeleteBySchema deletes all memories for a schema.
func (s *MemoryStorage) DeleteBySchema(ctx context.Context, schemaID string) (int64, error) {
	result := s.db.WithContext(ctx).
		Where("schema_id = ?", schemaID).
		Delete(&models.MemoryModel{})
	if result.Error != nil {
		return 0, fmt.Errorf("delete memories by schema: %w", result.Error)
	}
	slog.InfoContext(ctx, "memories cleared", "schema_id", schemaID, "count", result.RowsAffected)
	return result.RowsAffected, nil
}

// DeleteByID deletes a single memory entry by ID.
func (s *MemoryStorage) DeleteByID(ctx context.Context, id string) error {
	result := s.db.WithContext(ctx).
		Where("id = ?", id).
		Delete(&models.MemoryModel{})
	if result.Error != nil {
		return fmt.Errorf("delete memory: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("memory not found: %s", id)
	}
	return nil
}

// CountBySchemaAndUser returns the number of memories for a schema+user pair.
func (s *MemoryStorage) CountBySchemaAndUser(ctx context.Context, schemaID, userID string) (int, error) {
	var count int64
	err := s.db.WithContext(ctx).
		Model(&models.MemoryModel{}).
		Where("schema_id = ? AND user_id = ?", schemaID, userID).
		Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("count memories: %w", err)
	}
	return int(count), nil
}

// evictIfNeeded removes the oldest entries when count >= maxEntries (FIFO, AC-MEM-RET-03).
func (s *MemoryStorage) evictIfNeeded(ctx context.Context, schemaID, userID string, maxEntries int) error {
	count, err := s.CountBySchemaAndUser(ctx, schemaID, userID)
	if err != nil {
		return err
	}

	// Need to make room for the new entry
	toDelete := count - maxEntries + 1
	if toDelete <= 0 {
		return nil
	}

	// Find IDs of oldest entries to delete
	var oldest []models.MemoryModel
	err = s.db.WithContext(ctx).
		Where("schema_id = ? AND user_id = ?", schemaID, userID).
		Order("created_at ASC").
		Limit(toDelete).
		Find(&oldest).Error
	if err != nil {
		return fmt.Errorf("find oldest memories: %w", err)
	}

	ids := make([]string, len(oldest))
	for i, m := range oldest {
		ids[i] = m.ID
	}

	if err := s.db.WithContext(ctx).Where("id IN ?", ids).Delete(&models.MemoryModel{}).Error; err != nil {
		return fmt.Errorf("delete oldest memories: %w", err)
	}

	slog.DebugContext(ctx, "FIFO eviction", "schema_id", schemaID, "user_id", userID, "evicted", len(ids))
	return nil
}

// CleanupExpiredBySchema deletes memories older than retentionDays for a given schema.
func (s *MemoryStorage) CleanupExpiredBySchema(ctx context.Context, schemaID string, retentionDays int) (int64, error) {
	if retentionDays <= 0 {
		return 0, nil
	}
	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	result := s.db.WithContext(ctx).
		Where("schema_id = ? AND created_at < ?", schemaID, cutoff).
		Delete(&models.MemoryModel{})
	if result.Error != nil {
		return 0, fmt.Errorf("cleanup expired memories: %w", result.Error)
	}
	if result.RowsAffected > 0 {
		slog.InfoContext(ctx, "expired memories cleaned",
			"schema_id", schemaID, "retention_days", retentionDays, "deleted", result.RowsAffected)
	}
	return result.RowsAffected, nil
}

func memoryToModel(mem *domain.Memory) models.MemoryModel {
	metaJSON := "{}"
	if len(mem.Metadata) > 0 {
		if b, err := json.Marshal(mem.Metadata); err == nil {
			metaJSON = string(b)
		}
	}

	return models.MemoryModel{
		SchemaID: mem.SchemaID,
		UserID:   mem.UserID,
		Content:  mem.Content,
		Metadata: metaJSON,
	}
}

func modelToMemory(m *models.MemoryModel) *domain.Memory {
	metadata := make(map[string]string)
	if m.Metadata != "" {
		_ = json.Unmarshal([]byte(m.Metadata), &metadata)
	}

	return &domain.Memory{
		ID:        m.ID,
		SchemaID:  m.SchemaID,
		UserID:    m.UserID,
		Content:   m.Content,
		Metadata:  metadata,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

func modelsToMemories(ms []models.MemoryModel) []*domain.Memory {
	memories := make([]*domain.Memory, 0, len(ms))
	for i := range ms {
		memories = append(memories, modelToMemory(&ms[i]))
	}
	return memories
}
