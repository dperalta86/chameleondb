use crate::ast::Schema;
use crate::error::CharmeleonError;

// LALRPOP genera esto
lalrpop_mod!(pub schema);

pub fn parse_schema(input: &str) -> Result<Schema, CharmeleonError> {
    schema::SchemaParser::new()
        .parse(input)
        .map_err(|e| CharmeleonError::ParseError(format!("{:?}", e)))
}

#[cfg(test)]
mod tests {
    use super::*;
    use pretty_assertions::assert_eq;

    #[test]
    fn test_simple_entity() {
        let input = r#"
            entity User {
                id: uuid primary,
                email: string unique,
                age: int,
            }
        "#;
        
        let schema = parse_schema(input).unwrap();
        assert_eq!(schema.entities.len(), 1);
        
        let user = schema.entities.get("User").unwrap();
        assert_eq!(user.fields.len(), 3);
        assert!(user.fields.get("id").unwrap().primary_key);
        assert!(user.fields.get("email").unwrap().unique);
    }

    #[test]
    fn test_with_relations() {
        let input = r#"
            entity User {
                id: uuid primary,
                email: string,
                orders: [Order] via user_id,
            }
            
            entity Order {
                id: uuid primary,
                total: decimal,
                user: User,
            }
        "#;
        
        let schema = parse_schema(input).unwrap();
        assert_eq!(schema.entities.len(), 2);
        
        let user = schema.entities.get("User").unwrap();
        assert_eq!(user.relations.len(), 1);
        
        let orders_rel = user.relations.get("orders").unwrap();
        assert_eq!(orders_rel.target_entity, "Order");
    }
}