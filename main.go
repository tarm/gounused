package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	_ "golang.org/x/tools/go/gcimporter"
	"golang.org/x/tools/go/loader"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
	"golang.org/x/tools/go/types"
)

const debug = false

const usage = `
'%[1]s' finds unused assignements in your code.

The compiler checks for unused variables, but sometimes assignments
are never read before getting overwriten or ignored.  For example this
code:

   package main

   import "fmt"

   func main() {
        _, err := fmt.Println("Hello")
        _, err = fmt.Println(" world")
        _, err = fmt.Println(err)
   }

The err variable is used so the compiler does not complain, but the
first and third assignment to the err variable are never checked.
'%[1]s' finds that mistake as follows:

   $ %[1]s ./testdata/
   Unused assignment for 'err' ./testdata/test.go:6:5
   Unused assignment for 'err' ./testdata/test.go:8:5
   $ echo $?
   1
`

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, usage, os.Args[0])
	}
	flag.Parse()

	fail := myloader(flag.Args())
	if fail {
		os.Exit(1)
	}
}

// Return true if an unused var is found
func myloader(args []string) (failed bool) {
	var conf loader.Config
	conf.FromArgs(args, false)
	prog, err := conf.Load()
	if err != nil {
		log.Fatal(err)
	}

	for _, pi := range prog.Created {
		fmt.Println(pi)
	}

	info := prog.Package(args[0]).Info
	fset := prog.Fset

	ssaprog := ssautil.CreateProgram(prog, ssa.GlobalDebug)
	ssaprog.Build()

	fail := false
	for expr, object := range info.Uses {
		if _, ok := object.(*types.Var); !ok {
			continue
		}
		pkg, node, _ := prog.PathEnclosingInterval(expr.Pos(), expr.End())
		spkg := ssaprog.Package(pkg.Pkg)
		f := ssa.EnclosingFunction(spkg, node)
		if f == nil {
			fmt.Printf("Unknown function %v %v %v %v\n", fset.Position(expr.Pos()), object, pkg, prog)
			continue
		}
		value, _ := f.ValueForExpr(expr)
		// Unwrap unops and grab the value inside
		if v, ok := value.(*ssa.UnOp); ok {
			if debug {
				fmt.Println("Unwrapping unop")
			}
			value = v.X
		}
		if debug {
			fmt.Printf("%v %v: %v      %#v\n", fset.Position(expr.Pos()), expr, object, value)
		}
		if _, ok := value.(*ssa.Global); ok {
			if debug {
				fmt.Printf("     is global\n")
			}
			continue
		}
		if value == nil {
			continue
		}
		refs := value.Referrers()
		if refs == nil {
			continue
		}
		if debug {
			fmt.Printf("   (refs) %v\n", refs)
		}
		hasRef := false
		for _, r := range *refs {
			_, ok := r.(*ssa.DebugRef)
			hasRef = hasRef || !ok
			if debug && !ok {
				fmt.Printf("%v %v: %v      %v\n", fset.Position(expr.Pos()), expr, object, r)
			}
		}
		if !hasRef {
			fail = true
			fmt.Fprintf(os.Stderr, "Unused assignment for `%v` %v\n", expr, fset.Position(expr.Pos()))
		}
	}
	return fail
}
