package tknautogenerate

import (
	"bytes"
	"context"
	"embed"
	_ "embed"
	"fmt"
	"os"
	"regexp"
	"text/template"

	"github.com/google/go-github/v55/github"
	"gopkg.in/yaml.v2"
)

//go:embed templates/tknautogenerate.yaml
var tknAutogenerateYaml []byte

//go:embed templates/pipelinerun.yaml.go.tmpl
var defaultPipelineRun []byte

//go:embed templates/languages/*.tmpl
var defaultLanguagesPipelineRuns embed.FS

type Params struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

type Workspace struct {
	Disabled bool   `yaml:"disabled,omitempty"`
	Name     string `yaml:"name,omitempty"`
}
type Task struct {
	Name      string    `yaml:"name"`
	Params    []Params  `yaml:"params,omitempty"`
	Workspace Workspace `yaml:"workspace,omitempty"`
	RunAfter  []string  `yaml:"runAfter,omitempty"`
}

type Config struct {
	Name        string `yaml:"name"`
	Tasks       []Task `yaml:"tasks"`
	Pattern     string `yaml:"pattern,omitempty"`
	PipelineRun string `yaml:"pipelinerun,omitempty"`
}

type CliStruct struct {
	OwnerRepo            string `arg:"" help:"GitHub owner/repo" placeholder:"OWNER/REPO"`
	Token                string `help:"GitHub token to use" env:"GITHUB_TOKEN" placeholder:"TOKEN" short:"t"`
	TargetRef            string `help:"The target reference when fetching the files (default: main branch)" placeholder:"REF"`
	AutoGenerateYaml     string `help:"Path to the autogenerate.yaml file" placeholder:"FILE" type:"existingfile"`
	PipelineRunYaml      string `help:"path to the default pipelinerun template" placeholder:"FILE" type:"existingfile"`
	TemplatesLanguageDir string `help:"path to the per language templates directory" placeholder:"DIR" type:"existingdir"`
}

type AutoGenerate struct {
	configs       map[string]Config
	ghc           *github.Client
	cli           *CliStruct
	owner, repo   string
	files_in_repo []string
}

func (ag *AutoGenerate) New(filename string) error {
	content := tknAutogenerateYaml
	if filename != "" {
		var err error
		if _, err := os.Stat(filename); err != nil {
			return fmt.Errorf("file %s not found", filename)
		}
		// open file
		if content, err = os.ReadFile(filename); err != nil {
			return fmt.Errorf("failed to open file %s", filename)
		}
	}
	if err := yaml.Unmarshal(content, &ag.configs); err != nil {
		return fmt.Errorf("failed to parse yaml file %s: %w", filename, err)
	}
	return nil
}

func (ag *AutoGenerate) GetAllFilesInRepo(ctx context.Context) ([]string, error) {
	ret := []string{}
	targetRef := ag.cli.TargetRef
	if targetRef == "" {
		info, _, err := ag.ghc.Repositories.Get(ctx, ag.owner, ag.repo)
		if err != nil {
			return ret, err
		}
		targetRef = info.GetDefaultBranch()
	}
	tree, _, err := ag.ghc.Git.GetTree(ctx, ag.owner, ag.repo, targetRef, true)
	if err != nil {
		return ret, err
	}
	for _, entry := range tree.Entries {
		ret = append(ret, entry.GetPath())
	}
	return ret, nil
}

func (ag *AutoGenerate) GetTasks() ([]string, error) {
	var tasks []string
	for _, config := range ag.configs {
		if config.Pattern != "" {
			detected, fptasks, err := ag.GetFilePatternTasks(context.Background(), config)
			if err != nil {
				// TODO: handle error in main
				return []string{}, fmt.Errorf("Error getting file pattern tasks: %w", err)
			}
			if detected {
				tasks = append(tasks, fptasks...)
			}
			continue
		}
		for _, task := range config.Tasks {
			tasks = append(tasks, task.Name)
		}
	}
	return tasks, nil
}

func (ag *AutoGenerate) GetFilePatternTasks(ctx context.Context, config Config) (bool, []string, error) {
	var ret []string
	if ag.files_in_repo == nil {
		var err error
		if ag.files_in_repo, err = ag.GetAllFilesInRepo(ctx); err != nil {
			return false, ret, fmt.Errorf("Error getting all files in repo: %w", err)
		}
	}

	reg, err := regexp.Compile(config.Pattern)
	if err != nil {
		return false, ret, err
	}
	matched := false
	for _, file := range ag.files_in_repo {
		if reg.MatchString(file) {
			matched = true
			break
		}
	}
	if !matched {
		return false, ret, nil
	}
	for _, task := range config.Tasks {
		ret = append(ret, task.Name)
	}
	return true, ret, nil
}

func (ag *AutoGenerate) Output(configs map[string]Config, content string) (string, error) {
	funcMap := template.FuncMap{
		"add": func(a int, b int) int {
			return a + b
		},
	}
	tmpl, err := template.New("pipelineRun").Funcs(funcMap).Parse(content)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	all_tasks, err := ag.GetTasks()
	if err != nil {
		return "", fmt.Errorf("failed to get tasks: %w", err)
	}
	var outputBuffer bytes.Buffer
	data := map[string]interface{}{
		"Configs": configs,
		"Tasks":   all_tasks,
	}
	if err := tmpl.Execute(&outputBuffer, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}
	return outputBuffer.String(), nil
}
