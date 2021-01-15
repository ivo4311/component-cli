// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package componentarchive

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Masterminds/semver/v3"
	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	cdvalidation "github.com/gardener/component-spec/bindings-go/apis/v2/validation"
	"github.com/gardener/component-spec/bindings-go/ctf"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"sigs.k8s.io/yaml"

	"github.com/gardener/component-cli/pkg/utils"
)

// InitOptions defines all options for the init command.
type InitOptions struct {
	// ComponentArchivePath defines the path to the component archive to initialize
	ComponentArchivePath string
	ComponentName        string
	ComponentVersion     string
	OCIRegistryURLs      []string
}

// NewInitCommand creates a new init command that creates a base component descriptor
func NewInitCommand(ctx context.Context) *cobra.Command {
	opts := &InitOptions{}
	cmd := &cobra.Command{
		Use:   "init --name <component-name> --version <component-version> --oci-registry <url> <component-archive-path>",
		Args:  cobra.ExactArgs(1),
		Short: "Initializes a component archive as defined by CTF",
		Long: `
Initializes a component archive in the specified path by creating a Component Descriptor file and setting all mandatory fields.

The component-archive add command can then be used to add component references, resources, sources or blobs to the component archive.
`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := opts.Complete(args); err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}
			if err := opts.Run(ctx, osfs.New()); err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}
		},
	}
	opts.AddFlags(cmd.Flags())
	return cmd
}

// Run runs the export for a component archive.
func (o *InitOptions) Run(ctx context.Context, fs vfs.FileSystem) error {
	fileinfo, err := fs.Stat(o.ComponentArchivePath)
	if os.IsNotExist(err) {
		if err := fs.MkdirAll(o.ComponentArchivePath, os.ModeDir); err != nil {
			return fmt.Errorf("unable to mkdir %q", o.ComponentArchivePath)
		}
	} else if err != nil {
		return fmt.Errorf("unable to read %q: %w", o.ComponentArchivePath, err)
	} else if !fileinfo.IsDir() {
		return fmt.Errorf("%q is not a directory", o.ComponentArchivePath)
	}

	// TODO stat component descriptor file

	cd := &cdv2.ComponentDescriptor{
		Metadata: cdv2.Metadata{Version: "v2"},
		ComponentSpec: cdv2.ComponentSpec{
			ObjectMeta: cdv2.ObjectMeta{
				Name:    o.ComponentName,
				Version: o.ComponentVersion,
			},
			Provider: cdv2.InternalProvider,
		},
	}
	for _, url := range o.OCIRegistryURLs {
		cd.RepositoryContexts = utils.AddRepositoryContext(cd.RepositoryContexts, cdv2.OCIRegistryType, url)
	}

	err = cdv2.DefaultComponent(cd)
	if err != nil {
		return fmt.Errorf("unable to set component descriptor defaults: %w", err)
	}

	if err := cdvalidation.Validate(cd); err != nil {
		return fmt.Errorf("invalid resource: %w", err)
	}

	data, err := yaml.Marshal(cd)
	if err != nil {
		return fmt.Errorf("unable to encode component descriptor: %w", err)
	}

	compDescFilePath := filepath.Join(o.ComponentArchivePath, ctf.ComponentDescriptorFileName)
	if err := vfs.WriteFile(fs, compDescFilePath, data, 0644); err != nil {
		return fmt.Errorf("unable to write comonent descriptor: %w", err)
	}

	fmt.Printf("Successfully initialized component archive at %s\n", o.ComponentArchivePath)
	return nil
}

// Complete parses the given command arguments and applies default options.
func (o *InitOptions) Complete(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("expected exactly one argument that contains the path to the component archive")
	}
	o.ComponentArchivePath = args[0]

	return o.validate()
}

func (o *InitOptions) validate() error {
	if len(o.ComponentName) == 0 {
		return fmt.Errorf("component name must not be empty")
	}

	if _, err := semver.NewVersion(o.ComponentVersion); err != nil {
		return fmt.Errorf("component version must be a valid SemVer")
	}

	if len(o.OCIRegistryURLs) == 0 {
		return fmt.Errorf("at least one oci registry must be provided")
	}
	return nil
}

func (o *InitOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.ComponentName, "name", "", "the component name")
	fs.StringVar(&o.ComponentVersion, "version", "", "the component version")
	fs.StringArrayVar(&o.OCIRegistryURLs, "oci-registry", []string{}, "URL of the OCI Registry where the component archive will be stored")
}
