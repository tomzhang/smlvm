package sempass

import (
	"shanhu.io/smlvm/lexing"
	"shanhu.io/smlvm/pl/ast"
	"shanhu.io/smlvm/pl/tast"
	"shanhu.io/smlvm/pl/types"
	"shanhu.io/smlvm/syms"
	"shanhu.io/smlvm/toposort"
)

type pkgStruct struct {
	name *lexing.Token
	ast  *ast.Struct    // the struct AST node
	sym  *syms.Symbol   // the symbol
	t    *types.Struct  // type
	pt   *types.Pointer // pointer type
	deps []string       // depending identifiers
}

func newPkgStruct(s *ast.Struct) *pkgStruct {
	deps := listStructDeps(s)
	t := types.NewStruct(s.Name.Lit)

	return &pkgStruct{
		name: s.Name,
		ast:  s,
		deps: deps,
		t:    t,
		pt:   types.NewPointer(t),
	}
}

func declareStruct(b *builder, s *ast.Struct) *pkgStruct {
	ret := newPkgStruct(s)
	name := ret.name.Lit
	pos := ret.name.Pos
	t := &types.Type{ret.t}
	sym := syms.Make(b.path, name, tast.SymStruct, nil, t, pos)
	conflict := b.scope.Declare(sym)
	if conflict != nil {
		b.Errorf(pos, "%s already defined", name)
		b.Errorf(conflict.Pos, "previously defined here as a %s",
			tast.SymStr(conflict.Type),
		)
		return nil
	}

	ret.sym = sym
	return ret
}

func sortStructs(b *builder, m map[string]*pkgStruct) []*pkgStruct {
	s := toposort.NewSorter("struct")
	for name, ps := range m {
		s.AddNode(name, ps.name, ps.deps)
	}

	order := s.Sort(b)
	ret := make([]*pkgStruct, 0, len(order))
	for _, name := range order {
		ret = append(ret, m[name])
	}
	return ret
}

func buildFields(b *builder, ps *pkgStruct) {
	s := ps.ast
	t := ps.t

	for _, f := range s.Fields {
		ft := b.buildType(f.Type)
		if ft == nil {
			continue
		}

		for _, id := range f.Idents.Idents {
			name := id.Lit
			field := &types.Field{Name: name, T: ft}
			sym := syms.Make(b.path, name, tast.SymField, field, ft, id.Pos)
			conflict := t.Syms.Declare(sym)
			if conflict != nil {
				b.Errorf(id.Pos, "field %s already defined", id.Lit)
				b.Errorf(conflict.Pos, "previously defined here")
				continue
			}

			t.AddField(field)
		}
	}
}

func buildStructs(b *builder, structs []*ast.Struct) []*pkgStruct {
	m := make(map[string]*pkgStruct)
	for _, s := range structs {
		ps := declareStruct(b, s)
		if ps != nil {
			m[ps.name.Lit] = ps
		}
	}

	ret := sortStructs(b, m)
	for _, ps := range ret {
		buildFields(b, ps)
	}
	return ret
}
