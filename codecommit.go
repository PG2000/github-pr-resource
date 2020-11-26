package resource

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sts"
	"os"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/codecommit"
)

// AwsCodeCommit for testing purposes.
//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -o fakes/fake_codecommit.go . AwsCodeCommit
type AwsCodeCommit interface {
	ListOpenPullRequests() ([]*PullRequest, error)
	ListModifiedFiles(int, string, string) ([]string, error)
	PostComment(string, string, string, string) error
	GetPullRequest(string, string) (*PullRequest, error)
	GetChangedFiles(string, string) ([]ChangedFileObject, error)
	UpdateCommitStatus(string, string, string, string, string, string) error
	DeletePreviousComments(string) error
}

// AwsCodeCommitClient for handling requests to the AwsCodeCommit API.
type AwsCodeCommitClient struct {
	CodeCommit *codecommit.CodeCommit
	Owner      string
	Region     string
	Repository string
	Sts        *sts.STS
}

// NewAwsCodeCommitClient ...
func NewAwsCodeCommitClient(s *Source) (*AwsCodeCommitClient, error) {

	region, repository, err := ParseRepository(s.Repository)

	os.Setenv("AWS_REGION", region)

	if err != nil {
		return nil, err
	}

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	client := codecommit.New(sess)

	return &AwsCodeCommitClient{
		CodeCommit: client,
		Region:     region,
		Repository: repository,
		Sts:        sts.New(sess),
	}, nil
}

