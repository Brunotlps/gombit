// Package client generates a TypeScript API client from an OpenAPI 3.1 document.
//
// gombit client generate runs openapi-typescript for schema types and writes a
// thin openapi-fetch wrapper whose errors map to the D10 envelope.
//
// gombit client check regenerates the sample spec and client in-process from
// SampleApp and fails when committed artifacts would change.
package client
