/// Converts schema names to PostgreSQL naming conventions
///
/// Entity names: PascalCase → snake_case plural
///   User        → users
///   OrderItem   → order_items
///
/// Field names: already snake_case, pass through
///   email       → email
///   created_at  → created_at

/// Convert a PascalCase entity name to a snake_case plural table name
pub fn entity_to_table(entity_name: &str) -> String {
    let snake = pascal_to_snake(entity_name);
    pluralize(&snake)
}

/// Convert PascalCase to snake_case
/// "OrderItem" → "order_item"
/// "User"      → "user"
/// "UUID"      → "u_u_i_d" (edge case, handled separately if needed)
fn pascal_to_snake(name: &str) -> String {
    let mut result = String::new();

    for (i, ch) in name.chars().enumerate() {
        if ch.is_uppercase() && i > 0 {
            // Don't add underscore if previous char was also uppercase
            // This handles acronyms like "ID" better
            let prev = name.chars().nth(i - 1).unwrap();
            if prev.is_lowercase() {
                result.push('_');
            }
        }
        result.push(ch.to_lowercase().next().unwrap());
    }

    result
}

/// Simple pluralization (English rules, covers most cases)
/// "user"       → "users"
/// "order"      → "orders"
/// "order_item" → "order_items"
fn pluralize(name: &str) -> String {
    // For now: just add 's'
    // Can be extended later for irregular plurals
    format!("{}s", name)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_simple_entity() {
        assert_eq!(entity_to_table("User"), "users");
        assert_eq!(entity_to_table("Order"), "orders");
    }

    #[test]
    fn test_compound_entity() {
        assert_eq!(entity_to_table("OrderItem"), "order_items");
    }
}