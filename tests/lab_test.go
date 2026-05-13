package tests

import (
	"testing"

	"github.com/edu-lab-platform/internal/lab"
)

func TestContainerName(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"student1", "lab_student1"},
		{"admin", "lab_admin"},
		{"", "lab_"},
	}
	for _, tc := range cases {
		if got := lab.ContainerName(tc.in); got != tc.want {
			t.Errorf("ContainerName(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
