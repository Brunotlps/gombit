package framework

import "testing"

func TestStripHTML(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "plain", in: "hello", want: "hello"},
		{name: "script discarded", in: `<script>alert(1)</script>hi`, want: "hi"},
		{name: "bold stripped", in: `<b>hi</b>`, want: "hi"},
		{name: "img stripped", in: `x<img src=x onerror=alert(0)>y`, want: "xy"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := stripHTML(tt.in); got != tt.want {
				t.Fatalf("stripHTML(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
