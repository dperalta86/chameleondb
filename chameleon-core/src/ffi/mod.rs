use std::ffi::{CStr, CString};
use std::os::raw::c_char;
use std::ptr;

use crate::parser::parse_schema;
use crate::ast::Schema;

/// Result code for FFI functions
#[repr(C)]
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ChameleonResult {
    Ok = 0,
    ParseError = 1,
    ValidationError = 2,
    InternalError = 3,
}

/// Parse a schema from a string and return JSON representation
/// 
/// # Safety
/// - `input` must be a valid null-terminated C string
/// - Caller must free the returned string with `chameleon_free_string`
/// - Returns NULL on error, check `error_out` for details
/// 
/// # Example (from C/Go)
/// ```c
/// char* error = NULL;
/// char* json = chameleon_parse_schema("entity User { id: uuid primary, }", &error);
/// if (json) {
///     printf("%s\n", json);
///     chameleon_free_string(json);
/// } else {
///     printf("Error: %s\n", error);
///     chameleon_free_string(error);
/// }
/// ```
#[no_mangle]
pub unsafe extern "C" fn chameleon_parse_schema(
    input: *const c_char,
    error_out: *mut *mut c_char,
) -> *mut c_char {
    // Validate input pointer
    if input.is_null() {
        set_error(error_out, "Input string is null");
        return ptr::null_mut();
    }

    // Convert C string to Rust &str
    let input_str = match CStr::from_ptr(input).to_str() {
        Ok(s) => s,
        Err(e) => {
            set_error(error_out, &format!("Invalid UTF-8: {}", e));
            return ptr::null_mut();
        }
    };

    // Parse schema
    let schema = match parse_schema(input_str) {
        Ok(s) => s,
        Err(e) => {
            set_error(error_out, &format!("Parse error: {}", e));
            return ptr::null_mut();
        }
    };

    // Serialize to JSON
    let json = match serde_json::to_string_pretty(&schema) {
        Ok(j) => j,
        Err(e) => {
            set_error(error_out, &format!("JSON serialization error: {}", e));
            return ptr::null_mut();
        }
    };

    // Convert to C string
    match CString::new(json) {
        Ok(c_str) => c_str.into_raw(),
        Err(e) => {
            set_error(error_out, &format!("CString conversion error: {}", e));
            ptr::null_mut()
        }
    }
}

/// Validate a schema (checks relations, constraints, etc.)
/// 
/// # Safety
/// - `schema_json` must be a valid null-terminated C string containing JSON
/// - Returns ChameleonResult::Ok on success
#[no_mangle]
pub unsafe extern "C" fn chameleon_validate_schema(
    schema_json: *const c_char,
    error_out: *mut *mut c_char,
) -> ChameleonResult {
    if schema_json.is_null() {
        set_error(error_out, "Schema JSON is null");
        return ChameleonResult::InternalError;
    }

    let json_str = match CStr::from_ptr(schema_json).to_str() {
        Ok(s) => s,
        Err(e) => {
            set_error(error_out, &format!("Invalid UTF-8: {}", e));
            return ChameleonResult::InternalError;
        }
    };

    let schema: Schema = match serde_json::from_str(json_str) {
        Ok(s) => s,
        Err(e) => {
            set_error(error_out, &format!("JSON deserialization error: {}", e));
            return ChameleonResult::InternalError;
        }
    };

    // Basic validation (TODO: implement full type checker)
    if schema.entities.is_empty() {
        set_error(error_out, "Schema has no entities");
        return ChameleonResult::ValidationError;
    }

    // Check for entities with no fields
    for (name, entity) in &schema.entities {
        if entity.fields.is_empty() && entity.relations.is_empty() {
            set_error(error_out, &format!("Entity '{}' has no fields or relations", name));
            return ChameleonResult::ValidationError;
        }
    }

    ChameleonResult::Ok
}

/// Free a string allocated by Rust
/// 
/// # Safety
/// - `s` must be a pointer previously returned by a chameleon_* function
/// - Do not call this twice on the same pointer
/// - Passing NULL is safe (no-op)
#[no_mangle]
pub unsafe extern "C" fn chameleon_free_string(s: *mut c_char) {
    if !s.is_null() {
        drop(CString::from_raw(s));
    }
}

/// Get the version of the library
/// 
/// # Safety
/// Returns a static string, do not free
#[no_mangle]
pub extern "C" fn chameleon_version() -> *const c_char {
    static VERSION: &str = concat!(env!("CARGO_PKG_VERSION"), "\0");
    VERSION.as_ptr() as *const c_char
}

// Helper function to set error message
unsafe fn set_error(error_out: *mut *mut c_char, message: &str) {
    if !error_out.is_null() {
        if let Ok(c_str) = CString::new(message) {
            *error_out = c_str.into_raw();
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::ffi::CString;

    #[test]
    fn test_parse_schema_success() {
        let input = CString::new(r#"
            entity User {
                id: uuid primary,
                email: string,
            }
        "#).unwrap();

        let mut error: *mut c_char = ptr::null_mut();
        
        unsafe {
            let result = chameleon_parse_schema(input.as_ptr(), &mut error);
            
            assert!(!result.is_null(), "Parse should succeed");
            assert!(error.is_null(), "No error should be set");
            
            // Verify JSON output
            let json_str = CStr::from_ptr(result).to_str().unwrap();
            assert!(json_str.contains("User"));
            assert!(json_str.contains("email"));
            
            // Cleanup
            chameleon_free_string(result);
        }
    }

    #[test]
    fn test_parse_error_handling() {
        let input = CString::new("invalid syntax!!!").unwrap();
        let mut error: *mut c_char = ptr::null_mut();
        
        unsafe {
            let result = chameleon_parse_schema(input.as_ptr(), &mut error);
            
            assert!(result.is_null(), "Parse should fail");
            assert!(!error.is_null(), "Error should be set");
            
            let error_msg = CStr::from_ptr(error).to_str().unwrap();
            assert!(error_msg.contains("Parse error"));
            
            // Cleanup
            chameleon_free_string(error);
        }
    }

    #[test]
    fn test_validate_schema() {
        let schema_json = CString::new(r#"
        {
            "entities": {
                "User": {
                    "name": "User",
                    "fields": {
                        "id": {
                            "name": "id",
                            "field_type": "UUID",
                            "nullable": false,
                            "unique": false,
                            "primary_key": true,
                            "default": null
                        }
                    },
                    "relations": {}
                }
            }
        }
        "#).unwrap();

        let mut error: *mut c_char = ptr::null_mut();
        
        unsafe {
            let result = chameleon_validate_schema(schema_json.as_ptr(), &mut error);
            
            assert_eq!(result, ChameleonResult::Ok);
            assert!(error.is_null());
        }
    }

    #[test]
    fn test_version() {
        unsafe {
            let version = CStr::from_ptr(chameleon_version());
            let version_str = version.to_str().unwrap();
            assert_eq!(version_str, env!("CARGO_PKG_VERSION"));
        }
    }

    #[test]
    fn test_free_null_is_safe() {
        unsafe {
            chameleon_free_string(ptr::null_mut());
            // Should not crash
        }
    }
}