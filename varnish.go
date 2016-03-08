package varnish

// varnish.go

import (
    "bufio"
    "fmt"
    "os"
    "os/exec"

    "github.com/influxdata/telegraf"
    "github.com/influxdata/telegraf/plugins/inputs"
)

type Varnish struct {
    Ok bool
}

func (s *Varnish) Description() string {
    return "a plugin to collect stats from varnish"
}

func (s *Varnish) SampleConfig() string {
    return "ok = true # indicate if everything is fine"
}

func (s *Varnish) Gather(acc telegraf.Accumulator) error {

    cmdName := "/usr/bin/varnishstat"
    cmdArgs := []string{"-1"}

    cmd := exec.Command(cmdName, cmdArgs...)
    cmdReader, err := cmd.StdoutPipe()
    if err != nil {
        fmt.Fprintln(os.Stderr, "Error creating StdoutPipe for Cmd", err)
        os.Exit(1)
    }

    scanner := bufio.NewScanner(cmdReader)
    go func() {
        for scanner.Scan() {
            fmt.Printf("stat is | %s\n", scanner.Text())
        }
    }()

    err = cmd.Start()
    if err != nil {
        fmt.Fprintln(os.Stderr, "Error starting Cmd", err)
        os.Exit(1)
    }

    err = cmd.Wait()
    if err != nil {
        fmt.Fprintln(os.Stderr, "Error waiting for Cmd", err)
        os.Exit(1)
    }

    return nil
}

func init() {
    inputs.Add("varnish", func() telegraf.Input { return &Varnish{} })
}
