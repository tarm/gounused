package main

import (
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

func main() {
	var conf loader.Config
	conf.FromArgs(os.Args[1:], false)
	prog, err := conf.Load()
	if err != nil {
		log.Fatal(err)
	}

	for _, pi := range prog.Created {
		fmt.Println(pi)
	}

	info := prog.Package(os.Args[1]).Info
	fset := prog.Fset

	ssaprog := ssautil.CreateProgram(prog, ssa.GlobalDebug)
	ssaprog.BuildAll()

	fail := false
	for expr, object := range info.Uses {
		if _, ok := object.(*types.Var); !ok {
			continue
		}
		pkg, node, exact := prog.PathEnclosingInterval(expr.Pos(), expr.End())
		_ = exact // FIXME
		spkg := ssaprog.Package(pkg.Pkg)
		f := ssa.EnclosingFunction(spkg, node)
		if f == nil {
			fmt.Printf("Unknown function %v %v %v %v\n", fset.Position(expr.Pos()), object, pkg, prog)
			continue
		}
		value, _ := f.ValueForExpr(expr)
		// Unwrap unops and grab the value inside
		if v, ok := value.(*ssa.UnOp); ok {
			//fmt.Println("Unwrapping unop")
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
	if fail {
		os.Exit(1)
	}
}
