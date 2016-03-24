package varnish

// varnish.go

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/plugins/inputs"
)

type Varnish struct {
	Stats []string `toml:"stats"`
}

func (s *Varnish) Description() string {
	return "a plugin to collect stats from varnish"
}

var varnishSampleConfig = `
  ## By default, telegraf gather stats for 3 metric points.
  ## Setting stats will remove the defaults
  stats = ['MAIN.cache_hit', 'MAIN.cache_miss', 'MAIN.uptime']
`

func (s *Varnish) SampleConfig() string {
	return varnishSampleConfig
}

func stringInSlice(str string, list []string) bool {
	for _, v := range list {
		if v == str {
			return true
		}
	}
	return false
}

func (s *Varnish) Gather(acc telegraf.Accumulator) error {

	var stats []string

	if len(s.Stats) == 0 {
		stats = []string{"MAIN.cache_hit", "MAIN.cache_miss", "MAIN.uptime"}
	} else {
		stats = s.Stats
	}

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
			stat_line := strings.Fields(scanner.Text())
			if stringInSlice(stat_line[0], stats) {
				tmp := strings.Split(stat_line[0], ".")
				sect := tmp[0]
				subsect := tmp[1]
				val := stat_line[1]
			}
		}
		acc.AddFields("varnish", fields, tags)
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
