package uploader

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
)

type GitHubUploadRequest struct {
	Message string `json:"message"`
	Content string `json:"content"`
}

func UploadToGitHub(token, repo, path, filename string) error {
	fileContent, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("error reading file: %v", err)
	}

	uploadURL := fmt.Sprintf("https://api.github.com/repos/%s/contents/%s", repo, path)
	body := GitHubUploadRequest{
		Message: "Update lessons.ics",
		Content: encodeBase64(fileContent),
	}
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("error marshalling JSON: %v", err)
	}

	req, err := http.NewRequest("PUT", uploadURL, bytes.NewBuffer(bodyJSON))
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("error uploading to GitHub, status code: %d, response: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func encodeBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}
