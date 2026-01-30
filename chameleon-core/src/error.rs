use thiserror::Error;

#[derive(Error, Debug)]
pub enum CharmeleonError {
    #[error("Parse error: {0}")]
    ParseError(String),
    
    #[error("Type error: {0}")]
    TypeError(String),
    
    #[error("Unknown entity: {0}")]
    UnknownEntity(String),
    
    #[error("Unknown field: {0}")]
    UnknownField(String),
}