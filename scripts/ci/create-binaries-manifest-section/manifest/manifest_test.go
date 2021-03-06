package manifest_test

import (
	"errors"
	"io/ioutil"
	"regexp"

	"github.com/cloudfoundry/runtime-ci/scripts/ci/create-binaries-manifest-section/manifest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("UpdateReleasesAndStemcells", func() {
	var (
		updatedReleasesAndStemcellsFixture []byte
		cfDeploymentManifest               []byte
	)

	BeforeEach(func() {
		var err error

		updatedReleasesAndStemcellsFixture, err = ioutil.ReadFile("fixtures/updated_releases_and_stemcells.yml")
		Expect(err).NotTo(HaveOccurred())

		cfDeploymentManifest, err = ioutil.ReadFile("fixtures/cf-deployment.yml")
		Expect(err).NotTo(HaveOccurred())

		manifest.ResetYAMLMarshal()
	})

	It("updates the releases and stemcells without modifying the rest and returns the list of changes", func() {
		releases := []string{"release1", "release2"}
		buildDir := "fixtures/build"

		updatedManifest, changes, err := manifest.UpdateReleasesAndStemcells(releases, buildDir, cfDeploymentManifest)
		Expect(err).NotTo(HaveOccurred())

		r := regexp.MustCompile(`(?m:^releases:$)`)
		updatedManifestReleasesIndex := r.FindSubmatchIndex([]byte(updatedManifest))[0]
		cfDeploymentManifestReleasesIndex := r.FindSubmatchIndex([]byte(cfDeploymentManifest))[0]
		cfDeploymentPreamble := cfDeploymentManifest[:cfDeploymentManifestReleasesIndex]
		updatedManifestPreamble := updatedManifest[:updatedManifestReleasesIndex]

		updatedManifestReleasesAndStemcells := updatedManifest[updatedManifestReleasesIndex:]

		Expect(string(cfDeploymentPreamble)).To(Equal(string(updatedManifestPreamble)), "the preamble was changed by running the program")
		Expect(string(updatedManifestReleasesAndStemcells)).To(Equal(string(updatedReleasesAndStemcellsFixture)))

		Expect(changes).To(Equal("Updated release2-release, ubuntu-trusty stemcell"))
	})

	It("provides a default commit message if no version updates were performed", func() {
		releases := []string{"release1", "release2"}
		buildDir := "fixtures/nochanges-build"

		_, changes, err := manifest.UpdateReleasesAndStemcells(releases, buildDir, cfDeploymentManifest)
		Expect(err).NotTo(HaveOccurred())

		Expect(changes).To(Equal("No release or stemcell version updates"))
	})

	It("adds all the releases and stemcells to the commit message if no releases exist", func() {
		releases := []string{"release1", "release2"}
		buildDir := "fixtures/nochanges-build"

		cfDeploymentManifest := []byte(`
name: my-deployment
releases:
stemcells:
`)

		_, changes, err := manifest.UpdateReleasesAndStemcells(releases, buildDir, cfDeploymentManifest)
		Expect(err).NotTo(HaveOccurred())

		Expect(changes).To(Equal("Updated release1-release, release2-release, ubuntu-trusty stemcell"))
	})

	Context("failure cases", func() {
		It("ensures there is a releases key at the bottom of the manifest", func() {
			releases := []string{"release1", "release2"}
			buildDir := "fixtures/build"

			badManifest := []byte(`
name:
stemcells:
other_key:
`)
			_, _, err := manifest.UpdateReleasesAndStemcells(releases, buildDir, badManifest)
			Expect(err).To(MatchError("releases was not found at the bottom of the manifest"))
		})

		It("ensures there is a stemcell key at the bottom of the manifest", func() {
			releases := []string{"release1", "release2"}
			buildDir := "fixtures/build"

			badManifest := []byte(`
name:
stemcells:
releases:
other_key:
`)
			_, _, err := manifest.UpdateReleasesAndStemcells(releases, buildDir, badManifest)
			Expect(err).To(MatchError("stemcells was not found at the bottom of the manifest"))
		})

		It("returns an error when there are keys other than release and stemcells", func() {
			releases := []string{"release1", "release2"}
			buildDir := "fixtures/build"

			badManifest := []byte(`
name:
releases:
stemcells:
other_key:
`)
			_, _, err := manifest.UpdateReleasesAndStemcells(releases, buildDir, badManifest)
			Expect(err).To(MatchError(`found keys other than "releases" and "stemcells" at the bottom of the manifest`))
		})

		It("returns errors instead of panicking when url is missing", func() {
			releases := []string{"missing-url"}
			buildDir := "fixtures/broken-build"

			_, _, err := manifest.UpdateReleasesAndStemcells(releases, buildDir, cfDeploymentManifest)

			Expect(err).To(MatchError("open fixtures/broken-build/missing-url-release/url: no such file or directory"))
		})

		It("returns errors instead of panicking when version is missing", func() {
			releases := []string{"missing-version"}
			buildDir := "fixtures/broken-build"

			_, _, err := manifest.UpdateReleasesAndStemcells(releases, buildDir, cfDeploymentManifest)

			Expect(err).To(MatchError("open fixtures/broken-build/missing-version-release/version: no such file or directory"))
		})

		It("returns errors instead of panicking when sha1 is missing", func() {
			releases := []string{"missing-sha1"}
			buildDir := "fixtures/broken-build"

			_, _, err := manifest.UpdateReleasesAndStemcells(releases, buildDir, cfDeploymentManifest)

			Expect(err).To(MatchError("open fixtures/broken-build/missing-sha1-release/sha1: no such file or directory"))
		})

		It("returns errors instead of panicking when sha1 is missing", func() {
			releases := []string{"good-release"}
			buildDir := "fixtures/broken-build"

			_, _, err := manifest.UpdateReleasesAndStemcells(releases, buildDir, cfDeploymentManifest)

			Expect(err).To(MatchError("open fixtures/broken-build/stemcell/version: no such file or directory"))
		})

		It("returns an error when the manifest is not valid yaml", func() {
			releases := []string{"release1", "release2"}
			buildDir := "fixtures/build"

			cfDeploymentManifest := []byte(`
%%%
releases:
%%%
`)
			_, _, err := manifest.UpdateReleasesAndStemcells(releases, buildDir, cfDeploymentManifest)
			Expect(err).To(MatchError(ContainSubstring("could not find expected directive name")))
		})

		It("returns an error when the releases section is malformed", func() {
			releases := []string{"release1", "release2"}
			buildDir := "fixtures/build"

			cfDeploymentManifest := []byte(`
name: my-deployment
releases:
- wrong type
stemcells:
- alias: my-stemcell
`)

			_, _, err := manifest.UpdateReleasesAndStemcells(releases, buildDir, cfDeploymentManifest)
			Expect(err).To(MatchError(ContainSubstring("`wrong type` into manifest.Release")))
		})

		It("returns an error when the stemcells section is malformed", func() {
			releases := []string{"release1", "release2"}
			buildDir := "fixtures/build"

			cfDeploymentManifest := []byte(`
name: my-deployment
releases:
- name: my-release
stemcells:
- wrong type
`)

			_, _, err := manifest.UpdateReleasesAndStemcells(releases, buildDir, cfDeploymentManifest)
			Expect(err).To(MatchError(ContainSubstring("`wrong type` into manifest.Stemcell")))
		})

		It("returns an error when the yaml marshaller fails", func() {
			manifest.SetYAMLMarshal(func(interface{}) ([]byte, error) {
				return nil, errors.New("failed to marshal yaml")
			})
			releases := []string{"release1", "release2"}
			buildDir := "fixtures/build"

			_, _, err := manifest.UpdateReleasesAndStemcells(releases, buildDir, cfDeploymentManifest)
			Expect(err).To(MatchError("failed to marshal yaml"))
		})
	})
})
