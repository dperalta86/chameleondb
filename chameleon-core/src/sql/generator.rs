use crate::ast::{Schema, RelationKind};
use crate::query::{
    Query, FilterExpr, FilterCondition, FilterValue,
    ComparisonOp, LogicalOp, SortDirection,
};
use super::naming::entity_to_table;

/// Result of generating SQL for a query
/// A single logical query can produce multiple SQL statements
/// (main query + eager loading queries)
#[derive(Debug, Clone, PartialEq)]
pub struct GeneratedSQL {
    /// The main SELECT query
    pub main_query: String,
    /// Eager loading queries (one per include level)
    /// Each tuple is (relation_name, sql)
    pub eager_queries: Vec<(String, String)>,
}

/// Generate SQL from a Query + Schema
pub fn generate_sql(query: &Query, schema: &Schema) -> Result<GeneratedSQL, SqlGenError> {
    let entity = schema.entities.get(&query.entity)
        .ok_or_else(|| SqlGenError::UnknownEntity(query.entity.clone()))?;

    let table_name = entity_to_table(&query.entity);

    // Determine if we need JOINs (filters on relations)
    let join_filters = extract_join_filters(query);
    let needs_join = !join_filters.is_empty();

    // Build main query
    let main_query = build_main_query(
        &table_name,
        &query.entity,
        entity,
        &query.filters,
        &join_filters,
        needs_join,
        &query.order_by,
        query.limit,
        query.offset,
        schema,
    )?;

    // Build eager loading queries
    let eager_queries = build_eager_queries(
        &query.entity,
        &query.includes,
        schema,
    )?;

    Ok(GeneratedSQL {
        main_query,
        eager_queries,
    })
}

/// Extract filters that target relations (e.g., "orders.total")
fn extract_join_filters(query: &Query) -> Vec<&FilterExpr> {
    query.filters.iter()
        .filter(|f| filter_expr_is_nested(f))
        .collect()
}

/// Check if a filter expression references a nested path
fn filter_expr_is_nested(expr: &FilterExpr) -> bool {
    match expr {
        FilterExpr::Condition(cond) => cond.field.is_nested(),
        FilterExpr::Binary { left, right, .. } => {
            filter_expr_is_nested(left) || filter_expr_is_nested(right)
        }
    }
}

/// Build the main SELECT query
fn build_main_query(
    table_name: &str,
    entity_name: &str,
    entity: &crate::ast::Entity,
    filters: &[FilterExpr],
    join_filters: &[&FilterExpr],
    needs_join: bool,
    order_by: &[crate::query::OrderByClause],
    limit: Option<u64>,
    offset: Option<u64>,
    schema: &Schema,
) -> Result<String, SqlGenError> {
    let mut parts: Vec<String> = Vec::new();

    // SELECT
    let columns = build_select_columns(table_name, entity, needs_join);
    let distinct = if needs_join { "DISTINCT " } else { "" };
    parts.push(format!("SELECT {}{}", distinct, columns));

    // FROM
    parts.push(format!("FROM {}", table_name));

    // JOINs (for relation filters)
    if needs_join {
        let joins = build_joins(entity_name, join_filters, schema)?;
        parts.push(joins);
    }

    // WHERE
    let where_clause = build_where(filters, table_name, needs_join, schema, entity_name)?;
    if !where_clause.is_empty() {
        parts.push(format!("WHERE {}", where_clause));
    }

    // ORDER BY
    if !order_by.is_empty() {
        let order = build_order_by(order_by, table_name, needs_join);
        parts.push(order);
    }

    // LIMIT
    if let Some(limit) = limit {
        parts.push(format!("LIMIT {}", limit));
    }

    // OFFSET
    if let Some(offset) = offset {
        parts.push(format!("OFFSET {}", offset));
    }

    Ok(parts.join("\n"))
}

/// Build SELECT column list
fn build_select_columns(table_name: &str, entity: &crate::ast::Entity, qualify: bool) -> String {
    let columns: Vec<String> = entity.fields.keys()
        .map(|name| {
            if qualify {
                format!("{}.{}", table_name, name)
            } else {
                name.clone()
            }
        })
        .collect();

    columns.join(", ")
}

