package utils

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

type Executable string

func (this Executable) Command(args ...string) (*exec.Cmd, error) {
	var buf Executable
	if err := buf.Set(string(this)); err != nil {
		return nil, err
	}

	return exec.Command(string(buf), args...), nil
}

func (this *Executable) Set(plain string) error {
	result, err := exec.LookPath(plain)
	if err == nil || !os.IsNotExist(err) {
		*this = Executable(result)
		return err
	}

	for _, ext := range this.additionalExtensions() {
		result, err := exec.LookPath(plain + ext)
		if err == nil {
			*this = Executable(result)
			return nil
		}
		if err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	return fmt.Errorf("%w: %s", os.ErrNotExist, plain)
}

func (this Executable) String() string {
	return string(this)
}

func (this Executable) additionalExtensions() []string {
	switch runtime.GOOS {
	case "windows":
		parts := os.Getenv("PATHEXT")
		if strings.TrimSpace(parts) == "" {
			parts = ".COM;.EXE;.BAT;.CMD"
		}
		return strings.Split(parts, ";")
	default:
		return nil
	}
}

func (this Executable) Errorf(args []string, msg string, msgArgs ...interface{}) error {
	targetMsg := fmt.Sprintf("%s: %s", this.FormatArgs(args), msg)
	return fmt.Errorf(targetMsg, msgArgs...)
}

func (this Executable) FormatArgs(args []string) string {
	args = append([]string{string(this)}, args...)
	bufs := make([]string, len(args))
	for i, arg := range args {
		bufs[i] = fmt.Sprintf("%q", arg)
	}
	return "[" + strings.Join(bufs, ", ") + "]"
}
