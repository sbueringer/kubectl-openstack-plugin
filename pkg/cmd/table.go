package cmd

import (
	"bytes"
	"text/tabwriter"
	"sort"
	"fmt"
	"strings"
)

type table struct {
	header    []string
	lines     [][]string
	sortIndex int
}

func (t table) Len() int           { return len(t.lines) }
func (t table) Swap(i, j int)      { t.lines[i], t.lines[j] = t.lines[j], t.lines[i] }
func (t table) Less(i, j int) bool { return t.lines[i][t.sortIndex] < t.lines[j][t.sortIndex] }

func printTable(t table) (string, error) {

	buff := &bytes.Buffer{}
	w := tabwriter.NewWriter(buff, 0, 0, 2, ' ', 0)

	sort.Sort(t)

	fmt.Fprintf(w, "%s\n", strings.Join(t.header, "\t"))

	for _, line := range t.lines {
		fmt.Fprintf(w, "%s\n", strings.Join(line, "\t"))
	}

	w.Flush()
	return buff.String(), nil
}
