package registry

import "testing"

func TestEndpointParse(t *testing.T) {
	testData := []struct {
		str      string
		expected string
	}{
		{IndexServerAddress(), IndexServerAddress()},
		{"http://0.0.0.0:5000", "http://0.0.0.0:5000/v1/"},
		{"0.0.0.0:5000", "https://0.0.0.0:5000/v1/"},
	}
	for _, td := range testData {
		e, err := NewEndpoint(td.str)
		if err != nil {
			t.Errorf("%q: %s", td.str, err)
		}
		t.Logf("%#v", e)
		t.Logf("%#v", e.URL.String())
		if e.String() != td.expected {
			t.Errorf("expected %q, got %q", td.expected, e.String())
		}
	}
}
