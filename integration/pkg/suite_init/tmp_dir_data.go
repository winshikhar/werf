package suite_init

import (
	"fmt"
	"github.com/onsi/ginkgo"
	"github.com/werf/werf/integration/pkg/utils"
)

type TmpDirData struct {
	TmpDir      string
	TestDirPath string
}

func NewTmpDirData() *TmpDirData {
	data := &TmpDirData{}
	SetupTmpDir(&data.TmpDir, &data.TestDirPath)
	return data
}

func SetupTmpDir(tmpDir, testDirPath *string) bool {
	ginkgo.BeforeEach(func() {
		*tmpDir = utils.GetTempDir()
		*testDirPath = *tmpDir
		fmt.Println(*tmpDir)
	})

	//ginkgo.AfterEach(func() {
	//	err := os.RemoveAll(*tmpDir)
	//	Ω(err).ShouldNot(HaveOccurred())
	//})

	return true
}
