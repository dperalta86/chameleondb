#ifndef CHAMELEON_H
#define CHAMELEON_H

#include <stdarg.h>
#include <stdbool.h>
#include <stdint.h>
#include <stdlib.h>

/**
 * Result code for FFI functions
 */
typedef enum ChameleonResult {
  Ok = 0,
  ParseError = 1,
  ValidationError = 2,
  InternalError = 3,
} ChameleonResult;

/**
 * Parse a schema from a string and return JSON representation
 *
 * # Safety
 * - `input` must be a valid null-terminated C string
 * - Caller must free the returned string with `chameleon_free_string`
 * - Returns NULL on error, check `error_out` for details
 */
char *chameleon_parse_schema(const char *input, char **error_out);

/**
 * Validate a schema (checks relations, constraints, etc.)
 * Returns JSON with structured errors
 */
enum ChameleonResult chameleon_validate_schema(const char *input, char **error_out);

/**
 * Free a string allocated by Rust
 */
void chameleon_free_string(char *s);

/**
 * Get the version of the library
 */
const char *chameleon_version(void);

/**
 * Generate SQL from a query JSON + schema JSON
 */
enum ChameleonResult chameleon_generate_sql(const char *query_json,
                                            const char *schema_json,
                                            char **error_out);

/**
 * Generate migration SQL from a schema JSON
 */
enum ChameleonResult chameleon_generate_migration(const char *schema_json, char **error_out);

/**
 * Set schema cache for efficient batch operations
 *
 * Call this once before batch mutations, then pass NULL for schema_json
 * in generate_mutation_sql calls to reuse the cached schema
 */
const char *set_schema_cache(const char *schema_json);

/**
 * Clear the schema cache
 * Call this after batch operations to free memory
 */
const char *clear_schema_cache(void);

/**
 * Generate SQL for a mutation operation
 *
 * # Arguments
 * * `mutation_json` - Mutation spec: {"type":"insert|update|delete","entity":"Entity","fields":{...},"filters":{...}}
 * * `schema_json` - Schema JSON (pass NULL to use cached schema from set_schema_cache)
 *
 * # Returns
 * JSON: {"valid":true,"sql":"...","params":[...]} or {"valid":false,"error":"..."}
 */
const char *generate_mutation_sql(const char *mutation_json,
                                  const char *schema_json);

#endif  /* CHAMELEON_H */
