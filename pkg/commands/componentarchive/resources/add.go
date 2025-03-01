// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package resources

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/apis/v2/cdutils"
	cdvalidation "github.com/gardener/component-spec/bindings-go/apis/v2/validation"
	"github.com/gardener/component-spec/bindings-go/ctf"
	"github.com/go-logr/logr"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/util/validation/field"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/yaml"

	"github.com/gardener/component-cli/pkg/commands/componentarchive/input"
	"github.com/gardener/component-cli/pkg/componentarchive"
	"github.com/gardener/component-cli/pkg/logger"
)

// Options defines the options that are used to add resources to a component descriptor
type Options struct {
	componentarchive.BuilderOptions

	// either components can be added by a yaml resource template or by input flags
	// ResourceObjectPath defines the path to the resources defined as yaml or json
	ResourceObjectPath string
}

// ResourceOptions contains options that are used to describe a resource
type ResourceOptions struct {
	cdv2.Resource
	Input *input.BlobInput `json:"input,omitempty"`
}

// NewAddCommand creates a command to add additional resources to a component descriptor.
func NewAddCommand(ctx context.Context) *cobra.Command {
	opts := &Options{}
	cmd := &cobra.Command{
		Use:   "add [component archive path] [-r resource-path]",
		Args:  cobra.RangeArgs(0, 1),
		Short: "Adds a resource to an component archive",
		Long: `
add generates resources from a resource template and adds it to the given component descriptor in the component archive.
If the resource is already defined (quality by identity) in the component-descriptor it will be overwritten.

The component archive can be specified by the first argument, the flag "--archive" or as env var "COMPONENT_ARCHIVE_PATH".
The component archive is expected to be a filesystem archive. If the archive is given as tar please use the export command.

The resource template can be defined by specifying a file with the template with "resource" or it can be given through stdin.

The resource template is a multidoc yaml file so multiple templates can be defined.

<pre>

---
name: 'myimage'
type: 'ociImage'
relation: 'external'
version: 0.2.0
access:
  type: ociRegistry
  imageReference: eu.gcr.io/gardener-project/component-cli:0.2.0
...
---
name: 'myconfig'
type: 'json'
relation: 'local'
input:
  type: "file"
  path: "some/path"
...
---
name: 'myconfig'
type: 'json'
relation: 'local'
input:
  type: "dir"
  path: /my/path
  compress: true # defaults to false
  exclude: "*.txt"
...

</pre>
`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := opts.Complete(args); err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}

			if err := opts.Run(ctx, logger.Log, osfs.New()); err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}
		},
	}

	opts.AddFlags(cmd.Flags())

	return cmd
}

func (o *Options) Run(ctx context.Context, log logr.Logger, fs vfs.FileSystem) error {
	compDescFilePath := filepath.Join(o.ComponentArchivePath, ctf.ComponentDescriptorFileName)

	archive, err := o.BuilderOptions.Build(fs)
	if err != nil {
		return err
	}

	resources, err := o.generateResources(fs, archive.ComponentDescriptor)
	if err != nil {
		return err
	}

	for _, resource := range resources {
		if resource.Input != nil {
			log.Info(fmt.Sprintf("add input blob from %q", resource.Input.Path))
			if err := o.addInputBlob(fs, archive, &resource); err != nil {
				return err
			}
		} else {
			id := archive.ComponentDescriptor.GetResourceIndex(resource.Resource)
			if id != -1 {
				mergedRes := cdutils.MergeResources(archive.ComponentDescriptor.Resources[id], resource.Resource)
				if errList := cdvalidation.ValidateResource(field.NewPath(""), mergedRes); len(errList) != 0 {
					return errList.ToAggregate()
				}
				archive.ComponentDescriptor.Resources[id] = mergedRes
			} else {
				if errList := cdvalidation.ValidateResource(field.NewPath(""), resource.Resource); len(errList) != 0 {
					return errList.ToAggregate()
				}
				archive.ComponentDescriptor.Resources = append(archive.ComponentDescriptor.Resources, resource.Resource)
			}
		}

		if err := cdvalidation.Validate(archive.ComponentDescriptor); err != nil {
			return fmt.Errorf("invalid resource: %w", err)
		}

		data, err := yaml.Marshal(archive.ComponentDescriptor)
		if err != nil {
			return fmt.Errorf("unable to encode component descriptor: %w", err)
		}
		if err := vfs.WriteFile(fs, compDescFilePath, data, 0664); err != nil {
			return fmt.Errorf("unable to write modified comonent descriptor: %w", err)
		}
		log.V(2).Info(fmt.Sprintf("Successfully added %q resource %q %q to component descriptor", resource.Type, resource.Name, resource.Version))
	}
	log.V(2).Info("Successfully added all resources to component descriptor")
	return nil
}

