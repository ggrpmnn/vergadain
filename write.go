package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/fatih/color"
)

// FieldData.print prints out the content of a FieldData object in an
// easily readable format
func (fd *FieldData) Write(f *os.File) {
	// output to stdout
	if f == nil {
		field := color.New(color.BgGreen)
		field.Printf("Field Name: %s (ID: %s)", fd.Name, fd.ID)
		color.Unset()
		fmt.Println()
		for j := range fd.Values {
			color.Green("\tValue Name: %s (ID: %s)\n", fd.Values[j].Value, fd.Values[j].ID)
		}
	} else {
		w := bufio.NewWriter(f)
		defer w.Flush()
		w.WriteString(fmt.Sprintf("Field Name: %s (ID: %s)\n", fd.Name, fd.ID))
		for j := range fd.Values {
			w.WriteString(fmt.Sprintf("\tValue Name: %s (ID: %s)\n", fd.Values[j].Value, fd.Values[j].ID))
		}
	}
}

// WriteSeparator writes a specified separator in between FieldData values
func WriteSeparator(sep string, f *os.File) {
	if f == nil {
		fmt.Println(sep)
	} else {
		w := bufio.NewWriter(f)
		defer w.Flush()
		w.WriteString(fmt.Sprintln(sep))
	}
}
