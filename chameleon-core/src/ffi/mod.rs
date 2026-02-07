use std::ffi::{CStr, CString};
use std::os::raw::c_char;
use std::ptr;

use crate::parser::parse_schema;
use crate::ast::Schema;
use crate::ChameleonError;

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
            let json = serde_json::to_string(&e)
                .unwrap_or_else(|_| {
                    r#"{"kind":"InternalError","data":{"message":"Failed to serialize error"}}"#.to_string()
                });

        set_error(error_out, &json);
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

use serde::Serialize;

/// Structured validation error for JSON output
#[derive(Serialize)]
struct ValidationErrorJson {
    kind: String,
    message: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    line: Option<usize>,
    #[serde(skip_serializing_if = "Option::is_none")]
    column: Option<usize>,
    #[serde(skip_serializing_if = "Option::is_none")]
    snippet: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    suggestion: Option<String>,
}

/// Structured validation result for JSON output
#[derive(Serialize)]
struct ValidationResultJson {
    valid: bool,
    errors: Vec<ValidationErrorJson>,
}

/// Validate a schema (checks relations, constraints, etc.)
/// Returns JSON with structured errors
/// 
/// # Safety
/// - `input` must be a valid null-terminated C string containing schema DSL
/// - Caller must free the returned string with `chameleon_free_string`
#[no_mangle]
pub unsafe extern "C" fn chameleon_validate_schema(
    input: *const c_char,
    error_out: *mut *mut c_char,
) -> ChameleonResult {
    if input.is_null() {
        set_error(error_out, "Input is null");
        return ChameleonResult::InternalError;
    }

    let c_str = match CStr::from_ptr(input).to_str() {
        Ok(s) => s,
        Err(e) => {
            set_error(error_out, &format!("Invalid UTF-8: {}", e));
            return ChameleonResult::InternalError;
        }
    };

    let mut validation_errors: Vec<ValidationErrorJson> = Vec::new();

    // Parse schema
    match crate::parser::parse_schema(c_str) {
        Ok(schema) => {
            // Type check the schema
            let result = crate::typechecker::type_check(&schema);
            
            if result.errors.is_empty() {
                // Success - return valid JSON
                let result_json = ValidationResultJson {
                    valid: true,
                    errors: vec![],
                };
                
                let json = serde_json::to_string(&result_json)
                    .unwrap_or_else(|_| r#"{"valid":true,"errors":[]}"#.to_string());
                
                match CString::new(json) {
                    Ok(c_str) => {
                        *error_out = c_str.into_raw();
                        return ChameleonResult::Ok;
                    }
                    Err(_) => {
                        set_error(error_out, "Failed to convert JSON to C string");
                        return ChameleonResult::InternalError;
                    }
                }
            } else {
                // Validation errors - build structured response
                for err in result.errors {
                    validation_errors.push(ValidationErrorJson {
                        kind: "ValidationError".to_string(),
                        message: err.to_string(),
                        line: None,
                        column: None,
                        snippet: None,
                        suggestion: None,
                    });
                }
                
                let result_json = ValidationResultJson {
                    valid: false,
                    errors: validation_errors,
                };
                
                let json = serde_json::to_string(&result_json)
                    .unwrap_or_else(|_| r#"{"valid":false,"errors":[]}"#.to_string());
                
                match CString::new(json) {
                    Ok(c_str) => {
                        *error_out = c_str.into_raw();
                        return ChameleonResult::ValidationError;
                    }
                    Err(_) => {
                        set_error(error_out, "Failed to convert JSON to C string");
                        return ChameleonResult::InternalError;
                    }
                }
            }
        }
        Err(e) => {
            // Parse error - extract details if available
            match &e {
                ChameleonError::ParseError(detail) => {
                    validation_errors.push(ValidationErrorJson {
                        kind: "ParseError".to_string(),
                        message: detail.message.clone(),
                        line: Some(detail.line),
                        column: Some(detail.column),
                        snippet: detail.snippet.clone(),
                        suggestion: detail.suggestion.clone(),
                    });
                }
                _ => {
                    validation_errors.push(ValidationErrorJson {
                        kind: "ParseError".to_string(),
                        message: e.to_string(),
                        line: None,
                        column: None,
                        snippet: None,
                        suggestion: None,
                    });
                }
            }
            
            let result_json = ValidationResultJson {
                valid: false,
                errors: validation_errors,
            };
            
            let json = serde_json::to_string(&result_json)
                .unwrap_or_else(|_| r#"{"valid":false,"errors":[]}"#.to_string());
            
            match CString::new(json) {
                Ok(c_str) => {
                    *error_out = c_str.into_raw();
                    return ChameleonResult::ParseError;
                }
                Err(_) => {
                    set_error(error_out, "Failed to convert JSON to C string");
                    return ChameleonResult::InternalError;
                }
            }
        }
    }
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

/// Generate SQL from a query JSON + schema JSON
/// 
/// Input:  query_json  - serialized Query
///         schema_json - serialized Schema  
/// Output: returns JSON-serialized GeneratedSQL
///         error_out   - error message on failure
#[no_mangle]
pub unsafe extern "C" fn chameleon_generate_sql(
    query_json: *const c_char,
    schema_json: *const c_char,
    error_out: *mut *mut c_char,
) -> ChameleonResult {
    if query_json.is_null() || schema_json.is_null() {
        set_error(error_out, "Query or schema JSON is null");
        return ChameleonResult::InternalError;
    }

    let query_str = match CStr::from_ptr(query_json).to_str() {
        Ok(s) => s,
        Err(e) => {
            set_error(error_out, &format!("Invalid query JSON UTF-8: {}", e));
            return ChameleonResult::InternalError;
        }
    };

    let schema_str = match CStr::from_ptr(schema_json).to_str() {
        Ok(s) => s,
        Err(e) => {
            set_error(error_out, &format!("Invalid schema JSON UTF-8: {}", e));
            return ChameleonResult::InternalError;
        }
    };

    let query: crate::query::Query = match serde_json::from_str(query_str) {
        Ok(q) => q,
        Err(e) => {
            set_error(error_out, &format!("Query deserialization error: {}", e));
            return ChameleonResult::InternalError;
        }
    };

    let schema: Schema = match serde_json::from_str(schema_str) {
        Ok(s) => s,
        Err(e) => {
            set_error(error_out, &format!("Schema deserialization error: {}", e));
            return ChameleonResult::InternalError;
        }
    };

    match crate::sql::generate_sql(&query, &schema) {
        Ok(generated) => {
            let json = serde_json::to_string(&generated).unwrap();
            let c_str = CString::new(json).unwrap();
            let ptr = c_str.into_raw();
            // We need to return the pointer, reuse error_out as output channel
            *error_out = ptr;
            ChameleonResult::Ok
        }
        Err(e) => {
            set_error(error_out, &format!("SQL generation error: {}", e));
            return ChameleonResult::ValidationError;
        }
    }
}

/// Generate migration SQL from a schema JSON
///
/// Input:  schema_json - serialized Schema
/// Output: returns the DDL SQL string directly
///         error_out   - error message on failure
#[no_mangle]
pub unsafe extern "C" fn chameleon_generate_migration(
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
            set_error(error_out, &format!("Schema deserialization error: {}", e));
            return ChameleonResult::InternalError;
        }
    };

    match crate::migration::generate_migration(&schema) {
        Ok(migration) => {
            let c_str = CString::new(migration.sql).unwrap();
            let ptr = c_str.into_raw();
            *error_out = ptr;
            ChameleonResult::Ok
        }
        Err(e) => {
            set_error(error_out, &format!("Migration generation error: {}", e));
            ChameleonResult::ValidationError
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