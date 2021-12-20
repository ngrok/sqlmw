// +build ignore

package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"time"
)

func main() {
	var err error
	fn := flag.String("o", "", "output file")
	flag.Parse()

	out := os.Stdout
	if *fn != "" {
		out, err = os.Create(*fn)
		if err != nil {
			log.Fatalf("could not create file %q, %v", *fn, err)
		}
	}

	intfs := []string{
		"NextResultSet",
		"ColumnTypeDatabaseTypeName",
		"ColumnTypeLength",
		"ColumnTypeNullable",
		"ColumnTypePrecisionScale",
		"ColumnTypeScanType",
	}

	genComment(out)
	fmt.Fprintln(out, "package sqlmw")

	fmt.Fprintln(out, "")
	fmt.Fprintln(out, "import (")
	fmt.Fprintln(out, "\t\"context\"")
	fmt.Fprintln(out, "\t\"database/sql/driver\"")
	fmt.Fprintln(out, ")")

	fmt.Fprintln(out, "")
	genConst(out, intfs)

	fmt.Fprintln(out, "")
	genPickerTable(out, intfs)

	fmt.Fprintln(out, "")
	genWrapRows(out, intfs)

	err = out.Close()
	if err != nil {
		log.Fatalf("could close file, %v", err)
	}
}

func genComment(w io.Writer) {
	str := time.Now().Format(time.Stamp)
	fmt.Fprintln(w, "// Code generated using tool/rows_picker_gen.go DO NOT EDIT.")
	fmt.Fprintf(w, "// Date: %s\n", str)
	fmt.Fprintln(w, "")
}

func genConst(w io.Writer, intfs []string) {
	fmt.Fprintln(w, "const (")
	for i, n := range intfs {
		suf := ""
		if i == 0 {
			suf = " = 1 << iota"
		}
		fmt.Fprintf(w, "\trows%s%s\n", n, suf)
	}
	fmt.Fprintln(w, ")")
}

func forEachBit(n int, intfs []string, f func(n int, intf string)) {
	for i := 0; i < len(intfs); i++ {
		b := 1 << i
		if b&n == b {
			f(n, intfs[i])
		}
	}
}

func genPickerTable(w io.Writer, intfs []string) {
	tlen := 1 << len(intfs)
	fmt.Fprintf(w, "var pickRows = make([]func(*wrappedRows) driver.Rows, %d)\n\n", tlen)

	fmt.Fprintln(w, "func init() {")
	defer fmt.Fprintln(w, "}")

	fmt.Fprintln(w, `
	// plain driver.Rows
	pickRows[0] = func(r *wrappedRows) driver.Rows {
		return r
	}`)

	for i := 1; i < tlen; i++ {
		fmt.Fprintf(w, `
	// plain driver.Rows
	pickRows[%d] = func(r *wrappedRows) driver.Rows {
		return struct {
			*wrappedRows`, i)
		fmt.Fprintln(w, "")
		forEachBit(i, intfs, func(_ int, intf string) {
			fmt.Fprintf(w, "\t\t\twrappedRows%s\n", intf)
		})
		fmt.Fprintln(w, "\t\t}{\n\t\t\tr,")
		forEachBit(i, intfs, func(_ int, intf string) {
			fmt.Fprintf(w, "\t\t\twrappedRows%s{r.parent},\n", intf)
		})
		fmt.Fprintln(w, "\t\t}")
		fmt.Fprintln(w, "\t}")
	}
}

func genWrapRows(w io.Writer, intfs []string) {
	fmt.Fprintln(w, "func wrapRows(ctx context.Context, intr Interceptor, r driver.Rows) driver.Rows {")
	fmt.Fprintln(w, `	or := r
	for {
		ur, ok := or.(RowsUnwrapper)
		if !ok {
			break
		}
		or = ur.Unwrap()
	}

	id := 0`)

	defer fmt.Fprintln(w, `
	wr := &wrappedRows{
		ctx:    ctx,
		intr:   intr,
		parent: r,
	}
	return pickRows[id](wr)
}`)

	for _, n := range intfs {
		fmt.Fprintf(w, `
	if _, ok := or.(driver.Rows%s); ok {
		id += rows%[1]s
	}`, n)
	}
}
