package project

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/martinohmann/kickoff/internal/cli"
	"github.com/martinohmann/kickoff/internal/cmdutil"
	"github.com/martinohmann/kickoff/internal/file"
	"github.com/martinohmann/kickoff/internal/git"
	"github.com/martinohmann/kickoff/internal/gitignore"
	"github.com/martinohmann/kickoff/internal/kickoff"
	"github.com/martinohmann/kickoff/internal/license"
	"github.com/martinohmann/kickoff/internal/project"
	"github.com/martinohmann/kickoff/internal/repository"
	"github.com/martinohmann/kickoff/internal/template"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"helm.sh/helm/pkg/strvals"
)

// NewCreateCmd creates a command that can create projects from project
// skeletons using a variety of user-defined options.
func NewCreateCmd(f *cmdutil.Factory) *cobra.Command {
	o := &CreateOptions{
		IOStreams:  f.IOStreams,
		Config:     f.Config,
		HTTPClient: f.HTTPClient,
		Repository: f.Repository,
		GitClient:  git.NewClient(),
	}

	cmd := &cobra.Command{
		Use:   "create <skeleton-name> <output-dir>",
		Short: "Create a project from a skeleton",
		Long: cmdutil.LongDesc(`
			Create a project from a skeleton.`),
		Example: cmdutil.Examples(`
			# Create project
			kickoff project create myskeleton ~/repos/myproject

			# Create project from skeleton in specific repo
			kickoff project create myrepo:myskeleton ~/repos/myproject

			# Create project with license
			kickoff project create myskeleton ~/repos/myproject --license mit

			# Create project with gitignore
			kickoff project create myskeleton ~/repos/myproject --gitignore go,helm,hugo

			# Create project with value overrides from files
			kickoff project create myskeleton ~/repos/myproject --values values.yaml --values values2.yaml

			# Create project with value overrides via --set
			kickoff project create myskeleton ~/repos/myproject --set travis.enabled=true,mykey=mynewvalue

			# Dry run project creation
			kickoff project create myskeleton ~/repos/myproject --dry-run

			# Composition of multiple skeletons (comma separated)
			kickoff project create firstskeleton,secondskeleton,thirdskeleton ~/repos/myproject

			# Forces creation of project in existing directory, retaining existing files
			kickoff project create myskeleton ~/repos/myproject --force

			# Forces creation of project in existing directory, overwriting existing files
			kickoff project create myskeleton ~/repos/myproject --force --overwrite

			# Forces creation of project in existing directory, selectively overwriting existing files
			kickoff project create myskeleton ~/repos/myproject --force --overwrite-file README.md

			# Selectively skip the creating of certain files or dirs
			kickoff project create myskeleton ~/repos/myproject --skip-file README.md`),
		Args: cmdutil.ExactNonEmptyArgs(2),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				return cmdutil.SkeletonNames(f), cobra.ShellCompDirectiveDefault
			}
			return nil, cobra.ShellCompDirectiveFilterDirs
		},
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			o.SkeletonNames = strings.Split(args[0], ",")

			o.ProjectDir, err = filepath.Abs(args[1])
			if err != nil {
				return err
			}

			if err := o.Complete(); err != nil {
				return err
			}

			return o.Run()
		},
	}

	cmdutil.AddRepositoryFlag(cmd, f, &o.RepoNames)

	o.AddFlags(cmd)

	cmd.RegisterFlagCompletionFunc("license", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return cmdutil.LicenseNames(f), cobra.ShellCompDirectiveDefault
	})
	cmd.RegisterFlagCompletionFunc("gitignore", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return cmdutil.GitignoreNames(f), cobra.ShellCompDirectiveDefault
	})

	return cmd
}

// CreateOptions holds the options for the create command.
type CreateOptions struct {
	cli.IOStreams

	Config     func() (*kickoff.Config, error)
	HTTPClient func() *http.Client
	Repository func(...string) (kickoff.Repository, error)

	GitClient git.Client

	ProjectName  string
	ProjectDir   string
	ProjectHost  string
	ProjectOwner string
	License      string
	Gitignore    string
	Values       template.Values

	RepoNames      []string
	SkeletonNames  []string
	DryRun         bool
	Force          bool
	Overwrite      bool
	OverwriteFiles []string
	SkipFiles      []string
	InitGit        bool

	rawValues   []string
	valuesFiles []string
}

