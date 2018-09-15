package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type ImportConfigOptions struct {
	configFlags *genericclioptions.ConfigFlags

	fromDir             string
	openstackConfigFile string

	genericclioptions.IOStreams
}

var (
	importConfigExample = `
	# import config
	export OPENSTACK_CONFIG_FILE=~/clouds.yaml
	%[1]s import-config --from-dir ~/openstack/tenants/
`
)

func NewCmdImportConfig(streams genericclioptions.IOStreams) *cobra.Command {
	o := &ImportConfigOptions{
		configFlags: genericclioptions.NewConfigFlags(),
		IOStreams:   streams,
	}
	cmd := &cobra.Command{
		Use:          "import-config",
		Aliases:      []string{"rc"},
		Short:        "Import config from OpenStack rc files.",
		Example:      fmt.Sprintf(importConfigExample, "kubectl os"),
		SilenceUsage: true,
		RunE: func(c *cobra.Command, args []string) error {
			if err := o.Complete(c, args); err != nil {
				return err
			}
			if err := o.Validate(); err != nil {
				return err
			}
			if err := o.Run(); err != nil {
				return err
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&o.fromDir, "from-dir", o.fromDir, "dir where the rc files are located")
	o.configFlags.AddFlags(cmd.Flags())
	return cmd
}

func (o *ImportConfigOptions) Complete(cmd *cobra.Command, args []string) error {
	o.openstackConfigFile = os.Getenv("OPENSTACK_CONFIG_FILE")
	return nil
}

func (o *ImportConfigOptions) Validate() error {
	if o.fromDir == "" {
		return fmt.Errorf("--from-dir is mandatory")
	}
	if o.openstackConfigFile == "" {
		return fmt.Errorf("env var 'OPENSTACK_CONFIG_FILE' must be set")
	}
	return nil
}

func (o *ImportConfigOptions) Run() error {

	files, err := ioutil.ReadDir(o.fromDir)
	if err != nil {
		return fmt.Errorf("error reading dir %s: %v", o.fromDir, err)
	}

	usernameRegEx := regexp.MustCompile("OS_USERNAME='(.*)'")
	passwordRegEx := regexp.MustCompile("OS_PASSWORD='(.*)'")
	tenantNameRegEx := regexp.MustCompile("OS_TENANT_NAME=['\"](.*)['\"]")
	authUrlRegEx := regexp.MustCompile("OS_AUTH_URL='(.*)'")

	clouds := clouds{}
	clouds.Clouds = map[string]cloud{}
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".creds") {

			fmt.Printf("Importing config from: %q\n", f.Name())

			content, err := ioutil.ReadFile(path.Join(o.fromDir, f.Name()))
			if err != nil {
				return fmt.Errorf("error reading cred file %s: %v", path.Join(o.fromDir, f.Name()), err)
			}

			usernameMatch := usernameRegEx.FindSubmatch(content)
			passwordMatch := passwordRegEx.FindSubmatch(content)
			tenantNameMatch := tenantNameRegEx.FindSubmatch(content)
			authUrlMatch := authUrlRegEx.FindSubmatch(content)

			if len(usernameMatch) < 1 {
				return fmt.Errorf("error matching username regex")
			}
			if len(passwordMatch) < 1 {
				return fmt.Errorf("error matching username regex")
			}
			if len(tenantNameMatch) < 1 {
				return fmt.Errorf("error matching username regex")
			}
			if len(authUrlMatch) < 1 {
				return fmt.Errorf("error matching username regex")
			}

			clouds.Clouds[string(tenantNameMatch[1])] = cloud{
				Auth: cloudAuth{
					Username:    string(usernameMatch[1]),
					Password:    string(passwordMatch[1]),
					ProjectName: string(tenantNameMatch[1]),
					AuthUrl:     string(authUrlMatch[1]),
				},
			}
		}
	}

	out, err := yaml.Marshal(clouds)
	if err != nil {
		return fmt.Errorf("error marshalling couds: %v", err)
	}

	fmt.Printf("Writing config to %s\n", o.openstackConfigFile)

	err = ioutil.WriteFile(o.openstackConfigFile, out, os.ModePerm)
	if err != nil {
		return fmt.Errorf("error writing config file %s: %v", o.openstackConfigFile, err)
	}
	return nil
}

type clouds struct {
	Clouds map[string]cloud `yaml:"clouds"`
}
type cloud struct {
	Auth cloudAuth `yaml:"auth"`
}

type cloudAuth struct {
	AuthUrl     string `yaml:"auth_url"`
	ProjectName string `yaml:"project_name"`
	Username    string `yaml:"username"`
	Password    string `yaml:"password"`
}
