package giterminism_test

import (
	"bytes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/werf/werf/integration/pkg/utils"
	"runtime"
)

var _ = Describe("config", func() {
	BeforeEach(func() {
		gitInit()
		utils.CopyIn(utils.FixturePath("config"), SuiteData.TestDirPath)
		gitAddAndCommit("werf-giterminism.yaml")
	})

	{
		type entry struct {
			allowUncommitted        bool
			addConfig               bool
			commitConfig            bool
			changeConfigAfterCommit bool
			expectedErrSubstring    string
		}

		DescribeTable("config.allowUncommitted",
			func(e entry) {
				var contentToAppend string
				if e.allowUncommitted {
					contentToAppend = `
config:
  allowUncommitted: true`
				} else {
					contentToAppend = `
config:
  allowUncommitted: false`
				}
				fileCreateOrAppend("werf-giterminism.yaml", contentToAppend)
				gitAddAndCommit("werf-giterminism.yaml")

				if e.addConfig {
					fileCreateOrAppend("werf.yaml", `
configVersion: 1
project: none
---
`)
				}

				if e.commitConfig {
					gitAddAndCommit("werf.yaml")
				}

				if e.changeConfigAfterCommit {
					fileCreateOrAppend("werf.yaml", "\n")
				}

				output, err := utils.RunCommand(
					SuiteData.TestDirPath,
					SuiteData.WerfBinPath,
					"config", "render",
				)

				if e.expectedErrSubstring != "" {
					Ω(err).Should(HaveOccurred())
					Ω(string(output)).Should(ContainSubstring(e.expectedErrSubstring))
				} else {
					Ω(err).ShouldNot(HaveOccurred())
				}
			},
			Entry("werf.yaml not found", entry{
				expectedErrSubstring: `unable to read werf config: the file 'werf.yaml' not found in the project git repository`,
			}),
			Entry("werf.yaml not committed", entry{
				addConfig:            true,
				expectedErrSubstring: `unable to read werf config: the file 'werf.yaml' must be committed`,
			}),
			Entry("werf.yaml committed", entry{
				addConfig:    true,
				commitConfig: true,
			}),
			Entry("werf.yaml committed, werf.yaml has uncommitted changes", entry{
				addConfig:               true,
				commitConfig:            true,
				changeConfigAfterCommit: true,
				expectedErrSubstring:    `unable to read werf config: the file 'werf.yaml' changes must be committed`,
			}),
			Entry("config.allowUncommitted is true, werf.yaml not committed", entry{
				allowUncommitted: true,
				addConfig:        true,
			}),
			Entry("config.allowUncommitted is true, werf.yaml committed", entry{
				allowUncommitted: true,
				addConfig:        true,
				commitConfig:     true,
			}),
		)
	}

	{
		type entry struct {
			allowUncommitted         bool
			addSymlinks              bool
			commitSymlinks           bool
			changeSymlinkAfterCommit bool
			expectedErrSubstring     string
		}

		createSymlinkFile := func(relPath string, link string) {
			hashBytes, _ := utils.RunCommandWithOptions(
				SuiteData.TestDirPath,
				"git",
				[]string{"hash-object", "-w", "--stdin"},
				utils.RunCommandOptions{
					ToStdin:       link,
					ShouldSucceed: true,
				},
			)

			utils.RunSucceedCommand(
				SuiteData.TestDirPath,
				"git",
				"update-index", "--add", "--cacheinfo", "120000", string(bytes.TrimSpace(hashBytes)), relPath,
			)

			utils.RunSucceedCommand(
				SuiteData.TestDirPath,
				"git",
				"checkout", relPath,
			)
		}

		DescribeTable("config.allowUncommitted xxxxxxxxxxxxxx",
			func(e entry) {
				if e.allowUncommitted && runtime.GOOS == "windows" {
					Skip("skip on windows")
				}

				var contentToAppend string
				if e.allowUncommitted {
					contentToAppend = `
config:
  allowUncommitted: true`
				} else {
					contentToAppend = `
config:
  allowUncommitted: false`
				}
				fileCreateOrAppend("werf-giterminism.yaml", contentToAppend)
				gitAddAndCommit("werf-giterminism.yaml")

				if e.addSymlinks {
					fileCreateOrAppend("test", `
configVersion: 1
project: none
---
`)
					createSymlinkFile("werf.yaml", "test")
				}

				if e.commitSymlinks {
					gitAddAndCommit("test")
				}

				output, err := utils.RunCommand(
					SuiteData.TestDirPath,
					SuiteData.WerfBinPath,
					"config", "render",
				)

				if e.expectedErrSubstring != "" {
					Ω(err).Should(HaveOccurred())
					Ω(string(output)).Should(ContainSubstring(e.expectedErrSubstring))
				} else {
					Ω(err).ShouldNot(HaveOccurred())
				}
			},
			Entry("werf.yaml not committed", entry{
				addSymlinks:          true,
				expectedErrSubstring: `unable to read werf config: the file 'werf.yaml' must be committed`,
			}),
			Entry("werf.yaml allow uncommitted", entry{
				allowUncommitted: true,
				addSymlinks:      true,
			}),
			Entry("werf.yaml committed", entry{
				addSymlinks:    true,
				commitSymlinks: true,
			}),
		)
	}
})
