package eirinistaging_test

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"

	"code.cloudfoundry.org/eirini-staging"
)

var _ = Describe("Buildpackmanager", func() {

	var (
		client           *http.Client
		buildpackDir     string
		buildpacksJSON   []byte
		buildpackManager eirinistaging.Installer
		buildpacks       []eirinistaging.Buildpack
		server           *ghttp.Server
		responseContent  []byte
		err              error
	)

	BeforeEach(func() {
		client = http.DefaultClient

		buildpackDir, err = ioutil.TempDir("", "buildpacks")
		Expect(err).ToNot(HaveOccurred())

		responseContent, err = makeZippedPackage()
		Expect(err).ToNot(HaveOccurred())

		server = ghttp.NewServer()
		server.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest("GET", "/my-buildpack"),
				ghttp.RespondWith(http.StatusOK, responseContent),
			),
			ghttp.CombineHandlers(
				ghttp.VerifyRequest("GET", "/your-buildpack"),
				ghttp.RespondWith(http.StatusOK, responseContent),
			),
		)

	})

	JustBeforeEach(func() {
		buildpacksJSON, err = json.Marshal(buildpacks)
		Expect(err).NotTo(HaveOccurred())

		buildpackManager = eirinistaging.NewBuildpackManager(client, client, buildpackDir, string(buildpacksJSON))
		err = buildpackManager.Install()
	})

	Context("When a list of Buildpacks needs be installed", func() {
		BeforeEach(func() {
			buildpacks = []eirinistaging.Buildpack{
				{
					Name: "my_buildpack",
					Key:  "my-key",
					URL:  fmt.Sprintf("%s/my-buildpack", server.URL()),
				},
				{
					Name: "your_buildpack",
					Key:  "your-key",
					URL:  fmt.Sprintf("%s/your-buildpack", server.URL()),
				},
			}
		})

		It("should not fail", func() {
			Expect(err).ToNot(HaveOccurred())
		})

		It("should download all buildpacks to the given directory", func() {
			myMd5Dir := fmt.Sprintf("%x", md5.Sum([]byte("my_buildpack")))
			yourMd5Dir := fmt.Sprintf("%x", md5.Sum([]byte("your_buildpack")))
			Expect(filepath.Join(buildpackDir, myMd5Dir)).To(BeADirectory())
			Expect(filepath.Join(buildpackDir, yourMd5Dir)).To(BeADirectory())
		})

		It("should write a config.json file in the correct location", func() {
			Expect(filepath.Join(buildpackDir, "config.json")).To(BeAnExistingFile())
		})

		It("marshals the provided buildpacks to the config.json", func() {
			var actualBytes []byte
			actualBytes, err = ioutil.ReadFile(filepath.Join(buildpackDir, "config.json"))
			Expect(err).ToNot(HaveOccurred())

			var actualStringifiedBuildpacks []eirinistaging.StringifiedBuildpack
			err = json.Unmarshal(actualBytes, &actualStringifiedBuildpacks)
			Expect(err).ToNot(HaveOccurred())

			var actualBuildpacks []eirinistaging.Buildpack
			for _, b := range actualStringifiedBuildpacks {
				actualBuildpacks = append(actualBuildpacks, b.Buildpack)
			}

			Expect(buildpacks).To(ConsistOf(actualBuildpacks))
		})
	})

	Context("When a single buildpack with skip detect is provided", func() {
		BeforeEach(func() {
			detect := true
			buildpacks = []eirinistaging.Buildpack{
				{
					Name:       "my_buildpack",
					Key:        "my-key",
					URL:        fmt.Sprintf("%s/my-buildpack", server.URL()),
					SkipDetect: &detect,
				},
			}
		})

		It("should not fail", func() {
			Expect(err).ToNot(HaveOccurred())
		})

		It("should download all buildpacks to the given directory", func() {
			myMd5Dir := fmt.Sprintf("%x", md5.Sum([]byte("my_buildpack")))
			Expect(filepath.Join(buildpackDir, myMd5Dir)).To(BeADirectory())
		})

		It("should write a config.json file in the correct location", func() {
			Expect(filepath.Join(buildpackDir, "config.json")).To(BeAnExistingFile())
		})

		It("marshals the provided buildpacks to the config.json", func() {
			var actualBytes []byte
			actualBytes, err = ioutil.ReadFile(filepath.Join(buildpackDir, "config.json"))
			Expect(err).ToNot(HaveOccurred())

			var actualStringifiedBuildpacks []eirinistaging.StringifiedBuildpack
			err = json.Unmarshal(actualBytes, &actualStringifiedBuildpacks)
			Expect(err).ToNot(HaveOccurred())

			var actualBuildpacks []eirinistaging.Buildpack
			for _, b := range actualStringifiedBuildpacks {
				detect := true
				b.Buildpack.SkipDetect = &detect
				actualBuildpacks = append(actualBuildpacks, b.Buildpack)
			}

			Expect(buildpacks).To(ConsistOf(actualBuildpacks))
		})
	})

	Context("When the buildpack url is invalid", func() {
		BeforeEach(func() {
			server = ghttp.NewServer()
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/bad-buildpack"),
					ghttp.RespondWith(http.StatusInternalServerError, responseContent),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/bad-buildpack"),
					ghttp.RespondWith(http.StatusInternalServerError, responseContent),
				),
			)

			buildpacks = []eirinistaging.Buildpack{
				{
					Name: "bad_buildpack",
					Key:  "bad-key",
					URL:  fmt.Sprintf("%s/bad-buildpack", server.URL()),
				},
			}
		})

		It("should try both http clients", func() {
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("default client also failed")))
		})
	})
})