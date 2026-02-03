use serde::{Deserialize, Serialize};
use std::fmt;

/// Detailed parse error with position and context
#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct ParseErrorDetail {
    /// Human-readable error message
    pub message: String,
    /// Line number (1-indexed)
    pub line: usize,
    /// Column number (1-indexed)
    pub column: usize,
    /// Optional code snippet showing the error location
    pub snippet: Option<String>,
    /// Optional suggestion for fixing the error
    pub suggestion: Option<String>,
    /// The problematic token/text if available
    pub token: Option<String>,
}

impl ParseErrorDetail {
    pub fn new(message: String, line: usize, column: usize) -> Self {
        ParseErrorDetail {
            message,
            line,
            column,
            snippet: None,
            suggestion: None,
            token: None,
        }
    }

    pub fn with_snippet(mut self, snippet: String) -> Self {
        self.snippet = Some(snippet);
        self
    }

    pub fn with_suggestion(mut self, suggestion: String) -> Self {
        self.suggestion = Some(suggestion);
        self
    }

    pub fn with_token(mut self, token: String) -> Self {
        self.token = Some(token);
        self
    }
}

/// Main error type for ChameleonDB
#[derive(Debug, Clone, PartialEq)]
pub enum ChameleonError {
    /// Parse error with detailed position information
    ParseError(ParseErrorDetail),
    /// Validation error (type checking, etc)
    ValidationError(String),
    /// Internal error (should not happen in normal use)
    InternalError(String),
}

impl fmt::Display for ChameleonError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            ChameleonError::ParseError(detail) => {
                write!(f, "Parse error at line {}, column {}: {}", 
                    detail.line, detail.column, detail.message)
            }
            ChameleonError::ValidationError(msg) => write!(f, "Validation error: {}", msg),
            ChameleonError::InternalError(msg) => write!(f, "Internal error: {}", msg),
        }
    }
}

impl std::error::Error for ChameleonError {}

/// Convert from LALRPOP parse errors
impl From<lalrpop_util::ParseError<usize, lalrpop_util::lexer::Token<'_>, &str>> for ChameleonError {
    fn from(err: lalrpop_util::ParseError<usize, lalrpop_util::lexer::Token, &str>) -> Self {
        use lalrpop_util::ParseError;
        
        match err {
            ParseError::InvalidToken { location } => {
                ChameleonError::ParseError(
                    ParseErrorDetail::new(
                        "Invalid token".to_string(),
                        1, // We'll improve this
                        location + 1,
                    )
                )
            }
            ParseError::UnrecognizedEof { location, expected } => {
                let message = if expected.is_empty() {
                    "Unexpected end of file".to_string()
                } else {
                    format!("Unexpected end of file, expected one of: {}", expected.join(", "))
                };
                ChameleonError::ParseError(
                    ParseErrorDetail::new(message, 1, location + 1)
                        .with_suggestion("Check if you're missing a closing brace }".to_string())
                )
            }
            ParseError::UnrecognizedToken { token, expected } => {
                let (start, tok, _end) = token;
                let token_str = format!("{:?}", tok);
                
                let message = if expected.is_empty() {
                    format!("Unexpected token: {}", token_str)
                } else {
                    format!("Unexpected token: {}, expected one of: {}", 
                        token_str, expected.join(", "))
                };
                
                ChameleonError::ParseError(
                    ParseErrorDetail::new(message, 1, start + 1)
                        .with_token(token_str)
                )
            }
            ParseError::ExtraToken { token } => {
                let (start, tok, _end) = token;
                ChameleonError::ParseError(
                    ParseErrorDetail::new(
                        format!("Extra unexpected token: {:?}", tok),
                        1,
                        start + 1,
                    )
                )
            }
            ParseError::User { error } => {
                ChameleonError::InternalError(error.to_string())
            }
        }
    }
}