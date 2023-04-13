package probe

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strconv"
)

type diskMetricsImpl struct {
	path string
}

func NewDiskMetrics(path string) DiskMetricsInterface {
	return &diskMetricsImpl{
		path: path,
	}
}

func execWrap(stdin []byte, command string, args ...string) ([]byte, error) {
	c := exec.Command(command, args...)
	if stdin != nil {
		c.Stdin = bytes.NewReader(stdin)
	}
	var stdoutBuf, stderrBuf bytes.Buffer
	c.Stdout = &stdoutBuf
	c.Stderr = &stderrBuf
	if err := c.Run(); err != nil {
		return nil, fmt.Errorf("exec failed. stderr=%s, %w", stderrBuf.Bytes(), err)
	}
	return stdoutBuf.Bytes(), nil
}

func parseFioResult(fioOutput []byte, property string) (float64, error) {
	jqOut, err := execWrap(
		fioOutput,
		"jq",
		fmt.Sprintf(".jobs[0].%s.lat_ns.mean", property),
	)
	if err != nil {
		return 0.0, err
	}

	stringJqOut := string(jqOut)

	actualNumber, err := strconv.ParseFloat(stringJqOut[0:len(stringJqOut)-1], 64)
	if err != nil {
		return 0.0, err
	}

	// actualNumber is nano-seconds, so convert it to seconds order
	return actualNumber / 1_000_000_000, nil
}

func (mtr *diskMetricsImpl) GetMetrics(ctx context.Context) (*DiskMetrics, error) {
	fioStdout, err := execWrap(
		nil,
		"fio",
		fmt.Sprintf("-filename=%s/.iotest", mtr.path),
		"-direct=1",
		"-rw=readwrite",
		"-bs=4k",
		"-size=50M",
		"-numjobs=1",
		"-runtime=1",
		"-group_reporting",
		"-name=run1",
		"--output-format=json",
	)
	if err != nil {
		return nil, err
	}

	var metrics DiskMetrics
	metrics.ReadLatency, err = parseFioResult(fioStdout, "read")
	if err != nil {
		return nil, err
	}

	metrics.WriteLatency, err = parseFioResult(fioStdout, "write")
	if err != nil {
		return nil, err
	}

	return &metrics, nil
}
