package structs

type Data struct {
	EventType    string `json:"eventType,omitempty"`
	BaseBranch   string `json:"baseBranch,omitempty"`
	HeadBranch   string `json:"headBranch,omitempty"`
	BaseURL      string `json:"baseURL,omitempty"`
	HeadURL      string `json:"headURL,omitempty"`
	SHA          string `json:"sha,omitempty"`
	Organization string `json:"organization,omitempty"`
	Repository   string `json:"repository,omitempty"`
	URL          string `json:"URL,omitempty"`

	// GitHub
	GithubInstallationID int64 `json:"githubInstallationID,omitempty"`

	// GHE
	GHEURL string `json:"gheURL,omitempty"`

	// Bitbucket Cloud
	BitBucketAccountID string `json:"bitBucketAccountID,omitempty"`

	// Bitbucket Server
	BitBucketCloneURL string `json:"bitBucketCloneURL,omitempty"`

	// Gitlab
	GitlabSourceProjectID int `json:"gitlabSourceProjectID,omitempty"`
	GitlabTargetProjectID int `json:"gitlabTargetProjectID,omitempty"`

	// scm providers
	// this info is needed if auto generation is suported for different scm
	Provider string `json:"provider,omitempty"`
}

type PacRequest struct {
	Data  string `json:"data"`
	Token string `json:"token"`
}

type PacResponse struct {
	//PipelineRuns []*v1.PipelineRun `json:"pipelineruns"`
	PipelineRuns string `json:"pipelineruns"`
}
