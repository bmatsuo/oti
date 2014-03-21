package packer

import (
	"encoding/csv"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type ErrorOutput []Output

func (err ErrorOutput) Error() string { return fmt.Sprint([]Output(err)) }

func Command(name string, args ...string) *exec.Cmd {
	_args := []string{"-machine-readable", name}
	_args = append(_args, args...)
	return exec.Command("packer", args...)
}

func Run(cmd *exec.Cmd) ([]Output, error) {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	d := NoUI(NewDecoder(stdout))

	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	var outs []Output
	for {
		o, err := d.Decode()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		outs = append(outs, o)
	}

	err = cmd.Wait()
	if err != nil {
		return outs, err
	}

	return outs, nil
}

func Validate(path string) error {
	outs, err := Run(Command("validate", path))
	if err != nil {
		return ErrorOutput(outs)
	}
	return nil
}

func Fix(path string) ([]Output, error) {
	outs, err := Run(Command("fix", path))
	if err != nil {
		return nil, ErrorOutput(outs)
	}
	return outs, nil
}

func Inspect(path string) ([]Output, error) {
	outs, err := Run(Command("inspect", path))
	if err != nil {
		return nil, ErrorOutput(outs)
	}
	return outs, nil
}

type Opts struct {
	Vars     []string
	Varfiles []string
	Only     []string
	Except   []string
}

func Build(packerfile string, opts Opts) ([]Output, error) {
	var args []string
	for i := range opts.Vars {
		args = append(args, "-var="+opts.Vars[i])
	}
	for i := range opts.Varfiles {
		args = append(args, "-var-file="+opts.Varfiles[i])
	}
	if len(opts.Only) > 0 {
		args = append(args, "-only="+strings.Join(opts.Only, ","))
	}
	if len(opts.Except) > 0 {
		args = append(args, "-except="+strings.Join(opts.Except, ","))
	}
	args = append(args, packerfile)

	outs, err := Run(Command("build", args...))
	if err != nil {
		return nil, ErrorOutput(outs)
	}

	return outs, nil
}

func NoUI(d Decoder) Decoder {
	return decoderFunc(func() (Output, error) {
		for {
			o, err := d.Decode()
			if err != nil {
				return o, err
			}
			if o.Type == "ui" {
				continue
			}
			return o, nil
		}
	})
}

type decoderFunc func() (Output, error)

func (fn decoderFunc) Decode() (Output, error) { return fn() }

type Decoder interface {
	Decode() (Output, error)
}

func NewDecoder(r io.Reader) Decoder {
	return &decoder{csv.NewReader(r)}
}

type decoder struct {
	r *csv.Reader
}

func (d *decoder) Decode() (Output, error) {
	var o Output
	row, err := d.r.Read()
	if err != nil {
		return o, err
	}
	switch len(row) {
	default:
		fallthrough
	case 4:
		o.Data = row[3]
		o.Data = strings.Replace(o.Data, `\r`, "\r", -1)
		o.Data = strings.Replace(o.Data, `\n`, "\n", -1)
		o.Data = strings.Replace(o.Data, `%!(PACKER_COMMA)`, ",", -1)
		fallthrough
	case 3:
		o.Type = row[2]
		fallthrough
	case 2:
		o.Target = row[1]
		fallthrough
	case 1:
		unix, err := strconv.ParseInt(row[0], 10, 64)
		if err != nil {
			return o, err
		}
		o.Time = time.Unix(unix, 0)
	case 0:
		panic("no columns")
	}
	return o, nil
}

type Output struct {
	Time   time.Time
	Target string
	Type   string
	Data   string
}
