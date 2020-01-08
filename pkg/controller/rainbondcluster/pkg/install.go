package pkg

import (
	"encoding/json"
	"fmt"
	"github.com/GLYASAI/rainbond-operator/pkg/controller/rainbondcluster/types"
	"github.com/GLYASAI/rainbond-operator/pkg/util/commonutil"
	"io/ioutil"
	"os"
	"path"
)

// PackageInterface is responsible for processing installation packages, including extraction, loading images and pushing images
type PackageInterface interface {
	HistoryInterface
	GetMetadata() ([]string, error)
}

// HistoryInterface is responsible for obtaining installation history.
type HistoryInterface interface {
	ExtractionHistory() (*types.ExtractionHistory, error)
}

type installpkg struct {
	// Destination path of the installation package extraction.
	destPath string
}

func New(destPath string) PackageInterface {
	return &installpkg{
		destPath: destPath,
	}
}

func (i *installpkg) ExtractionHistory() (*types.ExtractionHistory, error) {
	exists, err := i.destPathExists()
	if err != nil {
		return nil, err
	}
	if !exists {
		return &types.ExtractionHistory{
			Status: types.HistoryStatusFalse,
			Reason: "DestPathNotExists",
		}, nil
	}

	return &types.ExtractionHistory{
		Status: types.HistoryStatusTrue,
	}, nil
}

func (i *installpkg) GetMetadata() ([]string, error) {
	exists, err := i.destPathExists()
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("Destination path does not exist: %s", i.destPath)
	}

	filename := path.Join(i.destPath, "metadata.json")
	if !commonutil.FileExists(filename) {
		return nil, fmt.Errorf("File not exists: %s", filename)
	}

	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("Error reading file %s: %v", filename, err)
	}

	var res []string
	if err := json.Unmarshal(b, &res); err != nil {
		return nil, fmt.Errorf("Error Unmarshalling: %v", err)
	}

	return res, nil
}

func (i *installpkg) destPathExists() (bool, error) {
	info, err := os.Stat(i.destPath)
	if err != nil {
		return false, err
	}

	if !info.IsDir() {
		return false, fmt.Errorf("%s is not a directory", i.destPath)
	}

	return true, nil
}
