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
 *
 * # Safety
 * - `schema_json` must be a valid null-terminated C string containing JSON
 * - Returns ChameleonResult::Ok on success
 */
enum ChameleonResult chameleon_validate_schema(const char *schema_json, char **error_out);

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

#endif  /* CHAMELEON_H */
