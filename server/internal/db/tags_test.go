package db

import "testing"

// Scan used to split unconditionally, so an empty tags column produced Tags{""}
// — a single empty tag that Value() then refused to write back, making such a
// row impossible to save.
func TestTagsRoundTrip(t *testing.T) {
	cases := []struct {
		name      string
		src       any
		wantLen   int
		wantValue any
	}{
		{"null", nil, 0, nil},
		{"empty string", "", 0, nil},
		{"empty bytes", []byte(""), 0, nil},
		{"single", "news", 1, "news"},
		{"multiple", "news,sports", 2, "news,sports"},
		{"bytes", []byte("news,sports"), 2, "news,sports"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var tags Tags
			if err := tags.Scan(tc.src); err != nil {
				t.Fatalf("Scan(%v): %v", tc.src, err)
			}
			if len(tags) != tc.wantLen {
				t.Fatalf("Scan(%v) = %#v, want length %d", tc.src, tags, tc.wantLen)
			}

			value, err := tags.Value()
			if err != nil {
				t.Fatalf("Value() after Scan(%v): %v", tc.src, err)
			}
			if value != tc.wantValue {
				t.Fatalf("Value() = %v, want %v", value, tc.wantValue)
			}
		})
	}
}

func TestTagsScanRejectsUnsupportedType(t *testing.T) {
	var tags Tags
	if err := tags.Scan(42); err == nil {
		t.Fatal("expected Scan(int) to fail")
	}
}

func TestTagsValueRejectsInvalidTag(t *testing.T) {
	tags := Tags{"Not Valid!"}
	if _, err := tags.Value(); err == nil {
		t.Fatal("expected an invalid tag to be rejected")
	}
}
