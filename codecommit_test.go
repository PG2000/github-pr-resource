package resource_test

import (
	resource "github.com/pg2000/codecommit-pr-resource"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseRepository(t *testing.T) {
	type args struct {
		repository string
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
			args:       args{repository: "codecommit::eu-central-1://devzone"},
			region:     "eu-central-1",
			repository: "devzone",
			wantErr:    false,
		}, {
			name:       "parse will return region and repository for us-west-1",
			args:       args{repository: "codecommit::us-west-1://test-repo"},
			region:     "us-west-1",
			repository: "test-repo",
			wantErr:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			region, repository, err := resource.ParseRepository(tt.args.repository)

			if assert.NoError(t, err) {
				assert.Equal(t, tt.region, region)
				assert.Equal(t, tt.repository, repository)
			}
		})
	}
}

func TestParseRepositoryFails(t *testing.T) {
	type args struct {
		repository string
	}
	tests := []struct {
		name       string
		args       args
		region     string
		repository string
		wantErr    bool
	}{
		{
			name:       "will not accept an account in the repository url",
			args:       args{repository: "codecommit:11111111111:us-west-1://test-repo"},
			region:     "us-west-1",
			repository: "test-repo",
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := resource.ParseRepository(tt.args.repository)
			assert.Error(t, err)
		})
	}
}