func (o *Options) Complete(args []string) error {
	o.BuilderOptions.Default()
	return o.validate()
}

func (o *Options) validate() error {
	return o.BuilderOptions.Validate()
}

func (o *Options) AddFlags(fs *pflag.FlagSet) {
	o.BuilderOptions.AddFlags(fs)
	// specify the resource
	fs.StringVarP(&o.ResourceObjectPath, "resource", "r", "", "The path to the resources defined as yaml or json")
}

func (o *Options) generateResources(fs vfs.FileSystem, cd *cdv2.ComponentDescriptor) ([]ResourceOptions, error) {
	resources := make([]ResourceOptions, 0)
	if len(o.ResourceObjectPath) != 0 {
		resourceObjectReader, err := fs.Open(o.ResourceObjectPath)
		if err != nil {
			return nil, fmt.Errorf("unable to read resource object from %s: %w", o.ResourceObjectPath, err)
		}
		defer resourceObjectReader.Close()
		resources, err = generateResourcesFromReader(cd, resourceObjectReader)
		if err != nil {
			return nil, fmt.Errorf("unable to read resources from %s: %w", o.ResourceObjectPath, err)
		}
	}

	stdinInfo, err := os.Stdin.Stat()
	if err != nil {
		return nil, fmt.Errorf("unable to read from stdin: %w", err)
	}
	if (stdinInfo.Mode()&os.ModeNamedPipe != 0) || stdinInfo.Size() != 0 {
		stdinResources, err := generateResourcesFromReader(cd, os.Stdin)
		if err != nil {
			return nil, fmt.Errorf("unable to read from stdin: %w", err)
		}
		resources = append(resources, stdinResources...)
	}
	return resources, nil
}

// generateResourcesFromPath generates a resource given resource options and a resource template file.
func generateResourcesFromReader(cd *cdv2.ComponentDescriptor, reader io.Reader) ([]ResourceOptions, error) {
	resources := make([]ResourceOptions, 0)
	yamldecoder := yamlutil.NewYAMLOrJSONDecoder(reader, 1024)
	for {
		resource := ResourceOptions{}
		if err := yamldecoder.Decode(&resource); err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("unable to decode resource: %w", err)
		}

		// automatically set the version to the component descriptors version for local resources
		if resource.Relation == cdv2.LocalRelation && len(resource.Version) == 0 {
			resource.Version = cd.GetVersion()
		}

		if resource.Input != nil && resource.Access != nil {
			return nil, fmt.Errorf("the resources %q input and access is defind. Only one option is allowed", resource.Name)
		}
		resources = append(resources, resource)
	}

	return resources, nil
}

func (o *Options) addInputBlob(fs vfs.FileSystem, archive *ctf.ComponentArchive, resource *ResourceOptions) error {
	blob, err := resource.Input.Read(fs, o.ResourceObjectPath)
	if err != nil {
		return err
	}

	err = archive.AddResource(&resource.Resource, ctf.BlobInfo{
		MediaType: resource.Type,
		Digest:    blob.Digest,
		Size:      blob.Size,
	}, blob.Reader)
	if err != nil {
		blob.Reader.Close()
		return fmt.Errorf("unable to add input blob to archive: %w", err)
	}
	if err := blob.Reader.Close(); err != nil {
		return fmt.Errorf("unable to close input file: %w", err)
	}
	return nil
}