/// Build JOIN clauses for relation filters
fn build_joins(
    entity_name: &str,
    join_filters: &[&FilterExpr],
    schema: &Schema,
) -> Result<String, SqlGenError> {
    let entity = schema.entities.get(entity_name).unwrap();
    let mut joins = Vec::new();
    let mut joined_relations: Vec<String> = Vec::new();

    for filter in join_filters {
        collect_join_relations(filter, &mut joined_relations);
    }

    // Deduplicate
    joined_relations.sort();
    joined_relations.dedup();

    for rel_name in &joined_relations {
        let relation = entity.relations.get(rel_name)
            .ok_or_else(|| SqlGenError::UnknownRelation {
                entity: entity_name.to_string(),
                relation: rel_name.clone(),
            })?;

        let target_table = entity_to_table(&relation.target_entity);
        let source_table = entity_to_table(entity_name);

        let fk = relation.foreign_key.as_ref()
            .ok_or_else(|| SqlGenError::MissingForeignKey {
                entity: entity_name.to_string(),
                relation: rel_name.clone(),
            })?;

        joins.push(format!(
            "INNER JOIN {} ON {}.{} = {}.id",
            target_table, target_table, fk, source_table
        ));
    }

    Ok(joins.join("\n"))
}

/// Collect relation names from nested filter expressions
fn collect_join_relations(expr: &FilterExpr, relations: &mut Vec<String>) {
    match expr {
        FilterExpr::Condition(cond) => {
            if cond.field.is_nested() {
                relations.push(cond.field.root().to_string());
            }
        }
        FilterExpr::Binary { left, right, .. } => {
            collect_join_relations(left, relations);
            collect_join_relations(right, relations);
        }
    }
}

/// Build WHERE clause
fn build_where(
    filters: &[FilterExpr],
    table_name: &str,
    qualify: bool,
    schema: &Schema,
    entity_name: &str,
) -> Result<String, SqlGenError> {
    if filters.is_empty() {
        return Ok(String::new());
    }

    let conditions: Vec<String> = filters.iter()
        .map(|f| filter_expr_to_sql(f, table_name, qualify, schema, entity_name))
        .collect::<Result<Vec<_>, _>>()?;

    Ok(conditions.join(" AND "))
}

/// Convert a FilterExpr to SQL
fn filter_expr_to_sql(
    expr: &FilterExpr,
    table_name: &str,
    qualify: bool,
    schema: &Schema,
    entity_name: &str,
) -> Result<String, SqlGenError> {
    match expr {
        FilterExpr::Condition(cond) => {
            condition_to_sql(cond, table_name, qualify, schema, entity_name)
        }
        FilterExpr::Binary { left, op, right } => {
            let left_sql = filter_expr_to_sql(left, table_name, qualify, schema, entity_name)?;
            let right_sql = filter_expr_to_sql(right, table_name, qualify, schema, entity_name)?;
            let op_sql = match op {
                LogicalOp::And => "AND",
                LogicalOp::Or => "OR",
            };
            Ok(format!("({} {} {})", left_sql, op_sql, right_sql))
        }
    }
}

/// Convert a single condition to SQL
fn condition_to_sql(
    cond: &FilterCondition,
    table_name: &str,
    qualify: bool,
    schema: &Schema,
    entity_name: &str,
) -> Result<String, SqlGenError> {
    let field_sql = if cond.field.is_nested() {
        // Nested: "orders.total" → "orders.total"
        let rel_name = cond.field.root();
        let entity = schema.entities.get(entity_name).unwrap();
        let relation = entity.relations.get(rel_name)
            .ok_or_else(|| SqlGenError::UnknownRelation {
                entity: entity_name.to_string(),
                relation: rel_name.to_string(),
            })?;
        let target_table = entity_to_table(&relation.target_entity);
        let field_name = &cond.field.segments[1];
        format!("{}.{}", target_table, field_name)
    } else if qualify {
        format!("{}.{}", table_name, cond.field.root())
    } else {
        cond.field.root().to_string()
    };

    let (op_sql, value_sql) = match (&cond.op, &cond.value) {
        (ComparisonOp::Like, FilterValue::String(s)) => {
            ("LIKE".to_string(), format!("'%{}%'", s))
        }
        (ComparisonOp::In, _) => {
            // In is handled specially - value should be a list
            // For now, placeholder
            ("IN".to_string(), "($IN_VALUES)".to_string())
        }
        (op, value) => {
            let op_str = match op {
                ComparisonOp::Eq  => "=",
                ComparisonOp::Neq => "!=",
                ComparisonOp::Gt  => ">",
                ComparisonOp::Gte => ">=",
                ComparisonOp::Lt  => "<",
                ComparisonOp::Lte => "<=",
                _ => unreachable!(),
            };
            (op_str.to_string(), value_to_sql(value))
        }
    };

    Ok(format!("{} {} {}", field_sql, op_sql, value_sql))
}

