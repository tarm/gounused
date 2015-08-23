package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"

	_ "golang.org/x/tools/go/gcimporter"
	"golang.org/x/tools/go/loader"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
	"golang.org/x/tools/go/types"
)

func main() {
	fset := token.NewFileSet() // positions are relative to fset
	fset.AddFile("main.go", -1, len(src))
	f, err := parser.ParseFile(fset, "main.go", src, 0)
	if err != nil {
		log.Fatal(err)
	}
	// Print the AST.
	ast.Print(fset, f)
	fmt.Println()

	// Type information
	info := &types.Info{
		Defs: make(map[*ast.Ident]types.Object),
		Uses: make(map[*ast.Ident]types.Object),
	}
	var tconf types.Config
	_, err = tconf.Check("main.go", fset, []*ast.File{f}, info)
	if err != nil {
		log.Fatal(err)
	}
	for k, v := range info.Defs {
		fmt.Println(fset.Position(k.Pos()), k, ":", v)
	}
	fmt.Println("Uses")
	for k, v := range info.Uses {
		fmt.Println(fset.Position(k.Pos()), k, ":", v)
	}
	fmt.Println()

	// SSA Information
	var conf loader.Config
	conf.CreateFromFiles("main", f)
	prog, err := conf.Load()
	if err != nil {
		log.Fatal(err)
	}
	ssaprog := ssautil.CreateProgram(prog, ssa.GlobalDebug)
	ssaprog.BuildAll()
	pkgs := ssaprog.AllPackages()
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

	// The real meat of things...
	// Create a mapping from Defs to ssa.Values
	// Make sure that each
	for _, p := range pkgs {
		if p.Object.Name() == "main" {
			f := p.Func("main")
			if f == nil {
				log.Fatal("Could not find function")
			}
			for expr, object := range info.Uses {
				if _, ok := object.(*types.Var); !ok {
					continue
				}
				value, _ := f.ValueForExpr(expr)
				fmt.Printf("%v %v: %v      %v\n", fset.Position(expr.Pos()), expr, object, value)
				if value == nil {
					continue
				}
				refs := value.Referrers()
				if refs == nil {
					continue
				}
				fmt.Printf("   (refs) %v\n", refs)
				hasRef := false
				for _, r := range *refs {
					_, ok := r.(*ssa.DebugRef)
					hasRef = hasRef || !ok
				}
				if !hasRef {
					fmt.Fprintf(os.Stderr, "\nUnused assignment for %q %v\n\n", expr, fset.Position(expr.Pos()))
				}
			}
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
