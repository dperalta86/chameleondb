use serde::{Deserialize, Serialize};
use super::filter::FilterExpr;

/// Sort direction
#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub enum SortDirection {
    Asc,
    Desc,
}

/// A single order-by clause
#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct OrderByClause {
    pub field: String,
    pub direction: SortDirection,
}

/// An include path for eager loading
/// "orders" → load orders
/// "orders.items" → load orders AND their items
#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct IncludePath {
    pub path: Vec<String>,
}

impl IncludePath {
    pub fn parse(path: &str) -> Self {
        IncludePath {
            path: path.split('.').map(|s| s.to_string()).collect(),
        }
    }
}

/// The complete query representation
/// This is what gets serialized over FFI and translated to SQL
#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct Query {
    /// Target entity name (e.g., "User")
    pub entity: String,

    /// Filter conditions (combined with AND by default)
    pub filters: Vec<FilterExpr>,

    /// Relations to eager-load
    pub includes: Vec<IncludePath>,

    /// Order by clauses
    pub order_by: Vec<OrderByClause>,

    /// Maximum number of results
    pub limit: Option<u64>,

    /// Number of results to skip
    pub offset: Option<u64>,
}

impl Query {
    /// Create a new query targeting an entity
    pub fn new(entity: &str) -> Self {
        Query {
            entity: entity.to_string(),
            filters: Vec::new(),
            includes: Vec::new(),
            order_by: Vec::new(),
            limit: None,
            offset: None,
        }
    }

    /// Add a filter condition
    pub fn filter(mut self, expr: FilterExpr) -> Self {
        self.filters.push(expr);
        self
    }

    /// Add an include path
    pub fn include(mut self, path: &str) -> Self {
        self.includes.push(IncludePath::parse(path));
        self
    }

    /// Add an order-by clause
    pub fn order_by(mut self, field: &str, direction: SortDirection) -> Self {
        self.order_by.push(OrderByClause {
            field: field.to_string(),
            direction,
        });
        self
    }

    /// Set limit
    pub fn limit(mut self, n: u64) -> Self {
        self.limit = Some(n);
        self
    }

    /// Set offset
    pub fn offset(mut self, n: u64) -> Self {
        self.offset = Some(n);
        self
    }
}