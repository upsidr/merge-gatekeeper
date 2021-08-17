package status

import (
	"fmt"
	"testing"
)

func aTest_status_Detail(t *testing.T) {
	tests := map[string]struct {
		s    *status
		want string
	}{
		"": {
			s:    &status{},
			want: "",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := tt.s.Detail()
			fmt.Println(got)
			if got != tt.want {
				t.Errorf("status.Detail() str = %s, want: %s", got, tt.want)
			}
		})
	}
}
