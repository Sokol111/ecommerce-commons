package mongo

import (
	"testing"
)

func TestBuildURI(t *testing.T) {
	tests := []struct {
		name     string
		conf     Config
		expected string
	}{
		{
			name: "connection string override",
			conf: Config{
				ConnectionString: "mongodb://custom:27017/mydb",
				Host:             "ignored",
				Port:             9999,
			},
			expected: "mongodb://custom:27017/mydb",
		},
		{
			name: "basic host and port",
			conf: Config{
				Host:     "localhost",
				Port:     27017,
				Database: "testdb",
			},
			expected: "mongodb://localhost:27017/testdb",
		},
		{
			name: "with username and password",
			conf: Config{
				Host:     "localhost",
				Port:     27017,
				Database: "testdb",
				Username: "admin",
				Password: "secret",
			},
			expected: "mongodb://admin:secret@localhost:27017/testdb",
		},
		{
			name: "with replica set",
			conf: Config{
				Host:       "localhost",
				Port:       27017,
				Database:   "testdb",
				ReplicaSet: "rs0",
			},
			expected: "mongodb://localhost:27017/testdb?replicaSet=rs0",
		},
		{
			name: "with direct connection",
			conf: Config{
				Host:             "localhost",
				Port:             27017,
				Database:         "testdb",
				DirectConnection: true,
			},
			expected: "mongodb://localhost:27017/testdb?directConnection=true",
		},
		{
			name: "with all options",
			conf: Config{
				Host:             "mongo.example.com",
				Port:             27018,
				Database:         "production",
				Username:         "user",
				Password:         "pass123",
				ReplicaSet:       "rs-prod",
				DirectConnection: true,
			},
			expected: "mongodb://user:pass123@mongo.example.com:27018/production?directConnection=true&replicaSet=rs-prod",
		},
		{
			name: "password with special characters",
			conf: Config{
				Host:     "localhost",
				Port:     27017,
				Database: "testdb",
				Username: "admin",
				Password: "p@ss:word/123",
			},
			expected: "mongodb://admin:p%40ss%3Aword%2F123@localhost:27017/testdb",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildURI(tt.conf)
			if result != tt.expected {
				t.Errorf("buildURI() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		conf    Config
		wantErr bool
	}{
		{
			name: "valid config with host port database",
			conf: Config{
				Host:     "localhost",
				Port:     27017,
				Database: "testdb",
			},
			wantErr: false,
		},
		{
			name: "valid config with connection string",
			conf: Config{
				ConnectionString: "mongodb://localhost:27017/testdb",
			},
			wantErr: false,
		},
		{
			name: "connection string overrides missing fields",
			conf: Config{
				ConnectionString: "mongodb://localhost:27017/testdb",
				Host:             "", // missing but ignored
				Port:             0,  // missing but ignored
			},
			wantErr: false,
		},
		{
			name: "missing host",
			conf: Config{
				Port:     27017,
				Database: "testdb",
			},
			wantErr: true,
		},
		{
			name: "missing port",
			conf: Config{
				Host:     "localhost",
				Database: "testdb",
			},
			wantErr: true,
		},
		{
			name: "missing database",
			conf: Config{
				Host: "localhost",
				Port: 27017,
			},
			wantErr: true,
		},
		{
			name:    "empty config",
			conf:    Config{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.conf)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
