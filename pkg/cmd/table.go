package cmd

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/olekukonko/tablewriter"
)

type table struct {
	header    []string
	lines     [][]string
	sortIndex int
	output    string
}

func (t table) Len() int           { return len(t.lines) }
func (t table) Swap(i, j int)      { t.lines[i], t.lines[j] = t.lines[j], t.lines[i] }
func (t table) Less(i, j int) bool { return t.lines[i][t.sortIndex] < t.lines[j][t.sortIndex] }

func convertToTable(t table) (string, error) {

	buff := &bytes.Buffer{}
	sort.Sort(t)

	switch t.output {
	case "raw":
		w := tabwriter.NewWriter(buff, 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "%s\n", strings.Join(t.header, "\t"))
		for _, line := range t.lines {
			fmt.Fprintf(w, "%s\n", strings.Join(line, "\t"))
		}
		w.Flush()
	case "markdown":
		table := tablewriter.NewWriter(buff)
		table.SetHeader(t.header)
		table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
		table.SetCenterSeparator("|")
		table.SetAutoWrapText(false)
		table.SetReflowDuringAutoWrap(false)
		table.SetAlignment(tablewriter.ALIGN_LEFT)
		table.AppendBulk(t.lines)
		table.Render()
	default:
		panic(fmt.Errorf("unknown output: %s", t.output))
	}

	return buff.String(), nil
}
