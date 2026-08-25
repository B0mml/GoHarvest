package parser

import (
	"testing"
)

func TestCleanPrice(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    float64
		wantErr bool
	}{
		{"US standard", "$19.99", 19.99, false},
		{"US with comma thousands", "$1,299.50", 1299.50, false},
		{"EU standard", "19,99 €", 19.99, false},
		{"EU with dot thousands", "1.299,99 €", 1299.99, false},
		{"German formatted EUR", "€ 45,00", 45.00, false},
		{"Plain integer", "150", 150.00, false},
		{"Single decimal digit", "49.9", 49.90, false},
		{"Invalid string", "Out of stock", 0, true},
		{"Empty string", "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CleanPrice(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("CleanPrice(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("CleanPrice(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
