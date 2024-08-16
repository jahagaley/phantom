package checks

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v50/github"
)

const (
	// Names for initial check
	SETUP = "Setup"

	// Names and patterns for Phantom Checks
	PHANTOM_BUILD_IMAGE                 = "Build Image"
	PHANTOM_BUILD_IMAGE_PATTERN         = "Build Image - %s"
	PHANTOM_TEST_IMAGE                  = "Run Test"
	PHANTOM_TEST_IMAGE_PATTERN          = "Run Test - %s"
	PHANTOM_RESOURCE_VALIDATION         = "Resource Validation"
	PHANTOM_RESOURCE_VALIDATION_PATTERN = "Resource Validation - %s"

	// Conclusions for checks
	CONCLUSION_SUCCESS   = "success"
	CONCLUSION_FAILURE   = "failure"
	CONCLUSION_CANCELLED = "cancelled"
	CONCLUSION_TIMED_OUT = "timed_out"

	// Statuses for checks
	STATUS_IN_PROGRESS = "in_progress"
	STATUS_COMPLETED   = "completed"
)

type CheckParams struct {
	// Fields to run checks
	Type          string
	Name          string
	Owner         string
	Repo          string
	HeadSHA       string
	Branch        string
	DefaultBranch string

	Options map[string]string

	// Github fields
	CheckRunID     int64
	InstallationID int64
	RepoID         int64
}

func GetCheckTypeAndOptions(checkName string) (string, map[string]string, error) {
	name := ""
	outputMap := make(map[string]string)
	if strings.HasPrefix(checkName, PHANTOM_BUILD_IMAGE) {
		if _, err := fmt.Sscanf(checkName, PHANTOM_BUILD_IMAGE_PATTERN, &name); err != nil {
			return "", nil, err
		}

		outputMap["name"] = name
		return PHANTOM_BUILD_IMAGE, outputMap, nil
	} else if strings.HasPrefix(checkName, PHANTOM_TEST_IMAGE) {
		if _, err := fmt.Sscanf(checkName, PHANTOM_TEST_IMAGE_PATTERN, &name); err != nil {
			return "", nil, err
		}

		outputMap["name"] = name
		return PHANTOM_TEST_IMAGE, outputMap, nil
	} else if strings.HasPrefix(checkName, PHANTOM_RESOURCE_VALIDATION) {
		if _, err := fmt.Sscanf(checkName, PHANTOM_RESOURCE_VALIDATION_PATTERN, &name); err != nil {
			return "", nil, err
		}

		outputMap["name"] = name
		return PHANTOM_RESOURCE_VALIDATION, outputMap, nil
	}

	return "", outputMap, fmt.Errorf("Unable to parse the check run requested.")
}

func CreateCheckRun(client *github.Client, checkParams *CheckParams) error {
	// Create a check run
	checkRun, _, err := client.Checks.CreateCheckRun(
		context.Background(),
		checkParams.Owner,
		checkParams.Repo,
		github.CreateCheckRunOptions{
			Name:    checkParams.Name,
			HeadSHA: checkParams.HeadSHA,
		},
	)
	if err != nil {
		return fmt.Errorf("Error creating check run: %v", err)
	}

	checkParams.CheckRunID = checkRun.GetID()
	return nil
}

func UpdateCheckRunStatus(
	client *github.Client,
	checkParams *CheckParams,
	status,
	summary string,
) error {
	// Update the check run status
	_, _, err := client.Checks.UpdateCheckRun(
		context.Background(),
		checkParams.Owner,
		checkParams.Repo,
		checkParams.CheckRunID,
		github.UpdateCheckRunOptions{
			Name:   checkParams.Name,
			Status: &status,
			Output: &github.CheckRunOutput{
				Title:   &checkParams.Name,
				Summary: &summary,
			},
		},
	)

	if err != nil {
		return fmt.Errorf("Error updating check run: %v", err)
	}

	return nil
}

func UpdateCheckRunCompletion(
	client *github.Client,
	checkParams *CheckParams,
	conclusion,
	summary string,
	annotations []*github.CheckRunAnnotation,
) error {
	// Update the check run status
	_, _, err := client.Checks.UpdateCheckRun(
		context.Background(),
		checkParams.Owner,
		checkParams.Repo,
		checkParams.CheckRunID,
		github.UpdateCheckRunOptions{
			Name:       checkParams.Name,
			Conclusion: &conclusion,
			Output: &github.CheckRunOutput{
				Title:       &checkParams.Name,
				Summary:     &summary,
				Annotations: annotations,
			},
		},
	)

	if err != nil {
		return fmt.Errorf("Error updating check run: %v", err)
	}

	return nil
}

func UpdateCheckRunCompletionWithOutput(
	client *github.Client,
	checkParams *CheckParams,
	conclusion,
	summary,
	outputText string,
	annotations []*github.CheckRunAnnotation,
) error {
	// Update the check run status
	_, _, err := client.Checks.UpdateCheckRun(
		context.Background(),
		checkParams.Owner,
		checkParams.Repo,
		checkParams.CheckRunID,
		github.UpdateCheckRunOptions{
			Name:       checkParams.Name,
			Conclusion: &conclusion,
			Output: &github.CheckRunOutput{
				Title:       &checkParams.Name,
				Summary:     &summary,
				Text:        &outputText,
				Annotations: annotations,
			},
		},
	)

	if err != nil {
		return fmt.Errorf("Error updating check run: %v", err)
	}

	return nil
}
