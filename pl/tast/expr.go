package tast

import (
	"shanhu.io/smlvm/lexing"
	"shanhu.io/smlvm/pl/types"
	"shanhu.io/smlvm/syms"
)

// This is the this pointer.
type This struct{ *Ref }

// Const is a constant.
type Const struct{ *Ref }

// Type is a type expression
type Type struct{ *Ref }

// NewType creates a new type expression of a particular type.
func NewType(t types.T) *Type {
	return &Type{NewTypeRef(t)}
}

// Cast cast from one type of reference to another
type Cast struct {
	From Expr
	*Ref
}

// NewCast creates a new casting operation
func NewCast(from Expr, to types.T) *Cast {
	return &Cast{from, NewRef(to)}
}

// Ident is an identifier.
type Ident struct {
	Token *lexing.Token
	*Ref
	Symbol *syms.Symbol
}

// MemberExpr is an expression of "a.b"
type MemberExpr struct {
	Expr Expr
	Sub  *lexing.Token
	*Ref
	Symbol *syms.Symbol
}

// OpExpr is an expression likfe "a+b"
type OpExpr struct {
	A  Expr
	Op *lexing.Token
	B  Expr
	*Ref
}

// StarExpr is an expression like "*a"
type StarExpr struct {
	Expr Expr
	*Ref
}

// CallExpr is an expression like "f(x)"
type CallExpr struct {
	Func Expr
	Args Expr
	*Ref
}

// IndexExpr is an expression like "a[b:c]"
// Both b and c are optional.
type IndexExpr struct {
	Array, Index, IndexEnd Expr
	HasColon               bool
	*Ref
}

// ExprList is a list of expressions.
type ExprList struct {
	Exprs []Expr
	*Ref
}

// Len returns the length of the expression list.
func (lst *ExprList) Len() int {
	return len(lst.Exprs)
}

// NewExprList creates a new expression list.
func NewExprList() *ExprList {
	return &ExprList{Ref: Void}
}

// Append appends an expression into the expression list.
func (lst *ExprList) Append(expr Expr) {
	lst.Ref = AppendRef(lst.Ref, expr.R())
	lst.Exprs = append(lst.Exprs, expr)
}

// MakeExprList makes the expression an expression list if it not one
// yet.
func MakeExprList(expr Expr) (*ExprList, bool) {
	ret, ok := expr.(*ExprList)
	if ok {
		return ret, true
	}
	if expr.R().Len() == 1 {
		ret := NewExprList()
		ret.Append(expr)
		return ret, true
	}
	return nil, false
}
