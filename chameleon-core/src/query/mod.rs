pub mod ast;
pub mod filter;

pub use ast::{Query, IncludePath, OrderByClause, SortDirection};
pub use filter::{FilterExpr, FilterValue, ComparisonOp, LogicalOp, FieldPath, FilterCondition};

#[cfg(test)]
mod tests {
    use super::*;

    // ─── FIELD PATH ───

    #[test]
    fn test_field_path_simple() {
        let path = FieldPath::parse("email");
        assert_eq!(path.segments, vec!["email"]);
        assert!(!path.is_nested());
        assert_eq!(path.root(), "email");
    }

    #[test]
    fn test_field_path_nested() {
        let path = FieldPath::parse("orders.total");
        assert_eq!(path.segments, vec!["orders", "total"]);
        assert!(path.is_nested());
        assert_eq!(path.root(), "orders");
    }

    #[test]
    fn test_field_path_deep_nested() {
        let path = FieldPath::parse("orders.items.price");
        assert_eq!(path.segments, vec!["orders", "items", "price"]);
        assert!(path.is_nested());
    }

    // ─── FILTER EXPRESSIONS ───

    #[test]
    fn test_simple_filter() {
        let filter = FilterExpr::condition(
            "email",
            ComparisonOp::Eq,
            FilterValue::String("ana@mail.com".to_string()),
        );

        assert_eq!(filter, FilterExpr::Condition(FilterCondition {
            field: FieldPath::parse("email"),
            op: ComparisonOp::Eq,
            value: FilterValue::String("ana@mail.com".to_string()),
        }));
    }

    #[test]
    fn test_nested_filter() {
        let filter = FilterExpr::condition(
            "orders.total",
            ComparisonOp::Gt,
            FilterValue::Int(100),
        );

        if let FilterExpr::Condition(cond) = &filter {
            assert!(cond.field.is_nested());
            assert_eq!(cond.field.root(), "orders");
        } else {
            panic!("Expected Condition");
        }
    }

    #[test]
    fn test_and_filter() {
        let f1 = FilterExpr::condition("age", ComparisonOp::Gte, FilterValue::Int(18));
        let f2 = FilterExpr::condition("age", ComparisonOp::Lte, FilterValue::Int(65));
        let combined = f1.and(f2);

        assert!(matches!(combined, FilterExpr::Binary { op: LogicalOp::And, .. }));
    }

    #[test]
    fn test_or_filter() {
        let f1 = FilterExpr::condition("status", ComparisonOp::Eq, FilterValue::String("active".to_string()));
        let f2 = FilterExpr::condition("status", ComparisonOp::Eq, FilterValue::String("pending".to_string()));
        let combined = f1.or(f2);

        assert!(matches!(combined, FilterExpr::Binary { op: LogicalOp::Or, .. }));
    }

    // ─── QUERY BUILDER ───

    #[test]
    fn test_simple_query() {
        let query = Query::new("User");

        assert_eq!(query.entity, "User");
        assert!(query.filters.is_empty());
        assert!(query.includes.is_empty());
        assert!(query.limit.is_none());
    }

    #[test]
    fn test_query_with_filter() {
        let query = Query::new("User")
            .filter(FilterExpr::condition(
                "email",
                ComparisonOp::Eq,
                FilterValue::String("ana@mail.com".to_string()),
            ));

        assert_eq!(query.filters.len(), 1);
    }

    #[test]
    fn test_query_with_multiple_filters() {
        let query = Query::new("User")
            .filter(FilterExpr::condition("age", ComparisonOp::Gte, FilterValue::Int(18)))
            .filter(FilterExpr::condition("age", ComparisonOp::Lte, FilterValue::Int(65)));

        assert_eq!(query.filters.len(), 2);
    }

    #[test]
    fn test_query_with_include() {
        let query = Query::new("User")
            .include("orders")
            .include("orders.items");

        assert_eq!(query.includes.len(), 2);
        assert_eq!(query.includes[0].path, vec!["orders"]);
        assert_eq!(query.includes[1].path, vec!["orders", "items"]);
    }

    #[test]
    fn test_query_with_order_by() {
        let query = Query::new("User")
            .order_by("created_at", SortDirection::Desc)
            .order_by("name", SortDirection::Asc);

        assert_eq!(query.order_by.len(), 2);
        assert_eq!(query.order_by[0].direction, SortDirection::Desc);
        assert_eq!(query.order_by[1].direction, SortDirection::Asc);
    }

    #[test]
    fn test_query_with_limit_offset() {
        let query = Query::new("User")
            .limit(10)
            .offset(20);

        assert_eq!(query.limit, Some(10));
        assert_eq!(query.offset, Some(20));
    }

    #[test]
    fn test_full_query() {
        let query = Query::new("User")
            .filter(FilterExpr::condition(
                "email",
                ComparisonOp::Eq,
                FilterValue::String("ana@mail.com".to_string()),
            ))
            .filter(FilterExpr::condition(
                "orders.total",
                ComparisonOp::Gt,
                FilterValue::Int(100),
            ))
            .include("orders")
            .include("orders.items")
            .order_by("created_at", SortDirection::Desc)
            .limit(10)
            .offset(0);

        assert_eq!(query.entity, "User");
        assert_eq!(query.filters.len(), 2);
        assert_eq!(query.includes.len(), 2);
        assert_eq!(query.order_by.len(), 1);
        assert_eq!(query.limit, Some(10));
        assert_eq!(query.offset, Some(0));
    }

    // ─── SERIALIZATION ───

    #[test]
    fn test_query_serialization() {
        let query = Query::new("User")
            .filter(FilterExpr::condition(
                "email",
                ComparisonOp::Eq,
                FilterValue::String("ana@mail.com".to_string()),
            ))
            .include("orders")
            .limit(10);

        // Serializar a JSON
        let json = serde_json::to_string(&query).unwrap();
        assert!(json.contains("User"));
        assert!(json.contains("email"));
        assert!(json.contains("orders"));

        // Deserializar de vuelta
        let restored: Query = serde_json::from_str(&json).unwrap();
        assert_eq!(query, restored);
    }
}