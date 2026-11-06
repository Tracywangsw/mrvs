package repository

import (
	"errors"
	"fmt"
	"testing"

	"github.com/go-sql-driver/mysql"
)

func TestIsDuplicateKey(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"1062", &mysql.MySQLError{Number: 1062}, true},
		{"wrapped 1062", fmt.Errorf("insert: %w", &mysql.MySQLError{Number: 1062}), true},
		{"other mysql error", &mysql.MySQLError{Number: 1452}, false},
		{"plain error", errors.New("duplicate"), false},
		{"nil", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsDuplicateKey(tt.err); got != tt.want {
				t.Fatalf("IsDuplicateKey() = %v, want %v", got, tt.want)
			}
		})
	}
}
