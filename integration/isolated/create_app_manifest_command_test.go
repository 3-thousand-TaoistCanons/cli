package isolated

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"code.cloudfoundry.org/cli/integration/helpers"
	"code.cloudfoundry.org/cli/util/manifest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"
	yaml "gopkg.in/yaml.v2"
)

func createManifest(appName string) (manifest.Manifest, error) {
	tmpDir, err := ioutil.TempDir("", "")
	defer os.RemoveAll(tmpDir)
	if err != nil {
		return manifest.Manifest{}, err
	}

	manifestPath := filepath.Join(tmpDir, "manifest.yml")
	Eventually(helpers.CustomCF(helpers.CFEnv{WorkingDirectory: tmpDir}, "create-app-manifest", appName, "-p", manifestPath)).Should(Exit(0))

	manifestContents, err := ioutil.ReadFile(manifestPath)
	if err != nil {
		return manifest.Manifest{}, err
	}

	var appsManifest manifest.Manifest
	err = yaml.Unmarshal(manifestContents, &appsManifest)
	if err != nil {
		return manifest.Manifest{}, err
	}

	return appsManifest, nil
}

var _ = Describe("create-app-manifest command", func() {
	var appName string
	var manifestFilePath string
	var tempDir string

	BeforeEach(func() {
		appName = helpers.NewAppName()
		var err error
		tempDir, err = ioutil.TempDir("", "create-manifest")
		Expect(err).ToNot(HaveOccurred())

		manifestFilePath = filepath.Join(tempDir, fmt.Sprintf("%s_manifest.yml", appName))
	})

	AfterEach(func() {
		os.RemoveAll(tempDir)
	})

	Context("Help", func() {
		It("displays the help information", func() {
			session := helpers.CustomCF(helpers.CFEnv{WorkingDirectory: tempDir}, "create-app-manifest", "--help")
			Eventually(session.Out).Should(Say("NAME:"))
			Eventually(session.Out).Should(Say("create-app-manifest - Create an app manifest for an app that has been pushed successfully"))
			Eventually(session.Out).Should(Say("USAGE:"))
			Eventually(session.Out).Should(Say("cf create-app-manifest APP_NAME \\[-p \\/path\\/to\\/<app-name>_manifest\\.yml\\]"))
			Eventually(session.Out).Should(Say(""))
			Eventually(session.Out).Should(Say("OPTIONS:"))
			Eventually(session.Out).Should(Say("-p      Specify a path for file creation. If path not specified, manifest file is created in current working directory."))
			Eventually(session.Out).Should(Say("SEE ALSO:"))
			Eventually(session.Out).Should(Say("apps, push"))
		})
	})

	Context("when the environment is not setup correctly", func() {
		Context("when no API endpoint is set", func() {
			BeforeEach(func() {
				helpers.UnsetAPI()
			})

			It("fails with no API endpoint set message", func() {
				session := helpers.CustomCF(helpers.CFEnv{WorkingDirectory: tempDir}, "create-app-manifest", appName)
				Eventually(session.Out).Should(Say("FAILED"))
				Eventually(session.Err).Should(Say("No API endpoint set. Use 'cf login' or 'cf api' to target an endpoint."))
				Eventually(session).Should(Exit(1))
			})
		})

		Context("when not logged in", func() {
			BeforeEach(func() {
				helpers.LogoutCF()
			})

			It("fails with not logged in message", func() {
				session := helpers.CustomCF(helpers.CFEnv{WorkingDirectory: tempDir}, "create-app-manifest", appName)
				Eventually(session.Out).Should(Say("FAILED"))
				Eventually(session.Err).Should(Say("Not logged in. Use 'cf login' to log in."))
				Eventually(session).Should(Exit(1))
			})
		})

		Context("when there is no org set", func() {
			BeforeEach(func() {
				helpers.LogoutCF()
				helpers.LoginCF()
			})

			It("fails with no targeted org error message", func() {
				session := helpers.CustomCF(helpers.CFEnv{WorkingDirectory: tempDir}, "create-app-manifest", appName)
				Eventually(session.Out).Should(Say("FAILED"))
				Eventually(session.Err).Should(Say("No org targeted, use 'cf target -o ORG' to target an org."))
				Eventually(session).Should(Exit(1))
			})
		})

		Context("when there is no space set", func() {
			BeforeEach(func() {
				helpers.LogoutCF()
				helpers.LoginCF()
				helpers.TargetOrg(ReadOnlyOrg)
			})

			It("fails with no targeted space error message", func() {
				session := helpers.CustomCF(helpers.CFEnv{WorkingDirectory: tempDir}, "create-app-manifest", appName)
				Eventually(session.Out).Should(Say("FAILED"))
				Eventually(session.Err).Should(Say("No space targeted, use 'cf target -s SPACE' to target a space."))
				Eventually(session).Should(Exit(1))
			})
		})
	})

	Context("when app name not provided", func() {
		It("displays a usage error", func() {
			session := helpers.CustomCF(helpers.CFEnv{WorkingDirectory: tempDir}, "create-app-manifest")
			Eventually(session.Err).Should(Say("Incorrect Usage: the required argument `APP_NAME` was not provided"))
			Eventually(session.Out).Should(Say("USAGE:"))
		})
	})

	Context("when the environment is setup correctly", func() {
		var (
			orgName   string
			spaceName string
			userName  string

			domainName string
		)

		BeforeEach(func() {
			orgName = helpers.NewOrgName()
			spaceName = helpers.NewSpaceName()

			setupCF(orgName, spaceName)
			userName, _ = helpers.GetCredentials()
			domainName = defaultSharedDomain()
		})

		Context("when the app does not exist", func() {
			It("displays a usage error", func() {
				session := helpers.CustomCF(helpers.CFEnv{WorkingDirectory: tempDir}, "create-app-manifest", appName)
				Eventually(session.Out).Should(Say("Creating an app manifest from current settings of app %s in org %s / space %s as %s\\.\\.\\.", appName, orgName, spaceName, userName))
				Eventually(session.Out).Should(Say("FAILED"))
				Eventually(session.Err).Should(Say("App %s not found", appName))
			})
		})

		Context("when the app exists", func() {
			BeforeEach(func() {
				helpers.WithHelloWorldApp(func(appDir string) {
					Eventually(helpers.CustomCF(helpers.CFEnv{WorkingDirectory: appDir}, "v2-push", appName)).Should(Exit(0))
				})
			})

			It("creates the manifest", func() {
				session := helpers.CustomCF(helpers.CFEnv{WorkingDirectory: tempDir}, "create-app-manifest", appName)
				Eventually(session.Out).Should(Say("Creating an app manifest from current settings of app %s in org %s / space %s as %s\\.\\.\\.", appName, orgName, spaceName, userName))
				Eventually(session.Out).Should(Say("OK"))
				expectedFilePath := helpers.ConvertPathToRegularExpression(fmt.Sprintf(".%s%s_manifest.yml", string(os.PathSeparator), appName))
				Eventually(session.Out).Should(Say("Manifest file created successfully at %s", expectedFilePath))

				expectedFile := fmt.Sprintf(`applications:
- name: %s
  disk_quota: 1G
  instances: 1
  memory: 32M
  routes:
  - route: %s.%s
  stack: cflinuxfs2
`, appName, strings.ToLower(appName), domainName)

				createdFile, err := ioutil.ReadFile(manifestFilePath)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(createdFile)).To(Equal(expectedFile))
			})

			Context("when the -p flag is provided", func() {
				Context("when the specified file is a directory", func() {
					It("displays a file creation error", func() {
						session := helpers.CustomCF(helpers.CFEnv{WorkingDirectory: tempDir}, "create-app-manifest", appName, "-p", tempDir)
						Eventually(session.Out).Should(Say("Creating an app manifest from current settings of app %s in org %s / space %s as %s\\.\\.\\.", appName, orgName, spaceName, userName))
						Eventually(session.Out).Should(Say("FAILED"))
						Eventually(session.Err).Should(Say("Error creating manifest file: open %s: is a directory", helpers.ConvertPathToRegularExpression(tempDir)))
					})
				})

				Context("when the specified file does not exist", func() {
					var newFile string

					BeforeEach(func() {
						newFile = filepath.Join(tempDir, "new-file.yml")
					})

					It("creates the manifest in the file", func() {
						session := helpers.CustomCF(helpers.CFEnv{WorkingDirectory: tempDir}, "create-app-manifest", appName, "-p", newFile)
						Eventually(session.Out).Should(Say("Creating an app manifest from current settings of app %s in org %s / space %s as %s\\.\\.\\.", appName, orgName, spaceName, userName))
						Eventually(session.Out).Should(Say("OK"))
						Eventually(session.Out).Should(Say("Manifest file created successfully at %s", helpers.ConvertPathToRegularExpression(newFile)))

						expectedFile := fmt.Sprintf(`applications:
- name: %s
  disk_quota: 1G
  instances: 1
  memory: 32M
  routes:
  - route: %s.%s
  stack: cflinuxfs2
`, appName, strings.ToLower(appName), domainName)

						createdFile, err := ioutil.ReadFile(newFile)
						Expect(err).ToNot(HaveOccurred())
						Expect(string(createdFile)).To(Equal(expectedFile))
					})
				})

				Context("when the specified file exists", func() {
					var existingFile string

					BeforeEach(func() {
						existingFile = filepath.Join(tempDir, "some-file")
						f, err := os.Create(existingFile)
						Expect(err).ToNot(HaveOccurred())
						Expect(f.Close()).To(Succeed())
					})

					It("overrides the previous file with the new manifest", func() {
						session := helpers.CustomCF(helpers.CFEnv{WorkingDirectory: tempDir}, "create-app-manifest", appName, "-p", existingFile)
						Eventually(session.Out).Should(Say("Creating an app manifest from current settings of app %s in org %s / space %s as %s\\.\\.\\.", appName, orgName, spaceName, userName))
						Eventually(session.Out).Should(Say("OK"))
						Eventually(session.Out).Should(Say("Manifest file created successfully at %s", helpers.ConvertPathToRegularExpression(existingFile)))

						expectedFile := fmt.Sprintf(`applications:
- name: %s
  disk_quota: 1G
  instances: 1
  memory: 32M
  routes:
  - route: %s.%s
  stack: cflinuxfs2
`, appName, strings.ToLower(appName), domainName)

						createdFile, err := ioutil.ReadFile(existingFile)
						Expect(err).ToNot(HaveOccurred())
						Expect(string(createdFile)).To(Equal(expectedFile))
					})
				})
			})
		})

		Context("when app was created with docker image", func() {
			var (
				dockerImage       string
				oldDockerPassword string
			)

			BeforeEach(func() {
				oldDockerPassword = os.Getenv("CF_DOCKER_PASSWORD")
				Expect(os.Setenv("CF_DOCKER_PASSWORD", "my-docker-password")).To(Succeed())

				dockerImage = "cloudfoundry/diego-docker-app-custom"
				Eventually(helpers.CF("v2-push", appName, "-o", dockerImage, "--docker-username", "some-docker-username")).Should(Exit())
			})

			AfterEach(func() {
				Expect(os.Setenv("CF_DOCKER_PASSWORD", oldDockerPassword)).To(Succeed())
			})

			It("creates the manifest", func() {
				session := helpers.CustomCF(helpers.CFEnv{WorkingDirectory: tempDir}, "create-app-manifest", appName, "-v")
				Eventually(session.Out).Should(Say("Creating an app manifest from current settings of app %s in org %s / space %s as %s\\.\\.\\.", appName, orgName, spaceName, userName))
				Eventually(session.Out).Should(Say("OK"))
				expectedFilePath := helpers.ConvertPathToRegularExpression(fmt.Sprintf(".%s%s_manifest.yml", string(os.PathSeparator), appName))
				Eventually(session.Out).Should(Say("Manifest file created successfully at %s", expectedFilePath))

				expectedFile := fmt.Sprintf(`applications:
- name: %s
  disk_quota: 1G
  docker:
    image: %s
    username: some-docker-username
  instances: 1
  memory: 32M
  routes:
  - route: %s.%s
  stack: cflinuxfs2
`, appName, dockerImage, strings.ToLower(appName), domainName)

				createdFile, err := ioutil.ReadFile(manifestFilePath)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(createdFile)).To(Equal(expectedFile))
			})
		})

		Context("when app has no hostname", func() {
			var domain helpers.Domain

			BeforeEach(func() {
				domain = helpers.NewDomain(orgName, helpers.DomainName(""))
				domain.Create()

				helpers.WithHelloWorldApp(func(appDir string) {
					Eventually(helpers.CustomCF(helpers.CFEnv{WorkingDirectory: appDir}, "push", appName, "--no-start", "-b", "staticfile_buildpack", "--no-hostname", "-d", domain.Name)).Should(Exit(0))
				})
			})

			It("contains routes without hostnames", func() {
				appManifest, err := createManifest(appName)
				Expect(err).ToNot(HaveOccurred())

				Expect(appManifest.Applications).To(HaveLen(1))
				Expect(appManifest.Applications[0].Routes).To(HaveLen(1))
				Expect(appManifest.Applications[0].Routes[0]).To(Equal(domain.Name))
			})
		})
	})
})
