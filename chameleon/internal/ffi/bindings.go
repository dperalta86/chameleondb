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
*/
import "C"
import (
	"errors"
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

// Version returns the Rust core library version
func Version() string {
	cVersion := C.chameleon_version()
	return C.GoString(cVersion)
}
