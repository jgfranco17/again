package again

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewConfig(t *testing.T) {
	cfg := NewConfig()

	assert.Equal(t, 3, cfg.Attempts)
	assert.NotNil(t, cfg.Backoff)
	assert.NotNil(t, cfg.Jitter)
	assert.NotNil(t, cfg.RetryIf)
	assert.Nil(t, cfg.OnRetry)
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: Config{
				Attempts: 3,
				Backoff:  Constant(100 * time.Millisecond),
			},
			wantErr: false,
		},
		{
			name: "zero attempts",
			cfg: Config{
				Attempts: 0,
			},
			wantErr: true,
		},
		{
			name: "negative attempts",
			cfg: Config{
				Attempts: -1,
			},
			wantErr: true,
		},
		{
			name: "minimal valid config",
			cfg: Config{
				Attempts: 1,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				var cfgErr *ConfigError
				assert.ErrorAs(t, err, &cfgErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfigError_Error(t *testing.T) {
	err := &ConfigError{
		Field:  "Attempts",
		Reason: "must be positive",
	}

	assert.Equal(t, "invalid config: Attempts must be positive", err.Error())
}
