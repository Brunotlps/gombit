// Package contract defines Gombit's Huma DTO conventions and the D10 error
// envelope used for validation failures.
//
// Public API routes are Huma-typed handlers. Request and response shapes come
// from Go structs and Huma validation tags — there is no bespoke Bind layer.
// Validation failures render as:
//
//	{"error":{"code":"validation_error","message":"...","fields":{...},"request_id":"..."}}
//
// Application error categories and success meta helpers are owned by later
// contract issues (M3-2+). See docs/contract.md.
package contract
