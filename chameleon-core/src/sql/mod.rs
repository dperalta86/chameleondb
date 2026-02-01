pub mod generator;
pub mod naming;

pub use generator::{generate_sql, GeneratedSQL, SqlGenError};

#[cfg(test)]
mod tests {
    use super::*;
    use crate::ast::*;
    use crate::query::*;

    /// Helper: build a standard test schema
    /// User (HasMany) → Order (HasMany) → OrderItem
    fn test_schema() -> Schema {
        let mut schema = Schema::new();

        // User
        let mut user = Entity::new("User".to_string());
        user.add_field(Field {
            name: "id".to_string(),
            field_type: FieldType::UUID,
            nullable: false, unique: false, primary_key: true,
            default: None, backend: None,
        });
        user.add_field(Field {
            name: "email".to_string(),
            field_type: FieldType::String,
            nullable: false, unique: true, primary_key: false,
            default: None, backend: None,
        });
        user.add_field(Field {
            name: "name".to_string(),
            field_type: FieldType::String,
            nullable: false, unique: false, primary_key: false,
            default: None, backend: None,
        });
        user.add_field(Field {
            name: "age".to_string(),
            field_type: FieldType::Int,
            nullable: true, unique: false, primary_key: false,
            default: None, backend: None,
        });
        user.add_relation(Relation {
            name: "orders".to_string(),
            kind: RelationKind::HasMany,
            target_entity: "Order".to_string(),
            foreign_key: Some("user_id".to_string()),
            through: None,
        });
        schema.add_entity(user);

        // Order
        let mut order = Entity::new("Order".to_string());
        order.add_field(Field {
            name: "id".to_string(),
            field_type: FieldType::UUID,
            nullable: false, unique: false, primary_key: true,
            default: None, backend: None,
        });
        order.add_field(Field {
            name: "total".to_string(),
            field_type: FieldType::Decimal,
            nullable: false, unique: false, primary_key: false,
            default: None, backend: None,
        });
        order.add_field(Field {
            name: "status".to_string(),
            field_type: FieldType::String,
            nullable: false, unique: false, primary_key: false,
            default: None, backend: None,
        });
        order.add_field(Field {
            name: "user_id".to_string(),
            field_type: FieldType::UUID,
            nullable: false, unique: false, primary_key: false,
            default: None, backend: None,
        });
        order.add_relation(Relation {
            name: "user".to_string(),
            kind: RelationKind::BelongsTo,
            target_entity: "User".to_string(),
            foreign_key: None,
            through: None,
        });
        order.add_relation(Relation {
            name: "items".to_string(),
            kind: RelationKind::HasMany,
            target_entity: "OrderItem".to_string(),
            foreign_key: Some("order_id".to_string()),
            through: None,
        });
        schema.add_entity(order);

        // OrderItem
        let mut item = Entity::new("OrderItem".to_string());
        item.add_field(Field {
            name: "id".to_string(),
            field_type: FieldType::UUID,
            nullable: false, unique: false, primary_key: true,
            default: None, backend: None,
        });
        item.add_field(Field {
            name: "quantity".to_string(),
            field_type: FieldType::Int,
            nullable: false, unique: false, primary_key: false,
            default: None, backend: None,
        });
        item.add_field(Field {
            name: "price".to_string(),
            field_type: FieldType::Decimal,
            nullable: false, unique: false, primary_key: false,
            default: None, backend: None,
        });
        item.add_field(Field {
            name: "order_id".to_string(),
            field_type: FieldType::UUID,
            nullable: false, unique: false, primary_key: false,
            default: None, backend: None,
        });
        item.add_relation(Relation {
            name: "order".to_string(),
            kind: RelationKind::BelongsTo,
            target_entity: "Order".to_string(),
            foreign_key: None,
            through: None,
        });
        schema.add_entity(item);

        schema
    }

    // ─── NAMING ───

    #[test]
    fn test_naming_conventions() {
        assert_eq!(naming::entity_to_table("User"), "users");
        assert_eq!(naming::entity_to_table("Order"), "orders");
        assert_eq!(naming::entity_to_table("OrderItem"), "order_items");
    }

    // ─── SIMPLE QUERIES ───

    #[test]
    fn test_fetch_all() {
        let schema = test_schema();
        let query = Query::new("User");
        let result = generate_sql(&query, &schema).unwrap();

        assert!(result.main_query.contains("SELECT"));
        assert!(result.main_query.contains("FROM users"));
        assert!(result.eager_queries.is_empty());
    }

    #[test]
    fn test_filter_equality() {
        let schema = test_schema();
        let query = Query::new("User")
            .filter(FilterExpr::condition(
                "email", ComparisonOp::Eq,
                FilterValue::String("ana@mail.com".to_string()),
            ));

        let result = generate_sql(&query, &schema).unwrap();
        assert!(result.main_query.contains("WHERE email = 'ana@mail.com'"));
    }

