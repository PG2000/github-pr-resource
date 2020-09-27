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
	Repository string
	Owner      string
	CodeCommit *codecommit.CodeCommit
	Sts        *sts.STS
}

// NewAwsCodeCommitClient ...
func NewAwsCodeCommitClient(s *Source) (*AwsCodeCommitClient, error) {

	//TODO: remove set env. get from repo name
	err := os.Setenv("AWS_REGION", "eu-central-1")

	if err != nil {
		fmt.Println(err)
	}

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	//TODO: remove
	getenv := os.Getenv("AWS_REGION")
	sess.Config.Region = aws.String(getenv)

	client := codecommit.New(sess)

	return &AwsCodeCommitClient{
		Sts:        sts.New(sess),
		CodeCommit: client,
		Owner:      "",
		Repository: s.Repository,
	}, nil
}

// ListOpenPullRequests gets the last commit on all open pull requests.
func (m *AwsCodeCommitClient) ListOpenPullRequests() ([]*PullRequest, error) {

	var newPullRequests []*PullRequest

	m.CodeCommit.Config.Region = aws.String("eu-central-1")
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

		pullRequestIdAsInt, err := strconv.Atoi(*id)

		if err != nil {
			return nil, fmt.Errorf("failed to convert pull request id: '%s' to int: %s", *id, err)
		}
		newPullRequests = append(newPullRequests, &PullRequest{
			PullRequestObject: PullRequestObject{
				ID:          *id,
				Number:      pullRequestIdAsInt,
				Title:       *pullRequest.PullRequest.Title,
				URL:         "TO BE Implemented",
				BaseRefName: getSimpleRefName(pullRequest.PullRequest.PullRequestTargets[0].DestinationReference),
				HeadRefName: getSimpleRefName(pullRequest.PullRequest.PullRequestTargets[0].SourceReference),
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

func getSimpleRefName(reference *string) string {
	return strings.Replace(*reference, "refs/heads/", "", 1)
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

		//repository, err := m.CodeCommit.GetRepository(&codecommit.GetRepositoryInput{
		//	RepositoryName: aws.String(m.Repository),
		//})

		return &PullRequest{
			PullRequestObject: PullRequestObject{
				ID:          prNumber,
				Number:      pullRequestIdAsInt,
				Title:       *pullRequest.PullRequest.Title,
				URL:         "TO BE Implemented",
				BaseRefName: getSimpleRefName(pullRequest.PullRequest.PullRequestTargets[0].DestinationReference),
				HeadRefName: getSimpleRefName(pullRequest.PullRequest.PullRequestTargets[0].SourceReference),
				Repository: struct{ URL string }{
					//TODO: get dynamic
					URL: "codecommit::eu-central-1://" + m.Repository,
				},
			},
			Tip: CommitObject{
				ID:            *pullRequest.PullRequest.PullRequestTargets[0].SourceCommit,
				CommittedDate: pullRequest.PullRequest.CreationDate,
				Author:        *pullRequest.PullRequest.AuthorArn,
				Message:       *pullRequest.PullRequest.Description,
			},
		}, nil
	}

	// Return an error if the commit was not found
	return nil, fmt.Errorf("commit with ref '%s' does not exist", commitRef)
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
				if *comment.AuthorArn == *identity.Arn {
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

func parseRepository(s string) (string, string, error) {
	parts := strings.Split(s, "/")
	if len(parts) != 2 {
		return "", "", errors.New("malformed repository")
	}
	return parts[0], parts[1], nil
}
