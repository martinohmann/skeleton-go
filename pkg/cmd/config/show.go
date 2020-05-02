package config

import (
	"github.com/spf13/cobra"
	"kickoff.run/pkg/cli"
	"kickoff.run/pkg/cmdutil"
)

func NewShowCmd(streams cli.IOStreams) *cobra.Command {
	o := &ShowOptions{IOStreams: streams}

	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show the kickoff config",
		Long: cmdutil.LongDesc(`
			Show the kickoff config`),
		Example: cmdutil.Examples(`
			# Show the default config
			kickoff config show

			# Show the config using different output
			kickoff config show --output json

			# Show a custom config file
			kickoff config show --config custom-config.yaml`),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := o.Complete(); err != nil {
				return err
			}

			if err := o.Validate(); err != nil {
				return err
			}

			return o.Run()
		},
	}

	cmdutil.AddConfigFlag(cmd, &o.ConfigPath)
	o.OutputFlags.AddFlags(cmd)

	return cmd
}

type ShowOptions struct {
	cli.IOStreams
	cmdutil.ConfigFlags
	cmdutil.OutputFlags
}

func (o *ShowOptions) Run() (err error) {
	switch o.Output {
	case "json":
		return cmdutil.RenderJSON(o.Out, o.Config)
	default:
		return cmdutil.RenderYAML(o.Out, o.Config)
	}
}
