use crate::ast::Schema;
use serde_json::{json, Value};

/// Generate SQL for a mutation operation
pub fn generate_mutation_sql(mutation: &Value, schema: &Schema) -> Value {
    let mutation_type = match mutation.get("type").and_then(|v| v.as_str()) {
        Some(t) => t,
        None => return error_response("Missing 'type' field in mutation"),
    };

    let entity_name = match mutation.get("entity").and_then(|v| v.as_str()) {
        Some(e) => e,
        None => return error_response("Missing 'entity' field in mutation"),
    };

    let entity = match schema.get_entity(entity_name) {
        Some(e) => e,
        None => return error_response(&format!("Entity '{}' not found in schema", entity_name)),
    };

    match mutation_type {
        "insert" => generate_insert_sql(mutation, entity),
        "update" => generate_update_sql(mutation, entity),
        "delete" => generate_delete_sql(mutation, entity),
        _ => error_response(&format!("Unknown mutation type: '{}'", mutation_type)),
    }
}

/// Generate INSERT SQL
fn generate_insert_sql(mutation: &Value, entity: &crate::ast::Entity) -> Value {
    let fields = match mutation.get("fields").and_then(|v| v.as_object()) {
        Some(f) => f,
        None => return error_response("Missing or invalid 'fields' in INSERT mutation"),
    };

    for field_name in fields.keys() {
        if !entity.fields.contains_key(field_name) {
            return error_response(&format!("Unknown field '{}' in entity '{}'", field_name, entity.name));
        }
    }

    let mut columns = Vec::new();
    let mut placeholders = Vec::new();
    let mut params = Vec::new();

    let mut param_index = 1;
    for field_name in fields.keys() {
        columns.push(format!("\"{}\"", field_name));
        placeholders.push(format!("${}", param_index));
        params.push(field_name.clone());
        param_index += 1;
    }

    let table_name = entity_to_table(&entity.name);
    let sql = format!(
        "INSERT INTO {} ({}) VALUES ({}) RETURNING *",
        table_name,
        columns.join(", "),
        placeholders.join(", ")
    );

    json!({
        "valid": true,
        "sql": sql,
        "params": params,
        "affected": 1
    })
}

/// Generate UPDATE SQL
fn generate_update_sql(mutation: &Value, entity: &crate::ast::Entity) -> Value {
    let fields = match mutation.get("fields").and_then(|v| v.as_object()) {
        Some(f) => f,
        None => return error_response("Missing or invalid 'fields' in UPDATE mutation"),
    };

    let filters = match mutation.get("filters").and_then(|v| v.as_object()) {
        Some(f) => f,
        None => return error_response("Missing or invalid 'filters' in UPDATE mutation"),
    };

    if filters.is_empty() {
        return error_response("UPDATE requires at least one filter (WHERE clause)");
    }

    for field_name in fields.keys() {
        if !entity.fields.contains_key(field_name) {
            return error_response(&format!("Unknown field '{}' in entity '{}'", field_name, entity.name));
        }
        if let Some(field) = entity.fields.get(field_name) {
            if field.primary_key {
                return error_response("Cannot update primary key");
            }
        }
    }

    let mut set_clauses = Vec::new();
    let mut params = Vec::new();
    let mut param_index = 1;

    for field_name in fields.keys() {
        set_clauses.push(format!("\"{}\"=${}", field_name, param_index));
        params.push(field_name.clone());
        param_index += 1;
    }

    let mut where_parts = Vec::new();
    for filter_field in filters.keys() {
        where_parts.push(format!("\"{}\"=${}", filter_field, param_index));
        params.push(filter_field.clone());
        param_index += 1;
    }

    let table_name = entity_to_table(&entity.name);
    let sql = format!(
        "UPDATE {} SET {} WHERE {} RETURNING *",
        table_name,
        set_clauses.join(", "),
        where_parts.join(" AND ")
    );

    json!({
        "valid": true,
        "sql": sql,
        "params": params,
        "affected": 1
    })
}

/// Generate DELETE SQL
fn generate_delete_sql(mutation: &Value, entity: &crate::ast::Entity) -> Value {
    let filters = match mutation.get("filters").and_then(|v| v.as_object()) {
        Some(f) => f,
        None => return error_response("Missing or invalid 'filters' in DELETE mutation"),
    };

    if filters.is_empty() {
        return error_response("DELETE requires at least one filter (WHERE clause) for safety");
    }

    let mut where_parts = Vec::new();
    let mut params = Vec::new();
    let mut param_index = 1;

    for filter_field in filters.keys() {
        where_parts.push(format!("\"{}\"=${}", filter_field, param_index));
        params.push(filter_field.clone());
        param_index += 1;
    }

    let table_name = entity_to_table(&entity.name);
    let sql = format!(
        "DELETE FROM {} WHERE {}",
        table_name,
        where_parts.join(" AND ")
    );

    json!({
        "valid": true,
        "sql": sql,
        "params": params,
        "affected": 1
    })
}

/// Helper: Convert entity name to table name
fn entity_to_table(entity_name: &str) -> String {
    let mut table_name = String::new();
    for (i, ch) in entity_name.chars().enumerate() {
        if i > 0 && ch.is_uppercase() {
            table_name.push('_');
        }
        table_name.push(ch.to_lowercase().next().unwrap());
    }
    table_name
}

/// Helper: Build error response JSON
fn error_response(message: &str) -> Value {
    json!({
        "valid": false,
        "error": message
    })
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_entity_to_table() {
        assert_eq!(entity_to_table("User"), "user");
        assert_eq!(entity_to_table("UserProfile"), "user_profile");
        assert_eq!(entity_to_table("OrderItem"), "order_item");
    }
}