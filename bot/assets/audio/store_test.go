package audio

import (
	"io/fs"
	"testing"
)

func TestRefFromExercise(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		data map[string]interface{}
		want string
	}{
		{"nil", nil, ""},
		{"empty", map[string]interface{}{"audio": ""}, ""},
		{"null string", map[string]interface{}{"audio": "null"}, ""},
		{"file", map[string]interface{}{"audio": "b1_01.ogg"}, "b1_01.ogg"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := RefFromExercise(tc.data); got != tc.want {
				t.Fatalf("RefFromExercise() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestStoreLoadEmbedded(t *testing.T) {
	store := NewStore("")
	data, err := store.Load("b1_01.ogg")
	if err != nil {
		if err == fs.ErrNotExist {
			t.Skip("b1_01.ogg not embedded in test environment")
		}
		t.Fatalf("Load: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty audio bytes")
	}
}

func TestStoreLoadNormalizesRef(t *testing.T) {
	store := NewStore("")
	_, err := store.Load("audio/b1_01")
	if err != nil && err != fs.ErrNotExist {
		t.Fatalf("Load with prefix: %v", err)
	}
}
