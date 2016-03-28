package varnish

// varnish.go

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strconv"
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

	sections := []string{"LCK", "MAIN", "MEMPOOL", "MGT", "SMA", "VBE"}
	stats := []string{"MAIN.cache_hit", "MAIN.cache_miss", "MAIN.uptime",
		"LCK.lru.locks", "LCK.smp.destroy",
		"MEMPOOL.sess1.randry", "MEMPOOL.sess1.surplus",
		"SMA.s0.c_freed"}
	statsFilter := make(map[string]bool)

	for _, s := range stats {
		statsFilter[s] = true
	}

	sectionMap := make(map[string]map[string]interface{})

	for _, s := range sections {
		sectionMap[s] = make(map[string]interface{})
	}

	cmdName := "/usr/bin/varnishstat"
	cmdArgs := []string{"-1"}

	cmd := exec.Command(cmdName, cmdArgs...)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("error running varnishstat: %s", err)
	}

	scanner := bufio.NewScanner(&out)
	for scanner.Scan() {
		cols := strings.Fields(scanner.Text())
		if len(cols) < 2 {
			continue
		}
		if !strings.Contains(cols[0], ".") {
			continue
		}

		metric := cols[0]
		value := cols[1]

		if !statsFilter[metric] {
			continue
		}

		parts := strings.SplitN(metric, ".", 2)
		section := parts[0]
		field := parts[1]

		// Only add the sections we care about
		if _, ok := sectionMap[section]; ok {
			sectionMap[section][field], err = strconv.Atoi(value)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Expected a numeric vlaue for %s = %v\n",
					metric, value)
			}
		}
	}

	for section, fields := range sectionMap {
		tags := map[string]string{
			"section": section,
		}
		if len(fields) == 0 {
			continue
		}

		acc.AddFields("varnish", fields, tags)
	}

	return nil
}

func init() {
	inputs.Add("varnish", func() telegraf.Input { return &Varnish{} })
}
