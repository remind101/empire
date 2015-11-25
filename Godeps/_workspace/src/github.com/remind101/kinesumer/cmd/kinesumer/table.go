package main

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
)

type Table struct {
	Rows []Row
}

type Row struct {
	Cells  []Cell
	Header bool
}

type Cell struct {
	Color *color.Color
	Text  string
}

func NewTable() *Table {
	return &Table{
		Rows: make([]Row, 0),
	}
}

func (t *Table) Done() {
	cols := 0
	for _, row := range t.Rows {
		if cols < len(row.Cells) {
			cols = len(row.Cells)
		}
	}

	widths := make([]int, cols)
	for _, row := range t.Rows {
		for col, cell := range row.Cells {
			if widths[col] < len(cell.Text) {
				widths[col] = len(cell.Text)
			}
		}
	}

	for _, row := range t.Rows {
		for col, cell := range row.Cells {
			if row.Header {
				cell.Color.Printf("%s%s  ", strings.ToUpper(cell.Text), strings.Repeat(" ", widths[col]-len(cell.Text)))
			} else {
				cell.Color.Printf("%s%s  ", cell.Text, strings.Repeat(" ", widths[col]-len(cell.Text)))
			}
		}
		fmt.Println()
	}
}

func (t *Table) AddRow() *Row {
	t.Rows = append(t.Rows, Row{
		Cells: make([]Cell, 0),
	})
	return &t.Rows[len(t.Rows)-1]
}

func (t *Table) AddRowWith(labels ...string) *Row {
	row := t.AddRow()
	for _, label := range labels {
		row.AddCellWithf("%s", label)
	}
	return row
}

func (r *Row) AddCell() *Cell {
	r.Cells = append(r.Cells, Cell{
		Color: color.New(),
		Text:  "-",
	})
	return &r.Cells[len(r.Cells)-1]
}

func (r *Row) AddCellWithf(format string, a ...interface{}) *Cell {
	cell := r.AddCell()
	cell.Printf(format, a...)
	return cell
}

func (c *Cell) Printf(format string, a ...interface{}) {
	c.Text = fmt.Sprintf(format, a...)
}

func StrShorten(s string, pre, post int) string {
	if len(s) <= pre+post+3 {
		return s
	}
	return s[0:pre] + "..." + s[len(s)-post:len(s)]
}
