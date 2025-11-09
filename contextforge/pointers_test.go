package contextforge

import (
	"testing"
	"time"
)

func TestString(t *testing.T) {
	v := "test"
	p := String(v)

	if p == nil {
		t.Fatal("String() returned nil")
	}

	if *p != v {
		t.Errorf("String() = %q, want %q", *p, v)
	}
}

func TestStringValue(t *testing.T) {
	tests := []struct {
		name  string
		input *string
		want  string
	}{
		{
			name:  "non-nil pointer",
			input: String("test"),
			want:  "test",
		},
		{
			name:  "nil pointer",
			input: nil,
			want:  "",
		},
		{
			name:  "empty string",
			input: String(""),
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StringValue(tt.input)
			if got != tt.want {
				t.Errorf("StringValue() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestInt(t *testing.T) {
	v := 42
	p := Int(v)

	if p == nil {
		t.Fatal("Int() returned nil")
	}

	if *p != v {
		t.Errorf("Int() = %d, want %d", *p, v)
	}
}

func TestIntValue(t *testing.T) {
	tests := []struct {
		name  string
		input *int
		want  int
	}{
		{
			name:  "non-nil pointer",
			input: Int(42),
			want:  42,
		},
		{
			name:  "nil pointer",
			input: nil,
			want:  0,
		},
		{
			name:  "zero value",
			input: Int(0),
			want:  0,
		},
		{
			name:  "negative value",
			input: Int(-10),
			want:  -10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IntValue(tt.input)
			if got != tt.want {
				t.Errorf("IntValue() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestBool(t *testing.T) {
	tests := []struct {
		name string
		val  bool
	}{
		{
			name: "true",
			val:  true,
		},
		{
			name: "false",
			val:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := Bool(tt.val)

			if p == nil {
				t.Fatal("Bool() returned nil")
			}

			if *p != tt.val {
				t.Errorf("Bool() = %v, want %v", *p, tt.val)
			}
		})
	}
}

func TestBoolValue(t *testing.T) {
	tests := []struct {
		name  string
		input *bool
		want  bool
	}{
		{
			name:  "non-nil pointer to true",
			input: Bool(true),
			want:  true,
		},
		{
			name:  "non-nil pointer to false",
			input: Bool(false),
			want:  false,
		},
		{
			name:  "nil pointer",
			input: nil,
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BoolValue(tt.input)
			if got != tt.want {
				t.Errorf("BoolValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTime(t *testing.T) {
	v := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	p := Time(v)

	if p == nil {
		t.Fatal("Time() returned nil")
	}

	if !p.Equal(v) {
		t.Errorf("Time() = %v, want %v", *p, v)
	}
}

func TestTimeValue(t *testing.T) {
	testTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name  string
		input *time.Time
		want  time.Time
	}{
		{
			name:  "non-nil pointer",
			input: Time(testTime),
			want:  testTime,
		},
		{
			name:  "nil pointer",
			input: nil,
			want:  time.Time{},
		},
		{
			name:  "zero time",
			input: Time(time.Time{}),
			want:  time.Time{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TimeValue(tt.input)
			if !got.Equal(tt.want) {
				t.Errorf("TimeValue() = %v, want %v", got, tt.want)
			}
		})
	}
}
