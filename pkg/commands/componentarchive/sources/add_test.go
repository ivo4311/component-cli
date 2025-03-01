// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package sources_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	cdv2 "github.com/gardener/component-spec/bindings-go/apis/v2"
	"github.com/gardener/component-spec/bindings-go/codec"
	"github.com/gardener/component-spec/bindings-go/ctf"
	testlog "github.com/go-logr/logr/testing"
	"github.com/mandelsoft/vfs/pkg/layerfs"
	"github.com/mandelsoft/vfs/pkg/memoryfs"
	"github.com/mandelsoft/vfs/pkg/osfs"
	"github.com/mandelsoft/vfs/pkg/projectionfs"
	"github.com/mandelsoft/vfs/pkg/vfs"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"

	"github.com/gardener/component-cli/pkg/commands/componentarchive/sources"
	"github.com/gardener/component-cli/pkg/componentarchive"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Sources Test Suite")
}

var _ = Describe("Add", func() {

	var testdataFs vfs.FileSystem

	BeforeEach(func() {
		fs, err := projectionfs.New(osfs.New(), "./testdata")
		Expect(err).ToNot(HaveOccurred())
		testdataFs = layerfs.New(memoryfs.New(), fs)
	})

	It("should add a source defined by a file", func() {

		opts := &sources.Options{
			BuilderOptions:   componentarchive.BuilderOptions{ComponentArchivePath: "./00-component"},
			SourceObjectPath: "./resources/00-src.yaml",
		}

		Expect(opts.Run(context.TODO(), testlog.NullLogger{}, testdataFs)).To(Succeed())

		data, err := vfs.ReadFile(testdataFs, filepath.Join(opts.ComponentArchivePath, ctf.ComponentDescriptorFileName))
		Expect(err).ToNot(HaveOccurred())

		cd := &cdv2.ComponentDescriptor{}
		Expect(codec.Decode(data, cd)).To(Succeed())

		Expect(cd.Sources).To(HaveLen(1))
		Expect(cd.Sources[0].IdentityObjectMeta).To(MatchFields(IgnoreExtras, Fields{
			"Name":    Equal("repo"),
			"Version": Equal("v0.0.1"),
			"Type":    Equal("git"),
		}))
	})

	It("should add a source defined by stdin", func() {
		input, err := os.Open("./testdata/resources/00-src.yaml")
		Expect(err).ToNot(HaveOccurred())
		defer input.Close()
		oldstdin := os.Stdin
		defer func() {
			os.Stdin = oldstdin
		}()
		os.Stdin = input

		opts := &sources.Options{
			BuilderOptions: componentarchive.BuilderOptions{ComponentArchivePath: "./00-component"},
		}

		Expect(opts.Run(context.TODO(), testlog.NullLogger{}, testdataFs)).To(Succeed())

		data, err := vfs.ReadFile(testdataFs, filepath.Join(opts.ComponentArchivePath, ctf.ComponentDescriptorFileName))
		Expect(err).ToNot(HaveOccurred())

		cd := &cdv2.ComponentDescriptor{}
		Expect(codec.Decode(data, cd)).To(Succeed())

		Expect(cd.Sources).To(HaveLen(1))
		Expect(cd.Sources[0].IdentityObjectMeta).To(MatchFields(IgnoreExtras, Fields{
			"Name":    Equal("repo"),
			"Version": Equal("v0.0.1"),
			"Type":    Equal("git"),
		}))
	})

	It("should add multiple sources defined by a multi doc file", func() {

		opts := &sources.Options{
			BuilderOptions:   componentarchive.BuilderOptions{ComponentArchivePath: "./00-component"},
			SourceObjectPath: "./resources/01-multi-doc.yaml",
		}

		Expect(opts.Run(context.TODO(), testlog.NullLogger{}, testdataFs)).To(Succeed())

		data, err := vfs.ReadFile(testdataFs, filepath.Join(opts.ComponentArchivePath, ctf.ComponentDescriptorFileName))
		Expect(err).ToNot(HaveOccurred())

		cd := &cdv2.ComponentDescriptor{}
		Expect(codec.Decode(data, cd)).To(Succeed())

		Expect(cd.Sources).To(HaveLen(2))
		Expect(cd.Sources[0].IdentityObjectMeta).To(MatchFields(IgnoreExtras, Fields{
			"Name":    Equal("repo"),
			"Version": Equal("v0.0.1"),
			"Type":    Equal("git"),
		}))
		Expect(cd.Sources[1].IdentityObjectMeta).To(MatchFields(IgnoreExtras, Fields{
			"Name":    Equal("baseRepo"),
			"Version": Equal("v18.4.0"),
			"Type":    Equal("git"),
		}))
	})

	It("should throw an error if an invalid source is defined", func() {
		opts := &sources.Options{
			BuilderOptions:   componentarchive.BuilderOptions{ComponentArchivePath: "./00-component"},
			SourceObjectPath: "./resources/10-invalid.yaml",
		}

		Expect(opts.Run(context.TODO(), testlog.NullLogger{}, testdataFs)).To(HaveOccurred())

		data, err := vfs.ReadFile(testdataFs, filepath.Join(opts.ComponentArchivePath, ctf.ComponentDescriptorFileName))
		Expect(err).ToNot(HaveOccurred())
		cd := &cdv2.ComponentDescriptor{}
		Expect(codec.Decode(data, cd)).To(Succeed())
		Expect(cd.Sources).To(HaveLen(0))
	})

	It("should overwrite the version of a already existing source", func() {

		opts := &sources.Options{
			BuilderOptions:   componentarchive.BuilderOptions{ComponentArchivePath: "./01-component"},
			SourceObjectPath: "./resources/02-overwrite.yaml",
		}

		Expect(opts.Run(context.TODO(), testlog.NullLogger{}, testdataFs)).To(Succeed())

		data, err := vfs.ReadFile(testdataFs, filepath.Join(opts.ComponentArchivePath, ctf.ComponentDescriptorFileName))
		Expect(err).ToNot(HaveOccurred())

		cd := &cdv2.ComponentDescriptor{}
		Expect(codec.Decode(data, cd)).To(Succeed())

		Expect(cd.Sources).To(HaveLen(1))
		Expect(cd.Sources[0].IdentityObjectMeta).To(MatchFields(IgnoreExtras, Fields{
			"Name":    Equal("repo"),
			"Version": Equal("v0.0.2"),
			"Type":    Equal("git"),
		}))
	})

})
