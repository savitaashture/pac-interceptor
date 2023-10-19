package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	githttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/savitaashture/pac-interceptor/autogenerate"
	"github.com/savitaashture/pac-interceptor/structs"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/storage/memory"
	"sigs.k8s.io/yaml"
)

func main() {
	log.Println("Attempting to start HTTP Server.")
	mux := http.NewServeMux()
	mux.HandleFunc("/pac-interceptor", handleRequest)
	if err := http.ListenAndServe(":8800", mux); err != nil {
		log.Panicln("Failed to start server. Error: %s", err)
	}
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	defer func(Body io.ReadCloser) {
		if err := Body.Close(); err != nil {
			handleError(&w, 500, "Internal Server Error", "Error in closing the body", err)
			return
		}
	}(r.Body)
	byteData, err := io.ReadAll(r.Body)
	if err != nil {
		handleError(&w, 500, "Internal Server Error", "Error reading data from body", err)
		return
	}

	request := structs.PacRequest{}
	if err = json.Unmarshal(byteData, &request); err != nil {
		handleError(&w, 500, "Internal Server Error", "Error unmarshalling JSON", err)
		return
	}

	handleSuccess(&w, request)
}

func handleSuccess(w *http.ResponseWriter, request structs.PacRequest) {
	writer := *w
	response := structs.PacResponse{}

	payloadData := structs.Data{}
	if err := decodeFromBase64(&payloadData, request.Data); err != nil {
		handleError(w, http.StatusInternalServerError, "Internal Server Error", "Error decoding data string", err)
		return
	}

	pipelinerun, err := clone(payloadData, request.Token)
	if err != nil {
		handleError(w, http.StatusInternalServerError, "Internal Server Error", "Error cloning", err)
		return
	}
	response.PipelineRuns = pipelinerun
	responseMarshalled, err := json.Marshal(response)
	if err != nil {
		handleError(w, http.StatusInternalServerError, "Internal Server Error", "Error marshalling response JSON", err)
		return
	}

	writer.Header().Add("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	if _, err = writer.Write(responseMarshalled); err != nil {
		handleError(w, http.StatusInternalServerError, "Internal Server Error", "Error writing response JSON", err)
		return
	}
}

func handleError(w *http.ResponseWriter, code int, responseText string, logMessage string, err error) {
	errorMessage := ""
	if err != nil {
		errorMessage = err.Error()
	}

	log.Println(logMessage, errorMessage)
	writer := *w
	writer.WriteHeader(code)
	writer.Write([]byte(responseText))
}

func decodeFromBase64(v interface{}, enc string) error {
	return json.NewDecoder(base64.NewDecoder(base64.StdEncoding, strings.NewReader(enc))).Decode(v)
}

func clone(payloadData structs.Data, token string) ([]*v1.PipelineRun, error) {
	fmt.Println("payloadData", payloadData.GithubOrganization, "***************", payloadData.GithubRepository)
	repo, err := git.Clone(memory.NewStorage(), nil, &git.CloneOptions{
		URL: fmt.Sprintf("https://github.com/%s/%s", payloadData.GithubOrganization, payloadData.GithubRepository),
		Auth: &githttp.BasicAuth{
			Username: "abc123", // yes, this can be anything except an empty string
			Password: token,
		},
		ReferenceName: plumbing.NewBranchReferenceName(payloadData.HeadBranch),
		Progress:      os.Stdout,
	})
	if err != nil {
		return nil, err
	}

	ref, err := repo.Head()
	if err != nil {
		return nil, err
	}

	// ... retrieving the commit object
	commit, err := repo.CommitObject(ref.Hash())
	if err != nil {
		return nil, err
	}

	tree, err := commit.Tree()
	if err != nil {
		return nil, err
	}

	var prs []*v1.PipelineRun
	var p v1.PipelineRun
	tektontree, err := tree.Tree(".tekton")
	if err != nil {
		if strings.Contains(err.Error(), "directory not found") {
			// call autogenerate library
			var cliStruct = &autogenerate.CliStruct{
				OwnerRepo: payloadData.GithubOrganization + "/" + payloadData.GithubRepository,
				Token:     token,
				TargetRef: payloadData.BaseBranch,
			}
			f, err := autogenerate.Detect(cliStruct)
			if err != nil {
				return nil, err
			}
			if err = yaml.Unmarshal([]byte(f), &p); err != nil {
				return nil, err
			}
			prs = append(prs, &p)
			return prs, nil
		}
		return nil, err
	}
	tektontree.Files().ForEach(func(f *object.File) error {
		if strings.HasSuffix(f.Name, "yaml") {
			filecontent, err := f.Contents()
			if err != nil {
				return err
			}
			if err = yaml.Unmarshal([]byte(filecontent), &p); err != nil {
				return err
			}
			prs = append(prs, &p)
		}
		return nil
	})
	for _, pr := range prs {
		pr.Name = "test-pac-interceptor-" + pr.Name
	}
	return prs, nil
}
