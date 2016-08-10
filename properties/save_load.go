package properties

import (
	"encoding/json"
	"os"
	"path/filepath"

	"code.cloudfoundry.org/garden"
)

func Load(path string) (*Manager, error) {
	mgr := NewManager()

	filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		propertiesFilePath := filepath.Join(path, "props.json")
		if _, err := os.Stat(propertiesFilePath); err != nil {
			if os.IsNotExist(err) {
				return nil
			} else {
				return err
			}
		}

		file, err := os.Open(propertiesFilePath)
		if err != nil {
			return err
		}

		properties := garden.Properties{}
		decoder := json.NewDecoder(file)
		err = decoder.Decode(&properties)
		if err != nil {
			return err
		}

		for key, value := range properties {
			mgr.Set(filepath.Base(path), key, value)
		}

		return nil
	})
	// f, err := os.Open(path)
	// if err != nil {
	// 	return NewManager(), nil
	// }

	// var mgr Manager
	// if err := json.NewDecoder(f).Decode(&mgr); err != nil {
	// 	return nil, err
	// }

	// return &mgr, nil
	return mgr, nil
}

func Save(path string, props garden.Properties) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	return json.NewEncoder(f).Encode(props)

}
