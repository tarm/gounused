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

	// SSA Information
	ssaprog := ssautil.CreateProgram(prog, ssa.GlobalDebug)
	ssaprog.BuildAll()
	pkgs := ssaprog.AllPackages()
	if debug {
		for _, p := range pkgs {
			if p.Object.Name() == "main" {
				f := p.Func("main")
				if f == nil {
					log.Fatal("Could not find function")
				}
				for _, inst := range f.Blocks[0].Instrs {
					if v, ok := inst.(ssa.Value); ok {
						fmt.Println(v.Name())
						fmt.Printf("  (type)\t%v\n", v.Type())
						fmt.Printf("  (inst)\t%v\n", inst)
						fmt.Printf("  (referrers)\t%v\n", v.Referrers())
						fmt.Printf("  (pos)\t%v\n", v.Pos())
						fmt.Printf("  (debug)\t%#v\n", v)
					}
				}
			}
		}
		fmt.Println()
	}

	// The real meat of things...
	// Create a mapping from Defs to ssa.Values
	// Make sure that each
	for expr, object := range info.Uses {
		if _, ok := object.(*types.Var); !ok {
			continue
		}
		pkg, node, exact := prog.PathEnclosingInterval(expr.Pos(), expr.End())
		_ = exact // FIXME
		spkg := ssaprog.Package(pkg.Pkg)
		f := ssa.EnclosingFunction(spkg, node)

		value, _ := f.ValueForExpr(expr)
		if debug {
			fmt.Printf("%v %v: %v      %v\n", fset.Position(expr.Pos()), expr, object, value)
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
			fmt.Fprintf(os.Stderr, "Unused assignment for `%v` %v\n", expr, fset.Position(expr.Pos()))
		}

	}
}

const src = `package main

import "fmt"

func main() {
        _, err := fmt.Println("Hello")
        _, err = fmt.Println(err)
}
`
