package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

func filter(files []os.FileInfo, f func(os.FileInfo) bool) []os.FileInfo {
	filtered := make([]os.FileInfo, 0)
	for _, file := range files {
		if f(file) {
			filtered = append(filtered, file)
		}
	}
	return filtered
}

func dirTreeRec(out io.Writer, path string, printFiles bool, prefix string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("Path is invalid")
	}
	files, err := file.Readdir(0)
	if err != nil {
		return fmt.Errorf("Couldn't read directory")
	}

	if !printFiles {
		files = filter(files, func(f os.FileInfo) bool {
			return f.IsDir()
		})
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].Name() < files[j].Name()
	})

	for ind, fileInfo := range files {
		var fileName, dirPrefix string
		if ind == len(files)-1 {
			fileName = prefix + "└───" + fileInfo.Name()
			dirPrefix = prefix + "	"
		} else {
			fileName = prefix + "├───" + fileInfo.Name()
			dirPrefix = prefix + "│	"
		}
		if !fileInfo.IsDir() && fileInfo.Size() == 0 {
			fileName += " (empty)"
		}
		if !fileInfo.IsDir() && fileInfo.Size() != 0 {
			fileName += fmt.Sprintf(" (%db)", fileInfo.Size())
		}

		fmt.Fprintln(out, fileName)

		if fileInfo.IsDir() {
			err := dirTreeRec(out, filepath.Join(path, fileInfo.Name()), printFiles, dirPrefix)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func dirTree(out io.Writer, path string, printFiles bool) error {
	return dirTreeRec(out, path, printFiles, "")
}

func main() {
	out := os.Stdout
	if !(len(os.Args) == 2 || len(os.Args) == 3) {
		panic("usage go run main.go . [-f]")
	}
	path := os.Args[1]
	printFiles := len(os.Args) == 3 && os.Args[2] == "-f"
	err := dirTree(out, path, printFiles)
	if err != nil {
		panic(err.Error())
	}
}
