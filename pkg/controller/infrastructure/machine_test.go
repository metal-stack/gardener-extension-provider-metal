package infrastructure

import "testing"

func Test_decodeMachineID(t *testing.T) {
	tests := []struct {
		name string
		id   string
		want string
	}{
		{
			name: "passing empty string",
			id:   "",
			want: "",
		},
		{
			name: "passing an id",
			id:   "metal:///fra-equ01/43c96a25-4328-4aed-9aae-426515bef162",
			want: "43c96a25-4328-4aed-9aae-426515bef162",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if got := decodeMachineID(tt.id); got != tt.want {
				t.Errorf("decodeMachineID() = %v, want %v", got, tt.want)
			}
		})
	}
}
