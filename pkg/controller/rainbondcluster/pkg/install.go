package pkg

import (
	"fmt"
	"github.com/GLYASAI/rainbond-operator/pkg/controller/rainbondcluster/types"
	"os"
)

// PackageInterface is responsible for processing installation packages, including extraction, loading images and pushing images
type PackageInterface interface {
	HistoryInterface
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
	info, err := os.Stat(i.destPath)
	if err != nil {
		return nil, err
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", i.destPath)
	}

	return &types.ExtractionHistory{
		Status: types.HistoryStatusTrue,
	}, nil
}
