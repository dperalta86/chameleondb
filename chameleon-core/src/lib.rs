pub mod ast;
pub mod parser;
pub mod error;
pub mod ffi;

pub use ast::*;
pub use parser::parse_schema;
pub use error::ChameleonError;

// Re-export FFI functions for external use
pub use ffi::{
    chameleon_parse_schema,
    chameleon_validate_schema,
    chameleon_free_string,
    chameleon_version,
    ChameleonResult,
};