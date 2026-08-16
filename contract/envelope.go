package contract

// Data is the minimal D10 success envelope. Pagination and other meta fields
// are added in M3-2.
type Data[T any] struct {
	Data T `json:"data"`
}
