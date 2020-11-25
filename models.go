package resource

import (
	"errors"
	"strconv"
	"time"
)

// Source represents the configuration for the resource.
type Source struct {
	Repository              string   `json:"repository"`
	Paths                   []string `json:"paths"`
	IgnorePaths             []string `json:"ignore_paths"`
	DisableCISkip           bool     `json:"disable_ci_skip"`
	GitCryptKey             string   `json:"git_crypt_key"`
	BaseBranch              string   `json:"base_branch"`
	Labels                  []string `json:"labels"`
}

// Validate the source configuration.
func (s *Source) Validate() error {
	if s.Repository == "" {
		return errors.New("repository must be set")
	}

	return nil
}

// Metadata output from get/put steps.
type Metadata []*MetadataField

// Add a MetadataField to the Metadata.
func (m *Metadata) Add(name, value string) {
	*m = append(*m, &MetadataField{Name: name, Value: value})
}

// MetadataField ...
type MetadataField struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Version communicated with Concourse.
type Version struct {
	PR            string    `json:"pr"`
	Commit        string    `json:"commit"`
	CommittedDate time.Time `json:"committed,omitempty"`
}

// NewVersion constructs a new Version.
func NewVersion(p *PullRequest) Version {
	return Version{
		PR:            strconv.Itoa(p.Number),
		Commit:        p.Tip.ID,
		CommittedDate: *p.Tip.CommittedDate,
	}
}

// PullRequest represents a pull request and includes the tip (commit).
type PullRequest struct {
	PullRequestObject
	Tip                 CommitObject
	ApprovedReviewCount int
	Labels              []LabelObject
}

// PullRequestObject represents the GraphQL commit node.
// https://developer.github.com/v4/object/pullrequest/
type PullRequestObject struct {
	ID          string
	Number      int
	Title       string
	URL         string
	BaseRefName string
	HeadRefName string
	Repository  struct {
		URL string
	}
}

// CommitObject represents the GraphQL commit node.
// https://developer.github.com/v4/object/commit/
type CommitObject struct {
	ID            string
	CommittedDate *time.Time
	Message       string
	Author        string
}

// ChangedFileObject represents the GraphQL FilesChanged node.
// https://developer.github.com/v4/object/pullrequestchangedfile/
type ChangedFileObject struct {
	Path string
}

// LabelObject represents the GraphQL label node.
// https://developer.github.com/v4/object/label
type LabelObject struct {
	Name string
}
