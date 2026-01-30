use serde::{Deserialize, Serialize};
use std::collections::HashMap;

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct Schema {
    pub entities: HashMap<String, Entity>,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct Entity {
    pub name: String,
    pub fields: HashMap<String, Field>,
    pub relations: HashMap<String, Relation>,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct Field {
    pub name: String,
    pub field_type: FieldType,
    pub nullable: bool,
    pub unique: bool,
    pub primary_key: bool,
    pub default: Option<DefaultValue>,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub enum FieldType {
    UUID,
    String,
    Int,
    Decimal,
    Bool,
    Timestamp,
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub enum DefaultValue {
    Now,
    UUIDv4,
    Literal(String),
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub struct Relation {
    pub name: String,
    pub kind: RelationKind,
    pub target_entity: String,
    pub foreign_key: Option<String>,
    pub through: Option<String>,  // para many_to_many
}

#[derive(Debug, Clone, PartialEq, Serialize, Deserialize)]
pub enum RelationKind {
    HasOne,
    HasMany,
    BelongsTo,
    ManyToMany,
}

impl Schema {
    pub fn new() -> Self {
        Schema {
            entities: HashMap::new(),
        }
    }
    
    pub fn add_entity(&mut self, entity: Entity) {
        self.entities.insert(entity.name.clone(), entity);
    }
}

impl Entity {
    pub fn new(name: String) -> Self {
        Entity {
            name,
            fields: HashMap::new(),
            relations: HashMap::new(),
        }
    }
    
    pub fn add_field(&mut self, field: Field) {
        self.fields.insert(field.name.clone(), field);
    }
    
    pub fn add_relation(&mut self, relation: Relation) {
        self.relations.insert(relation.name.clone(), relation);
    }
}