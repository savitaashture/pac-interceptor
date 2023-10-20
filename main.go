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
	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/storage/memory"
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
	fmt.Println("PPPPPPPPPPPPPPPP", pipelinerun)
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

// func clone(payloadData structs.Data, token string) ([]*v1.PipelineRun, error) {
func clone(payloadData structs.Data, token string) (string, error) {
	fmt.Println("payloadData", payloadData.GithubOrganization, "***************", payloadData.GithubRepository, "eveveveve", payloadData.EventType)
	repo, err := git.Clone(memory.NewStorage(), nil, &git.CloneOptions{
		URL: fmt.Sprintf("https://github.com/%s/%s", payloadData.GithubOrganization, payloadData.GithubRepository),
		Auth: &githttp.BasicAuth{
			Username: "abc123", // yes, this can be anything except an empty string
			Password: token,
		},
		ReferenceName: plumbing.NewBranchReferenceName(payloadData.HeadBranch),
		Progress:      os.Stdout,
	})
	fmt.Println("erorororor in clone", err)
	if err != nil {
		return "", err
	}

	ref, err := repo.Head()
	fmt.Println("erorororor in head", err)
	if err != nil {
		return "", err
	}

	// ... retrieving the commit object
	commit, err := repo.CommitObject(ref.Hash())
	fmt.Println("erorororor in hash", err)
	if err != nil {
		return "", err
	}

	tree, err := commit.Tree()
	fmt.Println("erorororor in tree", err)
	if err != nil {
		return "", err
	}

	//var prs []*v1.PipelineRun
	//var p v1.PipelineRun

	_, err = tree.Tree(".tekton")
	//tektontree, err := tree.Tree(".tekton")
	fmt.Println("erorororor in get tekton tree", err)
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
				return "", err
			}
			//if err = yaml.Unmarshal([]byte(f), &p); err != nil {
			//	return nil, err
			//}
			//prs = append(prs, &p)
			//return prs, nil
			fmt.Println("value o ffff", f)
			return f, nil
		}
		return "", err
	}

	//var finaldata string
	//tektontree.Files().ForEach(func(f *object.File) error {
	//	if strings.HasSuffix(f.Name, "yaml") || strings.HasSuffix(f.Name, "yml") {
	//		filecontent, err := f.Contents()
	//		if err != nil {
	//			return err
	//		}
	//		if !strings.HasPrefix(filecontent, "---") {
	//			finaldata += "---"
	//		}
	//		finaldata += "\n" + filecontent + "\n"
	//		//if err = yaml.Unmarshal([]byte(filecontent), &p); err != nil {
	//		//	return err
	//		//}
	//		//prs = append(prs, &p)
	//	}
	//	return nil
	//})
	////for _, pr := range prs {
	////	pr.Name = "test-pac-interceptor-" + pr.Name
	////}
	//return finaldata, nil
	return fmt.Sprintf("https://github.com/%s/%s have .tekton directory", payloadData.GithubOrganization, payloadData.GithubRepository), nil
}
