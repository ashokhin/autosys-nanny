package file

import (
	"errors"
	"fmt"
	"os"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"gopkg.in/yaml.v3"
)

func fileExists(filepath string) error {
	var err error

	fileinfo, err := os.Stat(filepath)

	if os.IsNotExist(err) {
		return err
	}

	// Return error if the fileinfo says the file path is a directory.
	if fileinfo.IsDir() {
		return newIsDirError(fmt.Sprintf("'%v' is a directory", filepath))
	}

	return nil
}

// loadYamlFile read YAML file to byte slice than
// yaml.Unmarshal decodes the first document found within the in byte slice and assigns decoded values into the out value.
func LoadYamlFile(filePath string, out interface{}, logger log.Logger) error {
	var err error
	var f []byte

	level.Debug(logger).Log("msg", "read YAML file", "value", filePath)

	if err := fileExists(filePath); err != nil {
		var isDirErr *ErrIsDir

		if os.IsNotExist(err) {
			level.Error(logger).Log("msg", "YAML file does not exists", "value", filePath, "error", err.Error())

			return err
		}

		if errors.As(err, &isDirErr) {
			level.Error(logger).Log("msg", "YAML path is not a file", "value", filePath, "error", err.Error())

			return err
		}
	}

	f, err = os.ReadFile(filePath)

	if err != nil {
		level.Error(logger).Log("msg", "read file error", "error", err.Error())

		return err
	}

	if err := yaml.Unmarshal(f, out); err != nil {
		level.Error(logger).Log("msg", "error unmarshal YAML configuration", "error", err.Error())

		return err
	}

	return nil
}
