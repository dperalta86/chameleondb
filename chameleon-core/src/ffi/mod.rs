use std::ffi::{CStr, CString};
use std::os::raw::c_char;
use std::ptr;

use crate::parser::parse_schema;
use crate::ast::Schema;
use crate::ChameleonError;
use std::sync::Mutex;
use lazy_static::lazy_static;
use serde_json::Value;

lazy_static! {
    static ref CACHED_SCHEMA: Mutex<Option<Schema>> = Mutex::new(None);
}

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
#[no_mangle]
pub unsafe extern "C" fn chameleon_parse_schema(
    input: *const c_char,
    error_out: *mut *mut c_char,
) -> *mut c_char {
    if input.is_null() {
        set_error(error_out, "Input string is null");
        return ptr::null_mut();
    }

    let input_str = match CStr::from_ptr(input).to_str() {
        Ok(s) => s,
        Err(e) => {
            set_error(error_out, &format!("Invalid UTF-8: {}", e));
            return ptr::null_mut();
        }
    };

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

    let json = match serde_json::to_string_pretty(&schema) {
        Ok(j) => j,
        Err(e) => {
            set_error(error_out, &format!("JSON serialization error: {}", e));
            return ptr::null_mut();
        }
    };

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

    match crate::parser::parse_schema(c_str) {
        Ok(schema) => {
            let result = crate::typechecker::type_check(&schema);
            
            if result.errors.is_empty() {
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
#[no_mangle]
pub unsafe extern "C" fn chameleon_free_string(s: *mut c_char) {
    if !s.is_null() {
        drop(CString::from_raw(s));
    }
}

/// Get the version of the library
#[no_mangle]
pub extern "C" fn chameleon_version() -> *const c_char {
    static VERSION: &str = concat!(env!("CARGO_PKG_VERSION"), "\0");
    VERSION.as_ptr() as *const c_char
}

/// Generate SQL from a query JSON + schema JSON
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
            *error_out = ptr;
            ChameleonResult::Ok
        }
        Err(e) => {
            set_error(error_out, &format!("SQL generation error: {}", e));
            ChameleonResult::ValidationError
        }
    }
}

/// Generate migration SQL from a schema JSON
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

    match crate::migration::generator::generate_migration(&schema) {
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

// ============================================================
// NEW: MUTATION SQL GENERATION (v0.1)
// ============================================================

/// Set schema cache for efficient batch operations
/// 
/// Call this once before batch mutations, then pass NULL for schema_json
/// in generate_mutation_sql calls to reuse the cached schema
#[no_mangle]
pub extern "C" fn set_schema_cache(schema_json: *const c_char) -> *const c_char {
    let schema_str = unsafe {
        CStr::from_ptr(schema_json)
            .to_str()
            .unwrap_or("")
            .to_string()
    };

    let schema: Schema = match serde_json::from_str(&schema_str) {
        Ok(s) => s,
        Err(e) => {
            let error_json = serde_json::json!({
                "valid": false,
                "error": format!("Failed to parse schema: {}", e)
            });
            return CString::new(error_json.to_string()).unwrap().into_raw();
        }
    };

    let mut cache = CACHED_SCHEMA.lock().unwrap();
    *cache = Some(schema);

    let result = serde_json::json!({"valid": true});
    CString::new(result.to_string()).unwrap().into_raw()
}

/// Clear the schema cache
/// Call this after batch operations to free memory
#[no_mangle]
pub extern "C" fn clear_schema_cache() -> *const c_char {
    let mut cache = CACHED_SCHEMA.lock().unwrap();
    *cache = None;

    let result = serde_json::json!({"valid": true});
    CString::new(result.to_string()).unwrap().into_raw()
}

/// Generate SQL for a mutation operation
/// 
/// # Arguments
/// * `mutation_json` - Mutation spec: {"type":"insert|update|delete","entity":"Entity","fields":{...},"filters":{...}}
/// * `schema_json` - Schema JSON (pass NULL to use cached schema from set_schema_cache)
/// 
/// # Returns
/// JSON: {"valid":true,"sql":"...","params":[...]} or {"valid":false,"error":"..."}
#[no_mangle]
pub extern "C" fn generate_mutation_sql(
    mutation_json: *const c_char,
    schema_json: *const c_char,
) -> *const c_char {
    let mutation_str = unsafe {
        CStr::from_ptr(mutation_json)
            .to_str()
            .unwrap_or("")
            .to_string()
    };

    let mutation_value: Value = match serde_json::from_str(&mutation_str) {
        Ok(v) => v,
        Err(e) => {
            let error_json = serde_json::json!({
                "valid": false,
                "error": format!("Invalid mutation JSON: {}", e)
            });
            return CString::new(error_json.to_string()).unwrap().into_raw();
        }
    };

    let schema = if schema_json.is_null() {
        let cache = CACHED_SCHEMA.lock().unwrap();
        match cache.clone() {
            Some(s) => s,
            None => {
                let error_json = serde_json::json!({
                    "valid": false,
                    "error": "No schema provided and cache is empty. Call set_schema_cache() first or pass schema_json."
                });
                return CString::new(error_json.to_string()).unwrap().into_raw();
            }
        }
    } else {
        let schema_str = unsafe {
            CStr::from_ptr(schema_json)
                .to_str()
                .unwrap_or("")
                .to_string()
        };

        let schema: Schema = match serde_json::from_str(&schema_str) {
            Ok(s) => s,
            Err(e) => {
                let error_json = serde_json::json!({
                    "valid": false,
                    "error": format!("Invalid schema JSON: {}", e)
                });
                return CString::new(error_json.to_string()).unwrap().into_raw();
            }
        };

        let mut cache = CACHED_SCHEMA.lock().unwrap();
        *cache = Some(schema.clone());

        schema
    };

    let result = crate::mutation::generate_mutation_sql(&mutation_value, &schema);
    CString::new(result.to_string()).unwrap().into_raw()
}

// ============================================================
// HELPER FUNCTION
// ============================================================

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
        let input = CString::new(
            r#"
            entity User {
                id: uuid primary,
                email: string,
            }
        "#,
        )
        .unwrap();

        let mut error: *mut c_char = ptr::null_mut();

        unsafe {
            let result = chameleon_parse_schema(input.as_ptr(), &mut error);

            assert!(!result.is_null(), "Parse should succeed");
            assert!(error.is_null(), "No error should be set");

            let json_str = CStr::from_ptr(result).to_str().unwrap();
            assert!(json_str.contains("User"));
            assert!(json_str.contains("email"));

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
            assert!(error_msg.contains("kind") || error_msg.contains("message"));

            chameleon_free_string(error);
        }
    }

    #[test]
    fn test_validate_schema_success() {
        let input = CString::new(
            r#"
            entity User {
                id: uuid primary,
                email: string,
            }
        "#,
        )
        .unwrap();

        let mut error: *mut c_char = ptr::null_mut();

        unsafe {
            let result = chameleon_validate_schema(input.as_ptr(), &mut error);

            assert_eq!(result, ChameleonResult::Ok, "Validation should succeed");
            
            if !error.is_null() {
                let json_str = CStr::from_ptr(error).to_str().unwrap();
                assert!(json_str.contains("\"valid\":true"));
                chameleon_free_string(error);
            }
        }
    }

    #[test]
    fn test_validate_schema_with_duplicates() {
        let input = CString::new(
            r#"
            entity User {
                id: uuid primary,
                email: string,
            }
            entity User {
                id: uuid primary,
                name: string,
            }
        "#,
        )
        .unwrap();

        let mut error: *mut c_char = ptr::null_mut();

        unsafe {
            let result = chameleon_validate_schema(input.as_ptr(), &mut error);

            assert_eq!(result, ChameleonResult::ValidationError, "Should detect duplicate");
            assert!(!error.is_null(), "Error message should be set");

            let json_str = CStr::from_ptr(error).to_str().unwrap();
            assert!(json_str.contains("\"valid\":false"));
            assert!(json_str.contains("Duplicate entity"));

            chameleon_free_string(error);
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
        }
    }
}