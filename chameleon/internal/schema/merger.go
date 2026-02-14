package schema

import (
	"fmt"
	"regexp"
	"strings"
)

// Merger es la interfaz para diferentes estrategias de merge
type Merger interface {
	// Merge combina múltiples contenidos de schema
	Merge(filenames []string, contents []string) (string, error)
	// Validate verifica que no haya conflictos (v0.2)
	Validate(merged string) error
}

// MergedSchemaResult contiene el schema merged + metadata de origen
type MergedSchemaResult struct {
	Content  string             // Schema merged
	LineMap  map[int]SourceLine // merged_line → source_location
	SavePath string             // Donde se guarda merged (para debugging)
}

// SourceLine rastrea dónde viene una línea del schema merged
type SourceLine struct {
	File       string // Nombre del archivo origen
	LineNumber int    // Número de línea en archivo origen
}

// SimpleMerger implementa merge básico para v0.1 con source tracking
type SimpleMerger struct{}

// Merge concatena múltiples archivos de schema con source line tracking
func (m *SimpleMerger) Merge(filenames []string, contents []string) (*MergedSchemaResult, error) {
	if len(filenames) != len(contents) {
		return nil, fmt.Errorf("filenames and contents length mismatch")
	}

	if len(filenames) == 0 {
		return nil, fmt.Errorf("no schema files to merge")
	}

	var merged strings.Builder
	lineMap := make(map[int]SourceLine)
	currentMergedLine := 1

	// Escribir cada archivo con comentario de origen
	for i, filename := range filenames {
		merged.WriteString("// ==========================================\n")
		currentMergedLine++
		merged.WriteString("// From: " + filename + "\n")
		currentMergedLine++
		merged.WriteString("// ==========================================\n")
		currentMergedLine++

		// Split content by lines y rastrear origen
		lines := strings.Split(contents[i], "\n")
		for lineIdx, line := range lines {
			if line == "" && lineIdx == len(lines)-1 {
				// Skip last empty line if it's from split
				continue
			}

			// Mapear línea merged a línea origen
			lineMap[currentMergedLine] = SourceLine{
				File:       filename,
				LineNumber: lineIdx + 1, // Line numbers start at 1
			}

			merged.WriteString(line)
			if lineIdx < len(lines)-1 {
				merged.WriteString("\n")
			}
			currentMergedLine++
		}

		merged.WriteString("\n\n")
		currentMergedLine += 2
	}

	return &MergedSchemaResult{
		Content: merged.String(),
		LineMap: lineMap,
	}, nil
}

// Validate valida que no haya conflictos en el schema merged
func (m *SimpleMerger) Validate(merged string) error {
	entityPattern := `entity\s+([A-Za-z_][A-Za-z0-9_]*)\s*\{`
	re := regexp.MustCompile(entityPattern)

	matches := re.FindAllStringSubmatch(merged, -1)
	if matches == nil {
		return fmt.Errorf("no entities found in schema")
	}

	// Track entity names
	entityCount := make(map[string]int)

	for _, match := range matches {
		if len(match) > 1 {
			entityName := match[1]
			entityCount[entityName]++
		}
	}

	// Check for duplicates
	var duplicates []string
	for entityName, count := range entityCount {
		if count > 1 {
			duplicates = append(duplicates, fmt.Sprintf("%s (appears %d times)", entityName, count))
		}
	}

	if len(duplicates) > 0 {
		return fmt.Errorf("duplicate entities found: %s\n\nEntity names must be unique across all schema files. "+
			"Define each entity only once.", strings.Join(duplicates, ", "))
	}

	return nil
}

// NewSimpleMerger crea un nuevo SimpleMerger
func NewSimpleMerger() *SimpleMerger {
	return &SimpleMerger{}
}
