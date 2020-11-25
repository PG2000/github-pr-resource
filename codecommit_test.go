package resource

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseRepository(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name       string
		args       args
		region     string
		repository string
		wantErr    bool
	}{
		{
			name:       "parse will return region and repository for eu-central-1",
			args:       args{s: "codecommit::eu-central-1://devzone"},
			region:     "eu-central-1",
			repository: "devzone",
			wantErr:    false,
		}, {
			name:       "parse will return region and repository for us-west-1",
			args:       args{s: "codecommit::us-west-1://test-repo"},
			region:     "us-west-1",
			repository: "test-repo",
			wantErr:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			region, repository, err := parseRepository(tt.args.s)

			if assert.NoError(t, err) {
				assert.Equal(t, tt.region, region)
				assert.Equal(t, tt.repository, repository)
			}
		})
	}
}

func TestParseRepositoryFails(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name       string
		args       args
		region     string
		repository string
		wantErr    bool
	}{
		{
			name:       "will not accept a profile within the schema",
			args:       args{s: "codecommit:11111111111:us-west-1://test-repo"},
			region:     "us-west-1",
			repository: "test-repo",
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := parseRepository(tt.args.s)
			assert.Error(t, err)
		})
	}
}
