package syncer

import "testing"

func TestIsNewerThan(t *testing.T) {
	testCases := []struct {
		name     string
		incoming FieldVersion
		current  FieldVersion
		expected bool
	}{
		{
			name:     "greater incoming time",
			current:  FieldVersion{0, "desktop"},
			incoming: FieldVersion{2, "desktop"},
			expected: true,
		},
		{
			name:     "lesser incoming time",
			current:  FieldVersion{2, "desktop"},
			incoming: FieldVersion{1, "desktop"},
			expected: false,
		},
		{
			name:     "same timestamp, greater device",
			current:  FieldVersion{2, "desktop"},
			incoming: FieldVersion{2, "laptop"},
			expected: true,
		},
		{
			name:     "same timestamp, lesser device",
			current:  FieldVersion{2, "desktop"},
			incoming: FieldVersion{2, "a"},
			expected: false,
		},
		{
			name:     "same timestamp, same device",
			current:  FieldVersion{2, "desktop"},
			incoming: FieldVersion{2, "desktop"},
			expected: false,
		},
		{
			name:     "lesser timestamp, greater device",
			current:  FieldVersion{2, "desktop"},
			incoming: FieldVersion{1, "laptop"},
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.incoming.IsNewerThan(tc.current)
			if tc.expected != result {
				t.Fatalf("wrong compare result. expected: %t, got %t", tc.expected, result)
			}
		})
	}

}
