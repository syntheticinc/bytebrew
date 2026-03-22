package domain

import (
	"fmt"
	"time"
)

// ProjectMetadata represents project metadata
type ProjectMetadata struct {
	ProjectKey string
	Name       string
	RootPath   string
	UserID     string
	Files      []FileDescription
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// FileDescription represents file metadata with AI-generated description
type FileDescription struct {
	FilePath    string
	Description string
	Language    string
	SizeBytes   int64
}

// NewProjectMetadata creates a new ProjectMetadata with validation
func NewProjectMetadata(projectKey, name, rootPath, userID string) (*ProjectMetadata, error) {
	pm := &ProjectMetadata{
		ProjectKey: projectKey,
		Name:       name,
		RootPath:   rootPath,
		UserID:     userID,
		Files:      make([]FileDescription, 0),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := pm.Validate(); err != nil {
		return nil, err
	}

	return pm, nil
}

// Validate validates the ProjectMetadata
func (pm *ProjectMetadata) Validate() error {
	if pm.ProjectKey == "" {
		return fmt.Errorf("project_key is required")
	}
	if pm.Name == "" {
		return fmt.Errorf("name is required")
	}
	if pm.RootPath == "" {
		return fmt.Errorf("root_path is required")
	}
	if pm.UserID == "" {
		return fmt.Errorf("user_id is required")
	}
	return nil
}

// AddFile adds a file description to the project
func (pm *ProjectMetadata) AddFile(file FileDescription) error {
	if file.FilePath == "" {
		return fmt.Errorf("file_path is required")
	}
	if file.Language == "" {
		return fmt.Errorf("language is required")
	}

	pm.Files = append(pm.Files, file)
	pm.UpdatedAt = time.Now()
	return nil
}

// FileCount returns the number of files in the project
func (pm *ProjectMetadata) FileCount() int {
	return len(pm.Files)
}

// TotalSize returns the total size of all files in bytes
func (pm *ProjectMetadata) TotalSize() int64 {
	var total int64
	for _, file := range pm.Files {
		total += file.SizeBytes
	}
	return total
}
