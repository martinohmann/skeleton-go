package config

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/apex/log"
	"github.com/ghodss/yaml"
	"github.com/kirsle/configdir"
	"github.com/martinohmann/skeleton-go/pkg/file"
	"github.com/spf13/cobra"
	gitconfig "github.com/tcnksm/go-gitconfig"
	"helm.sh/helm/pkg/strvals"
)

const (
	DefaultSkeleton = "default"
)

var (
	DefaultSkeletonsDir = configdir.LocalConfig("skeleton-go", "skeletons")
	DefaultConfigPath   = configdir.LocalConfig("skeleton-go", "config.yaml")

	SkeletonConfigFile = ".skeleton-go.yaml"
)

type Config struct {
	ProjectName  string
	License      string
	Author       AuthorConfig
	Repository   RepositoryConfig
	Skeleton     string
	SkeletonsDir string
	Custom       map[string]interface{}

	rawCustomValues []string
}

func NewDefaultConfig() *Config {
	return &Config{
		Skeleton: DefaultSkeleton,
	}
}

func (c *Config) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&c.Author.Fullname, "author-fullname", c.Author.Fullname, "Project author's fullname")
	cmd.Flags().StringVar(&c.Author.Email, "author-email", c.Author.Email, "Project author's e-mail")
	cmd.Flags().StringVar(&c.ProjectName, "project-name", c.ProjectName, "Name of the project. Will be inferred from the output dir if not explicitly set")
	cmd.Flags().StringVar(&c.License, "license", c.License, "License to use for the project. If set this will automatically populate the LICENSE file")
	cmd.Flags().StringVar(&c.Skeleton, "skeleton", c.Skeleton, "Name of the skeleton to create the project from")
	cmd.Flags().StringVar(&c.SkeletonsDir, "skeletons-dir", c.SkeletonsDir, fmt.Sprintf("Path to the skeletons directory. (defaults to %q if the directory exists)", DefaultSkeletonsDir))
	cmd.Flags().StringVar(&c.Repository.User, "repository-user", c.Repository.User, "Repository username")
	cmd.Flags().StringVar(&c.Repository.Name, "repository-name", c.Repository.Name, "Repository name (defaults to the project name)")
	cmd.Flags().StringArrayVar(&c.rawCustomValues, "set", c.rawCustomValues, "Set custom config values of the form key1=value1,key2=value2,deeply.nested.key3=value")
}

func (c *Config) SkeletonDir() string {
	return filepath.Join(c.SkeletonsDir, c.Skeleton)
}

func (c *Config) Complete(outputDir string) (err error) {
	if c.Author.Fullname == "" {
		c.Author.Fullname, err = gitconfig.Global("user.name")
		if err != nil {
			log.Warn("user.name not found in git config, set it to automatically populate author fullname")
		}
	}

	if c.Author.Email == "" {
		c.Author.Email, err = gitconfig.Global("user.email")
		if err != nil {
			log.Warn("user.email not found in git config, set it to automatically populate author email")
		}
	}

	if c.Repository.User == "" {
		c.Repository.User, err = gitconfig.Global("github.user")
		if err != nil {
			log.Warn("github.user not found in git config, set it to automatically populate repository user")
		}
	}

	if c.ProjectName == "" {
		c.ProjectName = filepath.Base(outputDir)
	}

	if c.Repository.Name == "" {
		c.Repository.Name = c.ProjectName
	}

	if c.Skeleton == "" {
		c.Skeleton = DefaultSkeleton
	}

	if c.SkeletonsDir == "" && file.Exists(DefaultSkeletonsDir) {
		c.SkeletonsDir = DefaultSkeletonsDir
	}

	if c.SkeletonsDir != "" {
		c.SkeletonsDir, err = filepath.Abs(c.SkeletonsDir)
		if err != nil {
			return err
		}
	}

	if len(c.rawCustomValues) > 0 {
		for _, rawValues := range c.rawCustomValues {
			err = strvals.ParseInto(rawValues, c.Custom)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *Config) Validate() error {
	if c.Skeleton == "" {
		return fmt.Errorf("--skeleton must be provided")
	}

	if c.SkeletonsDir == "" {
		return fmt.Errorf("--skeletons-dir must be provided")
	}

	if c.Repository.User == "" {
		return fmt.Errorf("--repository-user needs to be set")
	}

	return nil
}

type AuthorConfig struct {
	Fullname string
	Email    string
}

func (c AuthorConfig) String() string {
	return fmt.Sprintf("%s <%s>", c.Fullname, c.Email)
}

type RepositoryConfig struct {
	User string
	Name string
}

func Load(filePath string) (*Config, error) {
	buf, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var config Config

	err = yaml.Unmarshal(buf, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
