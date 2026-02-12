package ffi

/*
#include <stdlib.h>

// Forward declarations
typedef enum {
    CHAMELEON_OK = 0,
    CHAMELEON_PARSE_ERROR = 1,
    CHAMELEON_VALIDATION_ERROR = 2,
    CHAMELEON_INTERNAL_ERROR = 3,
} ChameleonResult;

char* chameleon_parse_schema(const char* input, char** error_out);
ChameleonResult chameleon_validate_schema(const char* schema_json, char** error_out);
void chameleon_free_string(char* s);
const char* chameleon_version(void);
extern int chameleon_generate_sql(const char* query_json, const char* schema_json, char** error_out);
extern int chameleon_generate_migration(const char* schema_json, char** error_out);
*/
import "C"
import (
	"errors"
	"fmt"
	"os"
	"unsafe"
)

// Result codes
const (
	ResultOk              = C.CHAMELEON_OK
	ResultParseError      = C.CHAMELEON_PARSE_ERROR
	ResultValidationError = C.CHAMELEON_VALIDATION_ERROR
	ResultInternalError   = C.CHAMELEON_INTERNAL_ERROR
)

// ParseSchema calls Rust FFI to parse a schema string into JSON
func ParseSchema(input string) (string, error) {
	cInput := C.CString(input)
	defer C.free(unsafe.Pointer(cInput))

	var cError *C.char
	defer func() {
		if cError != nil {
			C.chameleon_free_string(cError)
		}
	}()

	cResult := C.chameleon_parse_schema(cInput, &cError)

	if cResult == nil {
		if cError != nil {
			return "", errors.New(C.GoString(cError))
		}
		return "", errors.New("unknown parse error")
	}

	defer C.chameleon_free_string(cResult)
	return C.GoString(cResult), nil
}

// ValidateSchema calls Rust FFI to validate a schema JSON
func ValidateSchema(schemaJSON string) error {
	cJSON := C.CString(schemaJSON)
	defer C.free(unsafe.Pointer(cJSON))

	var cError *C.char
	defer func() {
		if cError != nil {
			C.chameleon_free_string(cError)
		}
	}()

	result := C.chameleon_validate_schema(cJSON, &cError)

	if result != ResultOk {
		if cError != nil {
			return errors.New(C.GoString(cError))
		}
		return errors.New("validation failed")
	}

	return nil
}

// ValidateSchemaRaw validates schema and returns structured JSON errors
func ValidateSchemaRaw(schemaInput string) (string, error) {
	cInput := C.CString(schemaInput)
	defer C.free(unsafe.Pointer(cInput))
	var cError *C.char
	defer func() {
		if cError != nil {
			C.chameleon_free_string(cError)
		}
	}()

	result := C.chameleon_validate_schema(cInput, &cError)

	if cError != nil {
		errMsg := C.GoString(cError)
		if os.Getenv("DEBUG") == "1" {
			fmt.Fprintf(os.Stderr, "DEBUG ValidateSchemaRaw: result=%d, errMsg=%q\n", result, errMsg)
		}
		if result == ResultOk {
			return errMsg, nil
		}
		return errMsg, errors.New("validation failed")
	}

	if os.Getenv("DEBUG") == "1" {
		fmt.Fprintf(os.Stderr, "DEBUG ValidateSchemaRaw: cError is nil, result=%d\n", result)
	}
	return "", errors.New("unknown error")
}

// Version returns the Rust core library version
func Version() string {
	cVersion := C.chameleon_version()
	return C.GoString(cVersion)
}

// GenerateSQL calls the Rust SQL generator
// Takes query JSON and schema JSON, returns GeneratedSQL JSON
func GenerateSQL(queryJSON string, schemaJSON string) (string, error) {
	cQuery := C.CString(queryJSON)
	defer C.free(unsafe.Pointer(cQuery))

	cSchema := C.CString(schemaJSON)
	defer C.free(unsafe.Pointer(cSchema))

	var errorOut *C.char

	result := C.chameleon_generate_sql(cQuery, cSchema, &errorOut)

	if result != 0 {
		if errorOut != nil {
			errMsg := C.GoString(errorOut)
			C.chameleon_free_string(errorOut)
			return "", fmt.Errorf("%s", errMsg)
		}
		return "", fmt.Errorf("SQL generation failed with code %d", result)
	}

	// On success, error_out contains the result JSON
	if errorOut == nil {
		return "", fmt.Errorf("SQL generation returned null")
	}

	output := C.GoString(errorOut)
	C.chameleon_free_string(errorOut)
	return output, nil
}

// GenerateMigration calls the Rust migration generator
func GenerateMigration(schemaJSON string) (string, error) {
	cSchema := C.CString(schemaJSON)
	defer C.free(unsafe.Pointer(cSchema))

	var errorOut *C.char

	result := C.chameleon_generate_migration(cSchema, &errorOut)

	if result != 0 {
		if errorOut != nil {
			errMsg := C.GoString(errorOut)
			C.chameleon_free_string(errorOut)
			return "", fmt.Errorf("%s", errMsg)
		}
		return "", fmt.Errorf("migration generation failed with code %d", result)
	}

	if errorOut == nil {
		return "", fmt.Errorf("migration generation returned null")
	}

	output := C.GoString(errorOut)
	C.chameleon_free_string(errorOut)
	return output, nil
}

// GenerateMutationSQL calls Rust FFI to generate mutation SQL
// If schemaJSON is "", uses cached schema from previous SetSchemaCache call
func GenerateMutationSQL(mutationJSON string, schemaJSON string) string {
	// C code would call: generate_mutation_sql(mutation_json, schema_json)
	// Returns JSON string
	// Implementation depends on your C bindings
	// For now, placeholder:
	return `{"valid": false, "error": "FFI binding not yet implemented"}`
}

// SetSchemaCache calls Rust FFI to cache schema
func SetSchemaCache(schemaJSON string) string {
	// C code would call: set_schema_cache(schema_json)
	return `{"valid": false, "error": "FFI binding not yet implemented"}`
}

// ClearSchemaCache calls Rust FFI to clear cache
func ClearSchemaCache() string {
	// C code would call: clear_schema_cache()
	return `{"valid": true}`
}
