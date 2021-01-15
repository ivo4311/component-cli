package componentarchive_test

import (
	"context"

	"github.com/ghodss/yaml"
	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/ctf"

	"github.com/gardener/component-cli/pkg/commands/componentarchive"
)

var _ = Describe("Init", func() {

	var testdataFs vfs.FileSystem

	BeforeEach(func() {
		testdataFs = memoryfs.New()
	})

	It("should create a component descriptor with name, version and oci registry context", func() {
		opts := &componentarchive.InitOptions{
			ComponentArchivePath: "",
			ComponentName:        "component-name",
			ComponentVersion:     "1.0.0",
			OCIRegistryURLs:      []string{"my.oci.registry"},
		}

		Expect(opts.Run(context.TODO(), testdataFs)).To(Succeed())

		data, err := vfs.ReadFile(testdataFs, ctf.ComponentDescriptorFileName)
		Expect(err).ToNot(HaveOccurred())

		var actualCD cdv2.ComponentDescriptor
		err = yaml.Unmarshal(data, &actualCD)
		Expect(err).ToNot(HaveOccurred())

		expectedCD := cdv2.ComponentDescriptor{
			Metadata: cdv2.Metadata{Version: "v2"},
			ComponentSpec: cdv2.ComponentSpec{
				ObjectMeta: cdv2.ObjectMeta{
					Name:    "component-name",
					Version: "1.0.0",
				},
				Provider:            cdv2.InternalProvider,
				Sources:             []cdv2.Source{},
				Resources:           []cdv2.Resource{},
				ComponentReferences: []cdv2.ComponentReference{},
				RepositoryContexts: []cdv2.RepositoryContext{
					{Type: cdv2.OCIRegistryType, BaseURL: "my.oci.registry"}},
			},
		}

		Expect(actualCD).To(Equal(expectedCD))
	})
})
