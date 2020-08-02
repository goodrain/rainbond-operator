package framework

import (
	"bytes"
	"os"
	"os/exec"
)

func InstallReainbondOperator(operatorName, chartsPath, namespace string) error {
	cmd := exec.Command("helm", "install", operatorName, chartsPath, "--namespace", namespace)
	stderrBuf := &bytes.Buffer{}
	cmd.Stdout = os.Stdout
	cmd.Stderr = stderrBuf
	if err := cmd.Start(); err != nil {
		return err
	}
	if err := cmd.Wait(); err != nil {
		return err
	}
	log("INFO", stderrBuf.String())
	return nil
}
