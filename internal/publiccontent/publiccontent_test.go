package publiccontent

import "testing"

func TestWebURL(t *testing.T) {
	for _, tt := range []struct {
		value    string
		required bool
		valid    bool
	}{
		{value: "https://example.com/image.png", required: true, valid: true},
		{value: "http://localhost:8080/image.png", required: true, valid: true},
		{value: "", required: false, valid: true},
		{value: "", required: true, valid: false},
		{value: "javascript:alert(1)", required: true, valid: false},
		{value: "https://user:password@example.com", required: true, valid: false},
		{value: "//example.com/image.png", required: true, valid: false},
	} {
		_, err := webURL(tt.value, tt.required)
		if (err == nil) != tt.valid {
			t.Errorf("webURL(%q, %v): error=%v, valid=%v", tt.value, tt.required, err, tt.valid)
		}
	}
}
