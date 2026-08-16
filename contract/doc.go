// Package contract defines Gombit's Huma DTO conventions, the D10 success and
// error envelopes, and draft §41 application error category mapping.
//
// Public API routes are Huma-typed handlers. Request and response shapes come
// from Go structs and Huma validation tags — there is no bespoke Bind layer.
//
// Success:
//
//	{"data": ...}
//	{"data": ..., "meta": {"page": 1, "per_page": 20, "total": 125}}
//
// Validation failures (Huma Install path):
//
//	{"error":{"code":"validation_error","message":"...","fields":{...},"request_id":"..."}}
//
// Application errors (return contract.NotFound / New / ... via WithContext):
//
//	{"error":{"code":"not_found","message":"...","request_id":"..."}}
//
// See docs/contract.md.
package contract
