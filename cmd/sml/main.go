package main

import (
	"flag"
	"fmt"
	"os"

	"shanhu.io/smlvm/arch"
	"shanhu.io/smlvm/builds"
	"shanhu.io/smlvm/lexing"
	"shanhu.io/smlvm/srchome"
)

var (
	runTests = flag.Bool("test", true, "run tests")
	pkg      = flag.String("pkg", "/...", "package to build")
	homeDir  = flag.String("home", ".", "the home directory")
	plan     = flag.Bool("plan", false, "plan only")
	std      = flag.String("std", "", "standard library directory")
)

func handleErrs(errs []*lexing.Error) {
	if errs == nil {
		return
	}
	for _, err := range errs {
		fmt.Println(err)
	}
	os.Exit(-1)
}

func main() {
	flag.Parse()

	home := srchome.NewDirHome(*homeDir, *std)
	b := builds.NewBuilder(home, home)
	b.Verbose = true
	b.InitPC = arch.InitPC
	b.RunTests = *runTests

	pkgs, err := builds.SelectPkgs(home, *pkg)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}

	if !*plan {
		handleErrs(b.BuildPkgs(pkgs))
	} else {
		buildOrder, errs := b.Plan(pkgs)
		handleErrs(errs)
		for _, p := range buildOrder {
			fmt.Println(p)
		}
	}
}
