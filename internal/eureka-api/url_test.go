package eurekaapi

import "testing"

func TestNormalizeBaseURL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"https://example.com/eureka/v2", "https://example.com/eureka/v2"},
		{"https://example.com/eureka/v2/", "https://example.com/eureka/v2"},
		{"http://example.com/eureka", "http://example.com/eureka/v2"},
		{"http://example.com/eureka/", "http://example.com/eureka/v2"},
		{"https://example.com", "https://example.com/eureka/v2"},
		{"https://example.com/", "https://example.com/eureka/v2"},
	}

	for _, test := range tests {
		result, err := normalizeBaseURL(test.input)
		if err != nil {
			t.Errorf("normalizeBaseURL(%q) returned error: %v", test.input, err)
			continue
		}
		if result != test.expected {
			t.Errorf("normalizeBaseURL(%q) = %q; want %q", test.input, result, test.expected)
		}
	}
}
