package schema

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// Loader es la interfaz para cargar schemas
type Loader interface {
	// LoadAll carga todos los archivos de schema disponibles
	LoadAll() (filenames []string, contents []string, err error)
	// Load carga un archivo específico
	Load(filepath string) (string, error)
}

// FileLoader carga schemas desde archivos del filesystem
type FileLoader struct {
	schemaPaths []string // Rutas donde buscar .cham files
}

// NewFileLoader crea un nuevo FileLoader
func NewFileLoader(schemaPaths []string) *FileLoader {
	return &FileLoader{
		schemaPaths: schemaPaths,
	}
}

// LoadAll carga todos los archivos .cham de los schema paths
func (fl *FileLoader) LoadAll() ([]string, []string, error) {
	var filenames []string
	var contents []string

	// Buscar en todos los schema paths
	for _, schemaPath := range fl.schemaPaths {
		files, err := fl.findSchemaFiles(schemaPath)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to find schema files in %s: %w", schemaPath, err)
		}

		// Cargar cada archivo
		for _, file := range files {
			content, err := fl.Load(file)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to load %s: %w", file, err)
			}

			// Agregar al resultado (solo basename para legibilidad)
			filenames = append(filenames, filepath.Base(file))
			contents = append(contents, content)
		}
	}

	if len(filenames) == 0 {
		return nil, nil, fmt.Errorf("no schema files found in %v", fl.schemaPaths)
	}

	// Ordenar alfabéticamente para consistencia
	sort.Strings(filenames)
	// TODO: Reordenar contents según filenames ordenados

	return filenames, contents, nil
}

// Load carga un archivo específico
func (fl *FileLoader) Load(filepath string) (string, error) {
	content, err := os.ReadFile(filepath)
	if err != nil {
		return "", fmt.Errorf("failed to read schema file: %w", err)
	}
	return string(content), nil
}

// findSchemaFiles encuentra todos los .cham files en un directorio
func (fl *FileLoader) findSchemaFiles(dirPath string) ([]string, error) {
	// Resolver path absoluto
	if !filepath.IsAbs(dirPath) {
		wd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		dirPath = filepath.Join(wd, dirPath)
	}

	var files []string

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".cham" {
			files = append(files, filepath.Join(dirPath, entry.Name()))
		}
	}

	// Ordenar para consistencia
	sort.Strings(files)

	return files, nil
}