// ListOpenPullRequests gets the last commit on all open pull requests.
func (m *AwsCodeCommitClient) ListOpenPullRequests() ([]*PullRequest, error) {
	var newPullRequests []*PullRequest

	repository, err := m.CodeCommit.GetRepository(&codecommit.GetRepositoryInput{
		RepositoryName: aws.String(m.Repository),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get repository %s:  %s", m.Repository, err)
	}

	input := &codecommit.ListPullRequestsInput{
		RepositoryName:    aws.String(m.Repository),
		MaxResults:        aws.Int64(100),
		PullRequestStatus: aws.String(codecommit.PullRequestStatusEnumOpen),
	}

	pullRequests, err := m.CodeCommit.ListPullRequests(input)

	if err != nil {
		return nil, fmt.Errorf("failed to list pull requests: %s", err)
	}

	for _, id := range pullRequests.PullRequestIds {
		pullRequestInput := &codecommit.GetPullRequestInput{
			PullRequestId: id,
		}

		pullRequest, err := m.CodeCommit.GetPullRequest(pullRequestInput)

		if err != nil {
			return nil, fmt.Errorf("failed to get get pull request with id %s: %s", *id, err)
		}

		prNumber, err := strconv.Atoi(*id)

		if err != nil {
			return nil, fmt.Errorf("failed to convert pull request id: '%s' to int: %s", *id, err)
		}
		newPullRequests = append(newPullRequests, &PullRequest{
			PullRequestObject: PullRequestObject{
				Number:      prNumber,
				Title:       *pullRequest.PullRequest.Title,
				URL:         *repository.RepositoryMetadata.CloneUrlHttp,
				BaseRefName: GetSimpleRefName(pullRequest.PullRequest.PullRequestTargets[0].DestinationReference),
				HeadRefName: GetSimpleRefName(pullRequest.PullRequest.PullRequestTargets[0].SourceReference),
				Repository: struct{ URL string }{
					URL: *repository.RepositoryMetadata.CloneUrlHttp,
				},
			},
			Tip: CommitObject{
				ID:            *pullRequest.PullRequest.PullRequestTargets[0].SourceCommit,
				CommittedDate: pullRequest.PullRequest.CreationDate,
				Author:        *pullRequest.PullRequest.AuthorArn,
			},
		})

	}

	return newPullRequests, nil
}

// ListModifiedFiles in a pull request (not supported by V4 API).
func (m *AwsCodeCommitClient) ListModifiedFiles(prNumber int, commitId string, baseRefName string) ([]string, error) {

	input := codecommit.GetBranchInput{RepositoryName: aws.String(m.Repository), BranchName: aws.String(baseRefName)}
	branch, err := m.CodeCommit.GetBranch(&input)

	if err != nil {
		return nil, err
	}

	differences, err := m.CodeCommit.GetDifferences(&codecommit.GetDifferencesInput{
		AfterCommitSpecifier:  aws.String(commitId),
		BeforeCommitSpecifier: branch.Branch.CommitId,
		MaxResults:            aws.Int64(400),
		RepositoryName:        aws.String(m.Repository),
	})

	if err != nil {
		return nil, err
	}

	var files []string
	for _, difference := range differences.Differences {
		if difference.AfterBlob != nil {
			files = append(files, *difference.AfterBlob.Path)
		}
	}

	return files, nil
}

// PostComment to a pull request or issue.
func (m *AwsCodeCommitClient) PostComment(prNumber string, before, after, comment string) error {

	_, err := m.CodeCommit.PostCommentForPullRequest(&codecommit.PostCommentForPullRequestInput{
		PullRequestId:  aws.String(prNumber),
		RepositoryName: aws.String(m.Repository),
		Content:        aws.String(comment),
		AfterCommitId:  aws.String(after),
		BeforeCommitId: aws.String(before),
	})

	return err
}

// GetChangedFiles ...
func (m *AwsCodeCommitClient) GetChangedFiles(prNumber string, commitRef string) ([]ChangedFileObject, error) {
	var cfo []ChangedFileObject
	fmt.Println("GetChangedFiles not implemented for codecommit")
	return cfo, nil
}

// GetPullRequest ...
func (m *AwsCodeCommitClient) GetPullRequest(prNumber, commitRef string) (*PullRequest, error) {

	pullRequest, err := m.CodeCommit.GetPullRequest(&codecommit.GetPullRequestInput{
		PullRequestId: aws.String(prNumber),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to fetch pull request with id: %s, %s", prNumber, err)
	} else {
		pullRequestIdAsInt, err := strconv.Atoi(prNumber)

		if err != nil {
			return nil, fmt.Errorf("failed to convert pull request id: '%s' to int: %s", prNumber, err)
		}

		return &PullRequest{
			PullRequestObject: PullRequestObject{
				Number: pullRequestIdAsInt,
				Title:  *pullRequest.PullRequest.Title,
				URL: fmt.Sprintf(
					"https://%s.console.aws.amazon.com/codesuite/codecommit/repositories/%s/pull-requests/%s",
					m.Region,
					m.Repository,
					prNumber,
				),
				BaseRefName: GetSimpleRefName(pullRequest.PullRequest.PullRequestTargets[0].DestinationReference),
				HeadRefName: GetSimpleRefName(pullRequest.PullRequest.PullRequestTargets[0].SourceReference),
				Repository: struct{ URL string }{
					URL: fmt.Sprintf("codecommit::%s://%s", m.Region, m.Repository),
				},
			},
			Tip: CommitObject{
				ID:            *pullRequest.PullRequest.PullRequestTargets[0].SourceCommit,
				CommittedDate: pullRequest.PullRequest.CreationDate,
				Author:        *pullRequest.PullRequest.AuthorArn,
				Message:       ExtractDescription(pullRequest),
			},
		}, nil
	}

}

// UpdateCommitStatus for a given commit (not supported by V4 API).
func (m *AwsCodeCommitClient) UpdateCommitStatus(commitRef, baseContext, statusContext, status, targetURL, description string) error {
	if baseContext == "" {
		baseContext = "concourse-ci"
	}

	if statusContext == "" {
		statusContext = "status"
	}

	if targetURL == "" {
		targetURL = strings.Join([]string{os.Getenv("ATC_EXTERNAL_URL"), "builds", os.Getenv("BUILD_ID")}, "/")
	}

	if description == "" {
		description = fmt.Sprintf("Concourse CI build %s", status)
	}
	return nil
}

func (m *AwsCodeCommitClient) DeletePreviousComments(prNumber string) error {

	request, err := m.CodeCommit.GetCommentsForPullRequest(&codecommit.GetCommentsForPullRequestInput{
		PullRequestId: aws.String(prNumber),
	})

	if err != nil {
		return fmt.Errorf("failed to delete comments for pull request id '%s': %s", prNumber, err)
	}

	if len(request.CommentsForPullRequestData) > 0 {

		identity, err := m.Sts.GetCallerIdentity(&sts.GetCallerIdentityInput{})

		if err != nil {
			return fmt.Errorf("failed to get caller identity'%s': %s", prNumber, err)
		}

		for _, datum := range request.CommentsForPullRequestData {
			for _, comment := range datum.Comments {
				if *comment.AuthorArn == *identity.Arn && *comment.Deleted == false {
					input := codecommit.DeleteCommentContentInput{
						CommentId: comment.CommentId,
					}

					_, err := m.CodeCommit.DeleteCommentContent(&input)
					return err

				}
			}
		}
	}

	return nil
}

func ParseRepository(s string) (string, string, error) {
	if strings.Contains(s, "codecommit::") {
		s = strings.Replace(s, "codecommit::", "", -1)
	}
	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		return "", "", errors.New("malformed repository. A URL must be in the following format: codecommit::<region>://<repository>")
	}

	parts[1] = strings.Replace(parts[1], "//", "", -1)

	return parts[0], parts[1], nil
}

func ExtractDescription(pullRequest *codecommit.GetPullRequestOutput) string {
	if pullRequest.PullRequest.Description != nil {
		return *pullRequest.PullRequest.Description
	} else {
		return ""
	}
}

func GetSimpleRefName(reference *string) string {
	return strings.Replace(*reference, "refs/heads/", "", 1)
}
