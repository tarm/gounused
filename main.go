package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"

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
	ssaprog := ssautil.CreateProgram(prog, 0)
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
					fmt.Println("  (inst): ", inst)
					fmt.Println("  (value): ", v.String(), v.Type())
					fmt.Println("  (referrers): ", v.Referrers())
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