    #[test]
    fn test_filter_comparison() {
        let schema = test_schema();
        let query = Query::new("User")
            .filter(FilterExpr::condition(
                "age", ComparisonOp::Gte, FilterValue::Int(18),
            ));

        let result = generate_sql(&query, &schema).unwrap();
        assert!(result.main_query.contains("WHERE age >= 18"));
    }

    #[test]
    fn test_multiple_filters() {
        let schema = test_schema();
        let query = Query::new("User")
            .filter(FilterExpr::condition("age", ComparisonOp::Gte, FilterValue::Int(18)))
            .filter(FilterExpr::condition("age", ComparisonOp::Lte, FilterValue::Int(65)));

        let result = generate_sql(&query, &schema).unwrap();
        assert!(result.main_query.contains("age >= 18"));
        assert!(result.main_query.contains("age <= 65"));
        assert!(result.main_query.contains("AND"));
    }

    #[test]
    fn test_like_filter() {
        let schema = test_schema();
        let query = Query::new("User")
            .filter(FilterExpr::condition(
                "name", ComparisonOp::Like,
                FilterValue::String("ana".to_string()),
            ));

        let result = generate_sql(&query, &schema).unwrap();
        assert!(result.main_query.contains("LIKE '%ana%'"));
    }

    // ─── RELATIONS ───

    #[test]
    fn test_filter_on_relation() {
        let schema = test_schema();
        let query = Query::new("User")
            .filter(FilterExpr::condition(
                "orders.total", ComparisonOp::Gt, FilterValue::Int(100),
            ));

        let result = generate_sql(&query, &schema).unwrap();
        assert!(result.main_query.contains("DISTINCT"));
        assert!(result.main_query.contains("INNER JOIN orders"));
        assert!(result.main_query.contains("orders.total > 100"));
    }

    #[test]
    fn test_include_single() {
        let schema = test_schema();
        let query = Query::new("User")
            .include("orders");

        let result = generate_sql(&query, &schema).unwrap();
        assert_eq!(result.eager_queries.len(), 1);
        assert_eq!(result.eager_queries[0].0, "orders");
        assert!(result.eager_queries[0].1.contains("FROM orders"));
        assert!(result.eager_queries[0].1.contains("WHERE user_id IN ($PARENT_IDS)"));
    }

    #[test]
    fn test_include_nested() {
        let schema = test_schema();
        let query = Query::new("User")
            .include("orders")
            .include("orders.items");

        let result = generate_sql(&query, &schema).unwrap();
        assert_eq!(result.eager_queries.len(), 2);

        // First: orders
        assert_eq!(result.eager_queries[0].0, "orders");
        assert!(result.eager_queries[0].1.contains("FROM orders"));

        // Second: items
        assert_eq!(result.eager_queries[1].0, "items");
        assert!(result.eager_queries[1].1.contains("FROM order_items"));
        assert!(result.eager_queries[1].1.contains("WHERE order_id IN ($PARENT_IDS)"));
    }

    // ─── ORDER BY / LIMIT / OFFSET ───

    #[test]
    fn test_order_by() {
        let schema = test_schema();
        let query = Query::new("User")
            .order_by("name", SortDirection::Asc)
            .order_by("age", SortDirection::Desc);

        let result = generate_sql(&query, &schema).unwrap();
        assert!(result.main_query.contains("ORDER BY name ASC, age DESC"));
    }

    #[test]
    fn test_limit_offset() {
        let schema = test_schema();
        let query = Query::new("User")
            .limit(10)
            .offset(20);

        let result = generate_sql(&query, &schema).unwrap();
        assert!(result.main_query.contains("LIMIT 10"));
        assert!(result.main_query.contains("OFFSET 20"));
    }

    // ─── COMBINED ───

    #[test]
    fn test_full_query() {
        let schema = test_schema();
        let query = Query::new("User")
            .filter(FilterExpr::condition("age", ComparisonOp::Gte, FilterValue::Int(18)))
            .filter(FilterExpr::condition("orders.total", ComparisonOp::Gt, FilterValue::Int(50)))
            .include("orders")
            .include("orders.items")
            .order_by("name", SortDirection::Desc)
            .limit(10);

        let result = generate_sql(&query, &schema).unwrap();

        // Main query
        assert!(result.main_query.contains("DISTINCT"));
        assert!(result.main_query.contains("INNER JOIN orders"));
        assert!(result.main_query.contains("users.age >= 18"));
        assert!(result.main_query.contains("orders.total > 50"));
        assert!(result.main_query.contains("ORDER BY"));
        assert!(result.main_query.contains("LIMIT 10"));

        // Eager queries
        assert_eq!(result.eager_queries.len(), 2);
    }

    // ─── ERRORS ───

    #[test]
    fn test_unknown_entity() {
        let schema = test_schema();
        let query = Query::new("NonExistent");

        let result = generate_sql(&query, &schema);
        assert!(result.is_err());
        assert!(matches!(result.unwrap_err(), SqlGenError::UnknownEntity(_)));
    }
}