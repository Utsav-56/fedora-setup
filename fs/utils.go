package fs

import (
	"fmt"
	"os"
)

func Exists(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		if st := os.IsNotExist(err); st {
			return st
		}
		fmt.Printf("Error Determining file exists or not: %s", err)
	}

	return true
}

func IsFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		if st := os.IsNotExist(err); st {
			return st
		}
		fmt.Printf("Error Determining file exists or not: %s", err)
	}
	return info.Mode().IsRegular()
}

func IsDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		if st := os.IsNotExist(err); st {
			return st
		}
		fmt.Printf("Error Determining file exists or not: %s", err)
	}
	return info.Mode().IsDir()
}

func IsSymLink(path string) bool {
	info, err := os.Lstat(path)
	if err != nil {
		if st := os.IsNotExist(err); st {
			return st
		}
		fmt.Printf("Error Determining file exists or not: %s", err)
	}
	return info.Mode()&os.ModeSymlink != 0
}

func FileExists(path string) bool {
	return Exists(path) && IsFile(path)
}

func DirExists(path string) bool {
	return Exists(path) && IsDir(path)
}

func symLinkExists(path string) bool {
	return Exists(path) && IsSymLink(path)
}

func ReadFileString(path string) (string, error) {
	if !FileExists(path) {
		return "", fmt.Errorf("file not found: %s", path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("error reading file: %s", err)
	}
	return string(data), nil
}

func WriteFileString(path string, data string) error {
	if !FileExists(path) {
		return fmt.Errorf("file not found: %s", path)
	}

	err := os.WriteFile(path, []byte(data), 0644)
	if err != nil {
		return fmt.Errorf("error writing file: %s", err)
	}
	return nil
}
