#[test]
fn test_sql_output() {
    use crate::mutation::generate_mutation_sql;
    use crate::ast::{Entity, Field, FieldType, Schema};
    
    let mut schema = Schema::new();
    let mut user = Entity::new("User".to_string());
    user.add_field(Field {
        name: "id".to_string(),
        field_type: FieldType::UUID,
        nullable: false,
        unique: false,
        primary_key: true,
        default: None,
        backend: None,
    });
    user.add_field(Field {
        name: "email".to_string(),
        field_type: FieldType::String,
        nullable: false,
        unique: true,
        primary_key: false,
        default: None,
        backend: None,
    });
    schema.add_entity(user);

    let mutation = serde_json::json!({
        "type": "insert",
        "entity": "User",
        "fields": {"email": "test@mail.com"}
    });

    let result = generate_mutation_sql(&mutation, &schema);
    println!("RESULT: {}", serde_json::to_string_pretty(&result).unwrap());
    println!("SQL: {}", result["sql"]);
}