/// Convert a FilterValue to SQL literal
fn value_to_sql(value: &FilterValue) -> String {
    match value {
        FilterValue::String(s) => format!("'{}'", s),
        FilterValue::Int(n)    => n.to_string(),
        FilterValue::Float(f)  => f.to_string(),
        FilterValue::Bool(b)   => if *b { "true".to_string() } else { "false".to_string() },
        FilterValue::Null      => "NULL".to_string(),
    }
}

/// Build ORDER BY clause
fn build_order_by(
    order_by: &[crate::query::OrderByClause],
    table_name: &str,
    qualify: bool,
) -> String {
    let clauses: Vec<String> = order_by.iter()
        .map(|o| {
            let field = if qualify {
                format!("{}.{}", table_name, o.field)
            } else {
                o.field.clone()
            };
            let dir = match o.direction {
                SortDirection::Asc  => "ASC",
                SortDirection::Desc => "DESC",
            };
            format!("{} {}", field, dir)
        })
        .collect();

    format!("ORDER BY {}", clauses.join(", "))
}

/// Build eager loading queries
fn build_eager_queries(
    root_entity: &str,
    includes: &[crate::query::IncludePath],
    schema: &Schema,
) -> Result<Vec<(String, String)>, SqlGenError> {
    let mut queries = Vec::new();
    let mut processed: Vec<String> = Vec::new();

    for include in includes {
        build_eager_query_for_path(
            root_entity,
            &include.path,
            schema,
            &mut queries,
            &mut processed,
        )?;
    }

    Ok(queries)
}

/// Build a single eager loading query for a given path
fn build_eager_query_for_path(
    current_entity: &str,
    path: &[String],
    schema: &Schema,
    queries: &mut Vec<(String, String)>,
    processed: &mut Vec<String>,
) -> Result<(), SqlGenError> {
    if path.is_empty() {
        return Ok(());
    }

    let rel_name = &path[0];
    let full_path = format!("{}.{}", current_entity, rel_name);

    // Skip if already processed
    if processed.contains(&full_path) {
        // But continue processing deeper paths
        if path.len() > 1 {
            let entity = schema.entities.get(current_entity).unwrap();
            let relation = entity.relations.get(rel_name).unwrap();
            return build_eager_query_for_path(
                &relation.target_entity,
                &path[1..],
                schema,
                queries,
                processed,
            );
        }
        return Ok(());
    }

    let entity = schema.entities.get(current_entity)
        .ok_or_else(|| SqlGenError::UnknownEntity(current_entity.to_string()))?;

    let relation = entity.relations.get(rel_name)
        .ok_or_else(|| SqlGenError::UnknownRelation {
            entity: current_entity.to_string(),
            relation: rel_name.clone(),
        })?;

    let target_entity = schema.entities.get(&relation.target_entity)
        .ok_or_else(|| SqlGenError::UnknownEntity(relation.target_entity.clone()))?;

    let target_table = entity_to_table(&relation.target_entity);

    let fk = relation.foreign_key.as_ref()
        .ok_or_else(|| SqlGenError::MissingForeignKey {
            entity: current_entity.to_string(),
            relation: rel_name.clone(),
        })?;

    // Build columns
    let columns: Vec<String> = target_entity.fields.keys().cloned().collect();

    // Generate eager query with placeholder for IN clause
    let sql = format!(
        "SELECT {}\nFROM {}\nWHERE {} IN ($PARENT_IDS)",
        columns.join(", "),
        target_table,
        fk,
    );

    queries.push((rel_name.clone(), sql));
    processed.push(full_path);

    // Process deeper paths
    if path.len() > 1 {
        build_eager_query_for_path(
            &relation.target_entity,
            &path[1..],
            schema,
            queries,
            processed,
        )?;
    }

    Ok(())
}

/// Errors during SQL generation
#[derive(Debug, Clone, PartialEq)]
pub enum SqlGenError {
    UnknownEntity(String),
    UnknownRelation { entity: String, relation: String },
    MissingForeignKey { entity: String, relation: String },
}

impl std::fmt::Display for SqlGenError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            SqlGenError::UnknownEntity(name) =>
                write!(f, "Unknown entity: {}", name),
            SqlGenError::UnknownRelation { entity, relation } =>
                write!(f, "Unknown relation '{}' in entity '{}'", relation, entity),
            SqlGenError::MissingForeignKey { entity, relation } =>
                write!(f, "Missing foreign key for relation '{}' in '{}'", relation, entity),
        }
    }
}