package tknautogenerate

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	gh "github.com/google/go-github/v55/github"
)

func Detect(cli *CliStruct) (string, error) {
	ownerRepo := strings.Split(cli.OwnerRepo, "/")
	if len(ownerRepo) != 2 {
		return "", fmt.Errorf("owner/repo must be specified")
	}
	ctx := context.Background()
	ghC := gh.NewClient(nil)
	if cli.Token != "" {
		ghC = ghC.WithAuthToken(cli.Token)
	}
	detectLanguages, _, err := ghC.Repositories.ListLanguages(ctx, ownerRepo[0], ownerRepo[1])
	if err != nil {
		return "", err
	}

	ag := &AutoGenerate{ghc: ghC, owner: ownerRepo[0], repo: ownerRepo[1], cli: cli}
	if err := ag.New(cli.AutoGenerateYaml); err != nil {
		return "", err
	}

	configs := map[string]Config{}
	for k := range detectLanguages {
		kl := strings.ToLower(k)
		if c, ok := (ag.configs)[kl]; ok {
			kn := kl
			if c.Name != "" {
				kn = c.Name
			}
			configs[kn] = (ag.configs)[kl]
		}
	}

	if cli.PipelineRunYaml != "" {
		ret, err := os.ReadFile(cli.PipelineRunYaml)
		if err != nil {
			return "", fmt.Errorf("failed to read file %s", cli.PipelineRunYaml)
		}
		defaultPipelineRun = ret
	}
	var pipelineRun string
	for k, config := range ag.configs {
		pipelineRun = string(defaultPipelineRun)
		if config.PipelineRun != "" {
			var ret []byte
			var err error
			if cli.TemplatesLanguageDir != "" {
				ret, err = os.ReadFile(filepath.Join(cli.TemplatesLanguageDir, fmt.Sprintf("%s.yaml.go.tmpl", config.PipelineRun)))
			} else {
				ret, err = defaultLanguagesPipelineRuns.ReadFile(filepath.Join("templates", "languages", fmt.Sprintf("%s.yaml.go.tmpl", config.PipelineRun)))
			}
			if err != nil {
				return "", fmt.Errorf("failed to read template %s", pipelineRun)
			}
			pipelineRun = string(ret)
		}
		if config.Pattern == "" {
			continue
		}
		detected, fptasks, err := ag.GetFilePatternTasks(ctx, config)
		if err != nil {
			return "", fmt.Errorf("Error getting file pattern tasks: %w", err)
		}
		if config.Name != "" {
			k = config.Name
		}
		if len(fptasks) != 0 {
			configs[k] = config
		}
		if !detected {
			continue
		}
		// First match when we have a pipelineRun wins
		if config.PipelineRun != "" {
			break
		}
	}
	return ag.Output(configs, pipelineRun)
}
