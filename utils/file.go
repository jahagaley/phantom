package utils

import (
	"context"
	"io"
	"net/http"
	"os"
	"path"

	"github.com/google/go-github/v50/github"
	"github.com/rs/zerolog/log"

	"github.com/jahagaley/phantomcli/utils"
)

var (
	tempCompressedTarball = "temp.tar.gz"
)

func HandleGithubRepoDownload(client *github.Client, owner, repo, headSha string) (string, error) {

	// creating a temp directory for repo
	tempDir, err := os.MkdirTemp("", "")
	if err != nil {
		return "", err
	}

	// Download entire git repository using tar
	err = DownloadAndExtractRepoTarFile(client, tempDir, owner, repo, headSha)
	if err != nil {
		log.Warn().Err(err).Send()
		return "", err
	}

	return tempDir, nil
}

func DownloadAndExtractRepoTarFile(client *github.Client, tempDir, owner, repo, sha string) error {
	getOptions := &github.RepositoryContentGetOptions{}
	if len(sha) > 0 {
		getOptions = &github.RepositoryContentGetOptions{
			Ref: sha,
		}
	}
	url, _, err := client.Repositories.GetArchiveLink(
		context.Background(),
		owner,
		repo,
		github.Tarball,
		getOptions,
		true,
	)

	if err != nil {
		log.Warn().Err(err).Send()
		return err
	}

	log.Debug().Msg("Downloading Github repository tarball file.")
	resp, err := http.Get(url.String())
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	tempTarFile := path.Join(tempDir, tempCompressedTarball)
	file, err := os.Create(tempTarFile)
	if err != nil {
		os.Remove(tempTarFile)
		return err
	}
	defer os.Remove(tempTarFile)

	// Write the body to file
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return err
	}
	file.Close()

	log.Debug().Msg("Decompressing Github tarball file.")
	return utils.ExtractTarGz(tempTarFile, tempDir, 1)
}

func GetInputFiles(dir string) ([]string, error) {
	return getFilesRecursively(dir, "")
}

func getFilesRecursively(inputDir, filePath string) ([]string, error) {
	var inputFiles []string

	files, err := os.ReadDir(inputDir)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		filename := file.Name()
		newFilePath := path.Join(filePath, filename)
		newDir := path.Join(inputDir, filename)
		if !file.IsDir() {
			inputFiles = append(inputFiles, newFilePath)
		} else {
			newFiles, err := getFilesRecursively(newDir, newFilePath)
			if err != nil {
				return nil, err
			}
			inputFiles = append(inputFiles, newFiles...)

		}
	}

	return inputFiles, nil
}
