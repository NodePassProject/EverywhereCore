package evcore

import (
	"errors"
	"os"

	mihomoConst "github.com/metacubex/mihomo/constant"
)

func SetResourcesPath(path string) error {
	if path == "" {
		return errors.New("empty resources path")
	}
	if err := os.MkdirAll(path, 0o755); err != nil {
		return err
	}
	_ = os.Setenv("xray.location.asset", path)
	_ = os.Setenv("xray.location.cert", path)
	mihomoConst.SetHomeDir(path)
	if err := os.Chdir(path); err != nil {
		return err
	}
	return nil
}
