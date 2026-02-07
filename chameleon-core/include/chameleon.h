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
 *
 * # Example (from C/Go)
 * ```c
 * char* error = NULL;
 * char* json = chameleon_parse_schema("entity User { id: uuid primary, }", &error);
 * if (json) {
 *     printf("%s\n", json);
 *     chameleon_free_string(json);
 * } else {
 *     printf("Error: %s\n", error);
 *     chameleon_free_string(error);
 * }
 * ```
 */
char *chameleon_parse_schema(const char *input, char **error_out);

/**
 * Validate a schema (checks relations, constraints, etc.)
 * Returns JSON with structured errors
 *
 * # Safety
 * - `input` must be a valid null-terminated C string containing schema DSL
 * - Caller must free the returned string with `chameleon_free_string`
 */
enum ChameleonResult chameleon_validate_schema(const char *input, char **error_out);

/**
 * Free a string allocated by Rust
 *
 * # Safety
 * - `s` must be a pointer previously returned by a chameleon_* function
 * - Do not call this twice on the same pointer
 * - Passing NULL is safe (no-op)
 */
void chameleon_free_string(char *s);

/**
 * Get the version of the library
 *
 * # Safety
 * Returns a static string, do not free
 */
const char *chameleon_version(void);

/**
 * Generate SQL from a query JSON + schema JSON
 *
 * Input:  query_json  - serialized Query
 *         schema_json - serialized Schema
 * Output: returns JSON-serialized GeneratedSQL
 *         error_out   - error message on failure
 */
enum ChameleonResult chameleon_generate_sql(const char *query_json,
                                            const char *schema_json,
                                            char **error_out);

/**
 * Generate migration SQL from a schema JSON
 *
 * Input:  schema_json - serialized Schema
 * Output: returns the DDL SQL string directly
 *         error_out   - error message on failure
 */
enum ChameleonResult chameleon_generate_migration(const char *schema_json, char **error_out);

#endif  /* CHAMELEON_H */
