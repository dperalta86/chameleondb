use serde::{Deserialize, Serialize};

/// Represents a single value that can be used in a filter
#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub enum FilterValue {
    String(String),
    Int(i64),
    Float(f64),
    Bool(bool),
    Null,
}

/// Comparison operators
#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub enum ComparisonOp {
    Eq,      // ==
    Neq,     // !=
    Gt,      // >
    Gte,     // >=
    Lt,      // 
    Lte,     // <=
    Like,    // LIKE '%value%'
    In,      // IN (v1, v2, v3)
}

/// Logical operators to combine filters
#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub enum LogicalOp {
    And,
    Or,
}

/// A field path supports nested access: "orders.total"
/// Each segment is one level of navigation
#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct FieldPath {
    pub segments: Vec<String>,
}

impl FieldPath {
    /// Create a field path from a dot-separated string
    /// "orders.total" → ["orders", "total"]
    pub fn parse(path: &str) -> Self {
        FieldPath {
            segments: path.split('.').map(|s| s.to_string()).collect(),
        }
    }

    /// Returns true if this path crosses entity boundaries
    /// "email" → false (single entity)
    /// "orders.total" → true (User → Order)
    pub fn is_nested(&self) -> bool {
        self.segments.len() > 1
    }

    /// The root field name (first segment)
    pub fn root(&self) -> &str {
        &self.segments[0]
    }
}

/// A single filter condition
#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct FilterCondition {
    pub field: FieldPath,
    pub op: ComparisonOp,
    pub value: FilterValue,
}

/// A filter expression tree
/// Supports combining conditions with AND/OR
#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub enum FilterExpr {
    /// A single condition: field op value
    Condition(FilterCondition),
    /// Combines two expressions with AND/OR
    Binary {
        left: Box<FilterExpr>,
        op: LogicalOp,
        right: Box<FilterExpr>,
    },
}

impl FilterExpr {
    /// Create a simple condition
    pub fn condition(field: &str, op: ComparisonOp, value: FilterValue) -> Self {
        FilterExpr::Condition(FilterCondition {
            field: FieldPath::parse(field),
            op,
            value,
        })
    }

    /// Combine with AND
    pub fn and(self, other: FilterExpr) -> Self {
        FilterExpr::Binary {
            left: Box::new(self),
            op: LogicalOp::And,
            right: Box::new(other),
        }
    }

    /// Combine with OR
    pub fn or(self, other: FilterExpr) -> Self {
        FilterExpr::Binary {
            left: Box::new(self),
            op: LogicalOp::Or,
            right: Box::new(other),
        }
    }
}