// AddFlags adds flags for all project creation options to cmd.
func (o *CreateOptions) AddFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&o.DryRun, "dry-run", o.DryRun, "Only print what would be done")
	cmd.Flags().BoolVar(&o.Force, "force", o.Force, "Forces writing into existing output directory")
	cmd.Flags().BoolVar(&o.InitGit, "init-git", o.InitGit, "Initialize git in the project directory")
	cmd.Flags().BoolVar(&o.Overwrite, "overwrite", o.Overwrite, "Overwrite files that are already present in output directory")

	cmd.Flags().StringArrayVar(&o.OverwriteFiles, "overwrite-file", o.OverwriteFiles,
		"Overwrite a specific file in the output directory, if present. File path must be relative to the output directory. "+
			"If file is a dir, present files contained in it will be overwritten")
	cmd.Flags().StringArrayVar(&o.SkipFiles, "skip-file", o.SkipFiles,
		"Skip writing a specific file to the output directory. File path must be relative to the output directory. "+
			"If file is a dir, files contained in it will be skipped as well")
	cmd.Flags().StringArrayVar(&o.rawValues, "set", o.rawValues,
		"Set custom values of the form key1=value1,key2=value2,deeply.nested.key3=value that are then made available to .skel templates")
	cmd.Flags().StringArrayVar(&o.valuesFiles, "values", o.valuesFiles,
		"Load custom values from provided file, making them available to .skel templates. Values passed via --set take precedence")
	cmd.Flags().StringVar(&o.Gitignore, "gitignore", o.Gitignore,
		"Comma-separated list of gitignore template to use for the project. If set this will automatically populate the .gitignore file")

	cmd.Flags().StringVar(&o.License, "license", o.License, "License to use for the project. If set this will automatically populate the LICENSE file")
	cmd.Flags().StringVar(&o.ProjectHost, "host", o.ProjectHost, "Project repository host")
	cmd.Flags().StringVar(&o.ProjectName, "name", o.ProjectName, "Name of the project. Will be inferred from the output dir if not explicitly set")
	cmd.Flags().StringVar(&o.ProjectOwner, "owner", o.ProjectOwner, "Project repository owner. This should be the name of the SCM user, e.g. the GitHub user or organization name")
}

// Complete completes the project creation options.
func (o *CreateOptions) Complete() (err error) {
	if file.Exists(o.ProjectDir) && !o.Force {
		return fmt.Errorf("project dir %s already exists, add --force to overwrite", o.ProjectDir)
	}

	config, err := o.Config()
	if err != nil {
		return err
	}

	if o.ProjectName == "" {
		o.ProjectName = filepath.Base(o.ProjectDir)
	}

	if o.ProjectHost == "" {
		o.ProjectHost = config.Project.Host
	}

	if o.ProjectOwner == "" {
		o.ProjectOwner = config.Project.Owner
	}

	if o.ProjectOwner == "" {
		return errors.New("--owner needs to be set as it could not be inferred")
	}

	if o.License == "" {
		o.License = config.Project.License
	}

	if o.Gitignore == "" {
		o.Gitignore = config.Project.Gitignore
	}

	o.Values = config.Values

	if len(o.valuesFiles) > 0 {
		for _, path := range o.valuesFiles {
			vals, err := template.LoadValues(path)
			if err != nil {
				return err
			}

			err = o.Values.Merge(vals)
			if err != nil {
				return err
			}
		}
	}

	if len(o.rawValues) > 0 {
		for _, rawValues := range o.rawValues {
			err = strvals.ParseInto(rawValues, o.Values)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// Run loads all project skeletons that the user provided and creates the
// project at the output directory.
func (o *CreateOptions) Run() error {
	repo, err := o.Repository(o.RepoNames...)
	if err != nil {
		return err
	}

	skeletons, err := repository.LoadSkeletons(repo, o.SkeletonNames)
	if err != nil {
		return err
	}

	skeleton, err := kickoff.MergeSkeletons(skeletons...)
	if err != nil {
		return err
	}

	err = o.createProject(context.Background(), skeleton)
	if err != nil || !o.InitGit {
		return err
	}

	return o.initGitRepository(o.ProjectDir)
}

func (o *CreateOptions) createProject(ctx context.Context, s *kickoff.Skeleton) error {
	config := &project.Config{
		ProjectName:    o.ProjectName,
		Host:           o.ProjectHost,
		Owner:          o.ProjectOwner,
		Overwrite:      o.Overwrite,
		OverwriteFiles: o.OverwriteFiles,
		SkipFiles:      o.SkipFiles,
		Values:         o.Values,
		Output:         o.Out,
	}

	if o.License != "" && o.License != kickoff.NoLicense {
		client := license.NewClient(o.HTTPClient())

		license, err := client.GetLicense(ctx, o.License)
		if err != nil {
			return err
		}

		config.License = license
	}

	if o.Gitignore != "" && o.Gitignore != kickoff.NoGitignore {
		client := gitignore.NewClient(o.HTTPClient())

		template, err := client.GetTemplate(ctx, o.Gitignore)
		if err != nil {
			return err
		}

		config.Gitignore = template
	}

	if o.DryRun {
		config.Filesystem = afero.NewMemMapFs()
		fmt.Fprintf(o.Out, "%s changes will not be persisted to disk\n\n", color.YellowString("dry-run:"))
	}

	result, err := project.Create(s, o.ProjectDir, config)
	if err != nil {
		return err
	}

	if result.Stats[project.ActionTypeSkipExisting] > 0 {
		fmt.Fprintln(o.Out, "\nSome targets were skipped because they already existed, use --overwrite or --overwrite-file to overwrite")
	}

	return nil
}

func (o *CreateOptions) initGitRepository(path string) error {
	log.WithField("path", path).Debug("initializing git repository")

	if !o.DryRun {
		_, err := o.GitClient.Init(path)
		if err != nil && err != git.ErrRepositoryAlreadyExists {
			return err
		}
	}

	return nil
}
