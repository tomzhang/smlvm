package g8

import (
	"errors"
	"path/filepath"

	"e8vm.io/e8vm/asm8"
	"e8vm.io/e8vm/build8"
	"e8vm.io/e8vm/lex8"
)

// MakeMemHome makes a memory home for compiling.
// It contains the basic built-in packages.
func MakeMemHome(lang build8.Lang) *build8.MemHome {
	home := build8.NewMemHome(lang)
	home.AddLang("asm", asm8.Lang())
	builtin := home.NewPkg("asm/builtin")
	builtin.AddFile("", "builtin.s", builtInSrc)

	return home
}

func buildMainPkg(home *build8.MemHome, runTests bool) (
	image []byte, errs []*lex8.Error, log []byte,
) {
	b := build8.NewBuilder(home, home)
	b.RunTests = runTests
	if errs := b.BuildAll(); errs != nil {
		return nil, errs, nil
	}

	image = home.BinBytes("main")
	log = home.OutputBytes("main", "ir")
	if image == nil {
		err := errors.New("missing main() function, no binary created")
		return nil, lex8.SingleErr(err), log
	}

	return image, nil, log
}

func buildSingle(fname, s string, lang build8.Lang, runTests bool) (
	image []byte, errs []*lex8.Error, log []byte,
) {
	home := MakeMemHome(lang)

	pkg := home.NewPkg("main")
	name := filepath.Base(fname)
	pkg.AddFile(fname, name, s)

	return buildMainPkg(home, runTests)
}

// CompileSingle compiles a file into a bare-metal E8 image
func CompileSingle(fname, s string, golike bool) (
	[]byte, []*lex8.Error, []byte,
) {
	return buildSingle(fname, s, Lang(golike), false)
}

// CompileAndTestSingle compiles a file into a bare-metal E8 image and
// runs the tests.
func CompileAndTestSingle(fname, s string, golike bool) (
	[]byte, []*lex8.Error, []byte,
) {
	return buildSingle(fname, s, Lang(golike), true)
}
