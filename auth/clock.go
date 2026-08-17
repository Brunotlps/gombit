package auth

import "time"

// Clock is the time source used for token expiry. Tests inject a fake.
type Clock interface {
	Now() time.Time
}

type systemClock struct{}

func (systemClock) Now() time.Time {
	return time.Now()
}
