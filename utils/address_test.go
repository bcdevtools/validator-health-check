package utils

import (
	"github.com/stretchr/testify/require"
	"testing"
)

//goland:noinspection SpellCheckingInspection
func TestGetValconsHrpFromValoperHrp(t *testing.T) {
	tests := []struct {
		name           string
		valoper        string
		wantValconsHrp string
		wantSuccess    bool
	}{
		{
			name:           "normal",
			valoper:        "dymvaloper1s3fpgacm368dfyn4rmg2qv3h07cmdhr63e59yk",
			wantValconsHrp: "dymvalcons",
			wantSuccess:    true,
		},
		{
			name:           "normal",
			valoper:        "airvaloper1uyp6j8e7k8h8pks0u6kjyalu49g8rl6lahp8fa",
			wantValconsHrp: "airvalcons",
			wantSuccess:    true,
		},
		{
			name:        "no 1",
			valoper:     "dymvalopers3fpgacm368dfyn4rmg2qv3h07cmdhr63e59yk",
			wantSuccess: false,
		},
		{
			name:        "1 at beginning",
			valoper:     "1dymvalopers3fpgacm368dfyn4rmg2qv3h07cmdhr63e59yk",
			wantSuccess: false,
		},
		{
			name:        "valoper not end with valoper",
			valoper:     "dymvalidator1s3fpgacm368dfyn4rmg2qv3h07cmdhr63e59yk",
			wantSuccess: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotValconsHrp, gotSuccess := GetValconsHrpFromValoperHrp(tt.valoper)
			require.Equal(t, tt.wantSuccess, gotSuccess)
			require.Equal(t, tt.wantValconsHrp, gotValconsHrp)
		})
	}
}
