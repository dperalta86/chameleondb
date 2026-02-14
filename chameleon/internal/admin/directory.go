package admin

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/chameleon-db/chameleondb/chameleon/internal/config"
	"github.com/chameleon-db/chameleondb/chameleon/internal/journal"
	"github.com/chameleon-db/chameleondb/chameleon/internal/state"
)

// Directory manages the .chameleon/ directory structure
type Directory struct {
	rootDir string // .chameleon/
}

// NewDirectory creates a new directory manager
func NewDirectory(workDir string) *Directory {
	return &Directory{
		rootDir: filepath.Join(workDir, ".chameleon"),
	}
}

// Initialize creates the .chameleon/ directory structure
func (d *Directory) Initialize() error {
	// Create main directory
	if err := os.MkdirAll(d.rootDir, 0755); err != nil {
		return fmt.Errorf("failed to create .chameleon directory: %w", err)
	}

	// Create subdirectories
	subdirs := []string{
		"state",
		"state/migrations",
		"journal",
		"backups",
	}

	for _, subdir := range subdirs {
		path := filepath.Join(d.rootDir, subdir)
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("failed to create %s: %w", subdir, err)
		}
	}

	// Create .gitignore
	if err := d.createGitignore(); err != nil {
		return err
	}

	return nil
}

// createGitignore creates .chameleon/.gitignore
func (d *Directory) createGitignore() error {
	gitignorePath := filepath.Join(d.rootDir, ".gitignore")
	gitignoreContent := `# ChameleonDB Administrative Files
# State: Local to each developer
state/
journal/
backups/

# Keep config.yml (versioned)
!config.yml
`

	return os.WriteFile(gitignorePath, []byte(gitignoreContent), 0644)
}

// GetPaths returns all directory paths
func (d *Directory) GetPaths() DirectoryPaths {
	return DirectoryPaths{
		Root:       d.rootDir,
		Config:     filepath.Join(d.rootDir, "config.yml"),
		State:      filepath.Join(d.rootDir, "state"),
		Migrations: filepath.Join(d.rootDir, "state", "migrations"),
		Journal:    filepath.Join(d.rootDir, "journal"),
		Backups:    filepath.Join(d.rootDir, "backups"),
	}
}

// DirectoryPaths holds all important paths
type DirectoryPaths struct {
	Root       string
	Config     string
	State      string
	Migrations string
	Journal    string
	Backups    string
}

// ManagerFactory creates and initializes all managers
type ManagerFactory struct {
	workDir string
	dir     *Directory
}

// NewManagerFactory creates a new manager factory
func NewManagerFactory(workDir string) *ManagerFactory {
	return &ManagerFactory{
		workDir: workDir,
		dir:     NewDirectory(workDir),
	}
}

// Initialize initializes the entire .chameleon/ structure
func (mf *ManagerFactory) Initialize() error {
	// Create directory structure
	if err := mf.dir.Initialize(); err != nil {
		return err
	}

	return nil
}

// CreateConfigLoader creates a config loader
func (mf *ManagerFactory) CreateConfigLoader() *config.Loader {
	return config.NewLoader(mf.workDir)
}

// CreateJournalLogger creates a journal logger
func (mf *ManagerFactory) CreateJournalLogger() (*journal.Logger, error) {
	paths := mf.dir.GetPaths()
	return journal.NewLogger(paths.Journal)
}

// CreateStateTracker creates a state tracker
func (mf *ManagerFactory) CreateStateTracker() (*state.Tracker, error) {
	paths := mf.dir.GetPaths()
	return state.NewTracker(paths.State)
}

// Status returns the current directory structure status
func (mf *ManagerFactory) Status() (string, error) {
	paths := mf.dir.GetPaths()

	// Check if root exists
	if _, err := os.Stat(paths.Root); err != nil {
		return "not_initialized", nil
	}

	// Check subdirectories
	status := "initialized\n"
	status += fmt.Sprintf("  Config: %s\n", paths.Config)
	status += fmt.Sprintf("  State: %s\n", paths.State)
	status += fmt.Sprintf("  Journal: %s\n", paths.Journal)
	status += fmt.Sprintf("  Backups: %s\n", paths.Backups)

	// Check if config exists
	if _, err := os.Stat(paths.Config); err == nil {
		status += "  Config loaded: yes\n"
	} else {
		status += "  Config loaded: no\n"
	}

	return status, nil
}
