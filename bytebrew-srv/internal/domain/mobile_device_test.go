package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMobileDevice_Validate(t *testing.T) {
	tests := []struct {
		name    string
		device  MobileDevice
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid device",
			device: MobileDevice{
				ID: "dev-1", Name: "iPhone", DeviceToken: "tok-1",
				PairedAt: time.Now(), LastSeenAt: time.Now(),
			},
			wantErr: false,
		},
		{
			name: "missing id",
			device: MobileDevice{
				Name: "iPhone", DeviceToken: "tok-1",
			},
			wantErr: true,
			errMsg:  "device id is required",
		},
		{
			name: "missing name",
			device: MobileDevice{
				ID: "dev-1", DeviceToken: "tok-1",
			},
			wantErr: true,
			errMsg:  "device name is required",
		},
		{
			name: "missing device token",
			device: MobileDevice{
				ID: "dev-1", Name: "iPhone",
			},
			wantErr: true,
			errMsg:  "device token is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.device.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestMobileDevice_UpdateLastSeen(t *testing.T) {
	device := MobileDevice{
		ID: "dev-1", Name: "iPhone", DeviceToken: "tok-1",
		LastSeenAt: time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	before := time.Now()
	device.UpdateLastSeen()

	assert.False(t, device.LastSeenAt.Before(before))
}
