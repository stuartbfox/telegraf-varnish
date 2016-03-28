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

const (
	kwAll = "all"
)

// Varnish is used to store configuration values
type Varnish struct {
	Stats []string `toml:"stats"`
}

func (s *Varnish) Description() string {
	return "a plugin to collect stats from varnish"
}

var defaultStats = []string{"MAIN.cache_hit", "MAIN.cache_miss", "MAIN.uptime"}

var varnishSampleConfig = `
  ## By default, telegraf gather stats for 3 metric points.
  ## Setting stats will remove the defaults
  stats = %v

  ## Use the keyword 'all' to include everything
  stats = ['%s']
`

// SampleConfig displays configuration instructions
func (s *Varnish) SampleConfig() string {
	return fmt.Sprintf(varnishSampleConfig, defaultStats, kwAll)
}

// Builds a filter function that will indicate whether a given stat should
// be reported
func (s *Varnish) statsFilter() func(string) bool {
	stats := defaultStats
	if len(s.Stats) > 0 {
		stats = s.Stats
	}

	// Build a set for constant-time lookup of whether stats should be included
	filter := make(map[string]struct{})
	for _, s := range stats {
		filter[s] = struct{}{}
	}

	// Create a function that respects the kwAll by always returning true
	// if it is set
	return func(stat string) bool {
		if stats[0] == kwAll {
			return true
		}

		_, found := filter[stat]
		return found
	}
}

// Shell out to varnish_stat and return the output
func varnishStat() (*bytes.Buffer, error) {
	cmdName := "/usr/bin/varnishstat"
	cmdArgs := []string{"-1"}

	cmd := exec.Command(cmdName, cmdArgs...)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return &out, fmt.Errorf("error running varnishstat: %s", err)
	}

	return &out, nil
}

// Gather collects the configured stats from varnish_stat and adds them to the
// Accumulator
//
// The prefix of each stat (eg MAIN, MEMPOOL, LCK, etc) will be used as a
// 'section' tag and all stats that share that prefix will be reported as fields
// with that tag
func (s *Varnish) Gather(acc telegraf.Accumulator) error {

	out, err := varnishStat()
	if err != nil {
		return fmt.Errorf("error gathering metrics: %s", err)
	}

	statsFilter := s.statsFilter()
	sectionMap := make(map[string]map[string]interface{})
	scanner := bufio.NewScanner(out)
	for scanner.Scan() {
		cols := strings.Fields(scanner.Text())
		if len(cols) < 2 {
			continue
		}
		if !strings.Contains(cols[0], ".") {
			continue
		}

		stat := cols[0]
		value := cols[1]

		if !statsFilter(stat) {
			continue
		}

		parts := strings.SplitN(stat, ".", 2)
		section := parts[0]
		field := parts[1]

		// Init the section if necessary
		if _, ok := sectionMap[section]; !ok {
			sectionMap[section] = make(map[string]interface{})
		}

		sectionMap[section][field], err = strconv.Atoi(value)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Expected a numeric vlaue for %s = %v\n",
				stat, value)
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
