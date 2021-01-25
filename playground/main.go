package main

import (
	"fmt"
	"io/ioutil"
	"os"
)

func main() {
	symlinkPath := "symlink"
	targetPath := "test"

	err := ioutil.WriteFile(targetPath, []byte("Hello\n"), 0644)
	if err != nil {
		panic(err)
	}
	defer os.Remove(targetPath)

	err = os.Symlink(targetPath, symlinkPath)
	if err != nil {
		panic(err)
	}
	defer os.Remove(symlinkPath)

	s, err := os.Stat(symlinkPath)
	if err != nil {
		panic(err)
	}

	fmt.Println("symlink:", s.Mode()&os.ModeSymlink == os.ModeSymlink)
}
