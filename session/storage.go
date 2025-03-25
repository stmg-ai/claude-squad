package session

import (
	"claude-squad/config"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// GitWorktreeData represents the serializable data of a GitWorktree
type GitWorktreeData struct {
	RepoPath      string
	WorktreePath  string
	SessionName   string
	BranchName    string
	BaseCommitSHA string
}

// DiffStatsData represents the serializable data of a DiffStats
type DiffStatsData struct {
	Added   int
	Removed int
	Content string
}

// InstanceData represents the serializable data of an Instance
type InstanceData struct {
	Title     string
	Path      string
	Branch    string
	Status    Status
	Height    int
	Width     int
	CreatedAt time.Time
	UpdatedAt time.Time
	AutoYes   bool

	Program   string
	Worktree  GitWorktreeData
	DiffStats DiffStatsData
}

// Storage handles saving and loading instances
type Storage struct {
	filePath  string
	backupDir string
}

// NewStorage creates a new storage instance
func NewStorage() (*Storage, error) {
	dir, err := config.GetConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get config directory: %w", err)
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	backupDir := filepath.Join(dir, "backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}

	return &Storage{
		filePath:  filepath.Join(dir, "instances.json"),
		backupDir: backupDir,
	}, nil
}

// SaveInstances saves the list of instances to disk
func (s *Storage) SaveInstances(instances []*Instance) error {
	// Create backup if file exists
	if _, err := os.Stat(s.filePath); err == nil {
		timestamp := time.Now().Format("20060102_150405")
		backupFile := filepath.Join(s.backupDir, fmt.Sprintf("instances_%s.json", timestamp))
		if data, err := os.ReadFile(s.filePath); err == nil {
			err = os.WriteFile(backupFile, data, 0644)
			if err != nil {
				return fmt.Errorf("failed to create backup: %w", err)
			}
		}
	}

	// Convert and save instances
	data := make([]InstanceData, 0)
	for _, instance := range instances {
		if instance.Started() {
			data = append(data, instance.ToInstanceData())
		}
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal instances: %w", err)
	}

	return os.WriteFile(s.filePath, jsonData, 0644)
}

// LoadInstances loads the list of instances from disk
func (s *Storage) LoadInstances() ([]*Instance, error) {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []*Instance{}, nil
		}
		return nil, fmt.Errorf("failed to read instances: %w", err)
	}

	var instanceData []InstanceData
	if err := json.Unmarshal(data, &instanceData); err != nil {
		return nil, fmt.Errorf("failed to parse instances: %w", err)
	}

	instances := make([]*Instance, len(instanceData))
	for i, data := range instanceData {
		instance, err := FromInstanceData(data)
		if err != nil {
			return nil, fmt.Errorf("failed to create instance %s: %w", data.Title, err)
		}
		instances[i] = instance
	}

	return instances, nil
}

// DeleteInstance removes an instance from storage
func (s *Storage) DeleteInstance(title string) error {
	instances, err := s.LoadInstances()
	if err != nil {
		return err
	}

	for i, instance := range instances {
		if instance.Title == title {
			instances = append(instances[:i], instances[i+1:]...)
			return s.SaveInstances(instances)
		}
	}

	return fmt.Errorf("instance not found: %s", title)
}

// UpdateInstance updates an existing instance in storage
func (s *Storage) UpdateInstance(instance *Instance) error {
	instances, err := s.LoadInstances()
	if err != nil {
		return err
	}

	for i, existing := range instances {
		if existing.Title == instance.Title {
			instances[i] = instance
			return s.SaveInstances(instances)
		}
	}

	return fmt.Errorf("instance not found: %s", instance.Title)
}

// DeleteAllInstances removes all stored instances and their backups
func (s *Storage) DeleteAllInstances() error {
	// Remove the main instances file
	if err := os.Remove(s.filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete instances file: %w", err)
	}

	// Remove all backup files
	entries, err := os.ReadDir(s.backupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read backup directory: %w", err)
	}

	for _, entry := range entries {
		if err := os.Remove(filepath.Join(s.backupDir, entry.Name())); err != nil {
			return fmt.Errorf("failed to delete backup file %s: %w", entry.Name(), err)
		}
	}

	return nil
}
