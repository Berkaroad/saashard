// Copyright 2012, Google Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Copyright 2016 The kingshard Authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"): you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

// The MIT License (MIT)

// Copyright (c) 2016 Jerry Bai

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package sqlparser

import (
	"errors"
	"strconv"

	"github.com/berkaroad/saashard/sqlparser/sqltypes"
)

// Instructions for creating new types: If a type
// needs to satisfy an interface, declare that function
// along with that interface. This will help users
// identify the list of types to which they can assert
// those interfaces.
// If the member of a type has a string with a predefined
// list of values, declare those values as const following
// the type.
// For interfaces that define dummy functions to consolidate
// a set of types, define the function as ITypeName.
// This will help avoid name collisions.

// Parse parses the sql and returns a Statement, which
// is the AST representation of the query.
func Parse(sql string) (Statement, error) {
	// yyDebug = 4
	tokenizer := NewStringTokenizer(sql)
	if yyParse(tokenizer) != 0 {
		return nil, errors.New(tokenizer.LastError)
	}
	return tokenizer.ParseTree, nil
}

// SQLNode defines the interface for all nodes
// generated by the parser.
type SQLNode interface {
	Format(buf *TrackedBuffer)
}

// String returns a string representation of an SQLNode.
func String(node SQLNode) string {
	buf := NewTrackedBuffer(nil)
	buf.Fprintf("%v", node)
	return buf.String()
}

// Statement represents a statement.
type Statement interface {
	IStatement()
	SQLNode
}

// InsertRows represents the rows for an INSERT statement.
type InsertRows interface {
	IInsertRows()
	SQLNode
}

func (Values) IInsertRows() {}

// Comments represents a list of comments.
type Comments [][]byte

func (node Comments) Format(buf *TrackedBuffer) {
	for _, c := range node {
		buf.Fprintf("%s ", c)
	}
}

// SelectExprs represents SELECT expressions.
type SelectExprs []SelectExpr

func (node SelectExprs) Format(buf *TrackedBuffer) {
	var prefix string
	for _, n := range node {
		buf.Fprintf("%s%v", prefix, n)
		prefix = ", "
	}
}

// SelectExpr represents a SELECT expression.
type SelectExpr interface {
	ISelectExpr()
	SQLNode
}

func (*StarExpr) ISelectExpr()    {}
func (*NonStarExpr) ISelectExpr() {}

// StarExpr defines a '*' or 'table.*' expression.
type StarExpr struct {
	TableName []byte
}

func (node *StarExpr) Format(buf *TrackedBuffer) {
	if node.TableName != nil {
		buf.Fprintf("%s.", node.TableName)
	}
	buf.Fprintf("*")
}

// NonStarExpr defines a non-'*' select expr.
type NonStarExpr struct {
	Expr Expr
	As   []byte
}

func (node *NonStarExpr) Format(buf *TrackedBuffer) {
	buf.Fprintf("%v", node.Expr)
	if node.As != nil {
		buf.Fprintf(" as %s", node.As)
	}
}

// Columns represents an insert column list.
// The syntax for Columns is a subset of SelectExprs.
// So, it's castable to a SelectExprs and can be analyzed
// as such.
type Columns []SelectExpr

func (node Columns) Format(buf *TrackedBuffer) {
	if node == nil {
		return
	}
	buf.Fprintf("(%v)", SelectExprs(node))
}

// TableExprs represents a list of table expressions.
type TableExprs []TableExpr

func (node TableExprs) Format(buf *TrackedBuffer) {
	var prefix string
	for _, n := range node {
		buf.Fprintf("%s%v", prefix, n)
		prefix = ", "
	}
}

// TableExpr represents a table expression.
type TableExpr interface {
	ITableExpr()
	SQLNode
}

func (*AliasedTableExpr) ITableExpr() {}
func (*ParenTableExpr) ITableExpr()   {}
func (*JoinTableExpr) ITableExpr()    {}

// AliasedTableExpr represents a table expression
// coupled with an optional alias or index hint.
type AliasedTableExpr struct {
	Expr  SimpleTableExpr
	As    []byte
	Hints *IndexHints
}

func (node *AliasedTableExpr) Format(buf *TrackedBuffer) {
	buf.Fprintf("%v", node.Expr)
	if node.As != nil {
		buf.Fprintf(" as %s", node.As)
	}
	if node.Hints != nil {
		// Hint node provides the space padding.
		buf.Fprintf("%v", node.Hints)
	}
}

// SimpleTableExpr represents a simple table expression.
type SimpleTableExpr interface {
	ISimpleTableExpr()
	SQLNode
}

func (*TableName) ISimpleTableExpr() {}
func (*Subquery) ISimpleTableExpr()  {}

// TableName represents a table  name.
type TableName struct {
	Name, Qualifier []byte
}

func (node *TableName) Format(buf *TrackedBuffer) {
	if node.Qualifier != nil {
		escape(buf, node.Qualifier)
		buf.Fprintf(".")
	}
	escape(buf, node.Name)
}

// ParenTableExpr represents a parenthesized TableExpr.
type ParenTableExpr struct {
	Expr TableExpr
}

func (node *ParenTableExpr) Format(buf *TrackedBuffer) {
	buf.Fprintf("(%v)", node.Expr)
}

// JoinTableExpr represents a TableExpr that's a JOIN operation.
type JoinTableExpr struct {
	LeftExpr  TableExpr
	Join      string
	RightExpr TableExpr
	On        BoolExpr
}

// JoinTableExpr.Join
const (
	AST_JOIN          = "join"
	AST_STRAIGHT_JOIN = "straight_join"
	AST_LEFT_JOIN     = "left join"
	AST_RIGHT_JOIN    = "right join"
	AST_CROSS_JOIN    = "cross join"
	AST_NATURAL_JOIN  = "natural join"
)

func (node *JoinTableExpr) Format(buf *TrackedBuffer) {
	buf.Fprintf("%v %s %v", node.LeftExpr, node.Join, node.RightExpr)
	if node.On != nil {
		buf.Fprintf(" on %v", node.On)
	}
}

// IndexHints represents a list of index hints.
type IndexHints struct {
	Type    string
	Indexes [][]byte
}

const (
	AST_USE    = "use"
	AST_IGNORE = "ignore"
	AST_FORCE  = "force"
)

func (node *IndexHints) Format(buf *TrackedBuffer) {
	buf.Fprintf(" %s index ", node.Type)
	prefix := "("
	for _, n := range node.Indexes {
		buf.Fprintf("%s%s", prefix, n)
		prefix = ", "
	}
	buf.Fprintf(")")
}

// Where represents a WHERE or HAVING clause.
type Where struct {
	Type string
	Expr BoolExpr
}

// Where.Type
const (
	AST_WHERE  = "where"
	AST_HAVING = "having"
)

// NewWhere creates a WHERE or HAVING clause out
// of a BoolExpr. If the expression is nil, it returns nil.
func NewWhere(typ string, expr BoolExpr) *Where {
	if expr == nil {
		return nil
	}
	return &Where{Type: typ, Expr: expr}
}

func (node *Where) Format(buf *TrackedBuffer) {
	if node == nil {
		return
	}
	buf.Fprintf(" %s %v", node.Type, node.Expr)
}

// LikeExpr Expression
type LikeExpr struct {
	Expr ValExpr
}

func (node *LikeExpr) Format(buf *TrackedBuffer) {
	if node == nil {
		return
	}
	buf.Fprintf(" like %v", node.Expr)
}

// WhereExpr Expression
type WhereExpr struct {
	Expr BoolExpr
}

func (node *WhereExpr) Format(buf *TrackedBuffer) {
	if node == nil {
		return
	}
	buf.Fprintf(" where %v", node.Expr)
}

// Expr represents an expression.
type Expr interface {
	IExpr()
	SQLNode
}

func (*AndExpr) IExpr()        {}
func (*OrExpr) IExpr()         {}
func (*NotExpr) IExpr()        {}
func (*ParenBoolExpr) IExpr()  {}
func (*ComparisonExpr) IExpr() {}
func (*RangeCond) IExpr()      {}
func (*NullCheck) IExpr()      {}
func (*ExistsExpr) IExpr()     {}
func (StrVal) IExpr()          {}
func (NumVal) IExpr()          {}
func (ValArg) IExpr()          {}
func (*NullVal) IExpr()        {}
func (*ColName) IExpr()        {}
func (ValTuple) IExpr()        {}
func (*Subquery) IExpr()       {}
func (*BinaryExpr) IExpr()     {}
func (*UnaryExpr) IExpr()      {}
func (*FuncExpr) IExpr()       {}
func (*CaseExpr) IExpr()       {}
func (*LikeExpr) IExpr()       {}
func (*WhereExpr) IExpr()      {}

// BoolExpr represents a boolean expression.
type BoolExpr interface {
	IBoolExpr()
	Expr
}

func (*AndExpr) IBoolExpr()        {}
func (*OrExpr) IBoolExpr()         {}
func (*NotExpr) IBoolExpr()        {}
func (*ParenBoolExpr) IBoolExpr()  {}
func (*ComparisonExpr) IBoolExpr() {}
func (*RangeCond) IBoolExpr()      {}
func (*NullCheck) IBoolExpr()      {}
func (*ExistsExpr) IBoolExpr()     {}

// AndExpr represents an AND expression.
type AndExpr struct {
	Left, Right BoolExpr
}

func (node *AndExpr) Format(buf *TrackedBuffer) {
	buf.Fprintf("%v and %v", node.Left, node.Right)
}

// OrExpr represents an OR expression.
type OrExpr struct {
	Left, Right BoolExpr
}

func (node *OrExpr) Format(buf *TrackedBuffer) {
	buf.Fprintf("%v or %v", node.Left, node.Right)
}

// NotExpr represents a NOT expression.
type NotExpr struct {
	Expr BoolExpr
}

func (node *NotExpr) Format(buf *TrackedBuffer) {
	buf.Fprintf("not %v", node.Expr)
}

// ParenBoolExpr represents a parenthesized boolean expression.
type ParenBoolExpr struct {
	Expr BoolExpr
}

func (node *ParenBoolExpr) Format(buf *TrackedBuffer) {
	buf.Fprintf("(%v)", node.Expr)
}

// ComparisonExpr represents a two-value comparison expression.
type ComparisonExpr struct {
	Operator    string
	Left, Right ValExpr
}

// ComparisonExpr.Operator
const (
	AST_EQ       = "="
	AST_LT       = "<"
	AST_GT       = ">"
	AST_LE       = "<="
	AST_GE       = ">="
	AST_NE       = "!="
	AST_NSE      = "<=>"
	AST_IN       = "in"
	AST_NOT_IN   = "not in"
	AST_LIKE     = "like"
	AST_NOT_LIKE = "not like"
)

func (node *ComparisonExpr) Format(buf *TrackedBuffer) {
	buf.Fprintf("%v %s %v", node.Left, node.Operator, node.Right)
}

// RangeCond represents a BETWEEN or a NOT BETWEEN expression.
type RangeCond struct {
	Operator string
	Left     ValExpr
	From, To ValExpr
}

// RangeCond.Operator
const (
	AST_BETWEEN     = "between"
	AST_NOT_BETWEEN = "not between"
)

func (node *RangeCond) Format(buf *TrackedBuffer) {
	buf.Fprintf("%v %s %v and %v", node.Left, node.Operator, node.From, node.To)
}

// NullCheck represents an IS NULL or an IS NOT NULL expression.
type NullCheck struct {
	Operator string
	Expr     ValExpr
}

// NullCheck.Operator
const (
	AST_IS_NULL     = "is null"
	AST_IS_NOT_NULL = "is not null"
)

func (node *NullCheck) Format(buf *TrackedBuffer) {
	buf.Fprintf("%v %s", node.Expr, node.Operator)
}

// ExistsExpr represents an EXISTS expression.
type ExistsExpr struct {
	Subquery *Subquery
}

func (node *ExistsExpr) Format(buf *TrackedBuffer) {
	buf.Fprintf("exists %v", node.Subquery)
}

// ValExpr represents a value expression.
type ValExpr interface {
	IValExpr()
	Expr
}

func (StrVal) IValExpr()      {}
func (NumVal) IValExpr()      {}
func (ValArg) IValExpr()      {}
func (*NullVal) IValExpr()    {}
func (*ColName) IValExpr()    {}
func (ValTuple) IValExpr()    {}
func (*Subquery) IValExpr()   {}
func (*BinaryExpr) IValExpr() {}
func (*UnaryExpr) IValExpr()  {}
func (*FuncExpr) IValExpr()   {}
func (*CaseExpr) IValExpr()   {}

// StrVal represents a string value.
type StrVal []byte

func (node StrVal) Format(buf *TrackedBuffer) {
	s := sqltypes.MakeString([]byte(node))
	s.EncodeSQL(buf)
}

// NumVal represents a number.
type NumVal []byte

func (node NumVal) Format(buf *TrackedBuffer) {
	buf.Fprintf("%s", []byte(node))
}

// ValArg represents a named bind var argument.
type ValArg []byte

func (node ValArg) Format(buf *TrackedBuffer) {
	buf.WriteArg(string(node[1:]))
}

// NullVal represents a NULL value.
type NullVal struct{}

func (node *NullVal) Format(buf *TrackedBuffer) {
	buf.Fprintf("null")
}

// ColNames ColName list
type ColNames []*ColName

// Format ColNames
func (node ColNames) Format(buf *TrackedBuffer) {
	var prefix string
	for _, n := range node {
		buf.Fprintf("%s%v", prefix, n)
		prefix = ","
	}
}

// ColName represents a column name.
type ColName struct {
	Name, Qualifier []byte
}

func (node *ColName) Format(buf *TrackedBuffer) {
	if node.Qualifier != nil {
		escape(buf, node.Qualifier)
		buf.Fprintf(".")
	}
	escape(buf, node.Name)
}

func escape(buf *TrackedBuffer, name []byte) {
	if _, ok := keywords[string(name)]; ok {
		buf.Fprintf("`%s`", name)
	} else {
		needQuota := false
		for _, ch := range name {
			if !isLetter(uint16(ch)) && !isDigit(uint16(ch)) {
				needQuota = true
				break
			}
		}
		if needQuota {
			buf.Fprintf("`%s`", name)
		} else {
			buf.Fprintf("%s", name)
		}
	}
}

// Tuple represents a tuple. It can be ValTuple, Subquery.
type Tuple interface {
	ITuple()
	ValExpr
}

func (ValTuple) ITuple()  {}
func (*Subquery) ITuple() {}

// ValTuple represents a tuple of actual values.
type ValTuple ValExprs

func (node ValTuple) Format(buf *TrackedBuffer) {
	buf.Fprintf("(%v)", ValExprs(node))
}

// ValExprs represents a list of value expressions.
// It's not a valid expression because it's not parenthesized.
type ValExprs []ValExpr

func (node ValExprs) Format(buf *TrackedBuffer) {
	var prefix string
	for _, n := range node {
		buf.Fprintf("%s%v", prefix, n)
		prefix = ", "
	}
}

// Subquery represents a subquery.
type Subquery struct {
	Select SelectStatement
}

func (node *Subquery) Format(buf *TrackedBuffer) {
	buf.Fprintf("(%v)", node.Select)
}

// BinaryExpr represents a binary value expression.
type BinaryExpr struct {
	Operator    byte
	Left, Right Expr
}

// BinaryExpr.Operator
const (
	AST_BITAND = '&'
	AST_BITOR  = '|'
	AST_BITXOR = '^'
	AST_PLUS   = '+'
	AST_MINUS  = '-'
	AST_MULT   = '*'
	AST_DIV    = '/'
	AST_MOD    = '%'
)

func (node *BinaryExpr) Format(buf *TrackedBuffer) {
	buf.Fprintf("%v%c%v", node.Left, node.Operator, node.Right)
}

// UnaryExpr represents a unary value expression.
type UnaryExpr struct {
	Operator byte
	Expr     Expr
}

// UnaryExpr.Operator
const (
	AST_UPLUS  = '+'
	AST_UMINUS = '-'
	AST_TILDA  = '~'
)

func (node *UnaryExpr) Format(buf *TrackedBuffer) {
	buf.Fprintf("%c%v", node.Operator, node.Expr)
}

// FuncExpr represents a function call.
type FuncExpr struct {
	Name     []byte
	Distinct bool
	Exprs    ValExprs
}

func (node *FuncExpr) Format(buf *TrackedBuffer) {
	var distinct string
	if node.Distinct {
		distinct = "distinct "
	}
	buf.Fprintf("%s(%s%v)", node.Name, distinct, node.Exprs)
}

// CaseExpr represents a CASE expression.
type CaseExpr struct {
	Expr  ValExpr
	Whens []*When
	Else  ValExpr
}

func (node *CaseExpr) Format(buf *TrackedBuffer) {
	buf.Fprintf("case ")
	if node.Expr != nil {
		buf.Fprintf("%v ", node.Expr)
	}
	for _, when := range node.Whens {
		buf.Fprintf("%v ", when)
	}
	if node.Else != nil {
		buf.Fprintf("else %v ", node.Else)
	}
	buf.Fprintf("end")
}

// When represents a WHEN sub-expression.
type When struct {
	Cond BoolExpr
	Val  ValExpr
}

func (node *When) Format(buf *TrackedBuffer) {
	buf.Fprintf("when %v then %v", node.Cond, node.Val)
}

// Values represents a VALUES clause.
type Values []Tuple

func (node Values) Format(buf *TrackedBuffer) {
	prefix := "values "
	for _, n := range node {
		buf.Fprintf("%s%v", prefix, n)
		prefix = ", "
	}
}

// GroupBy represents a GROUP BY clause.
type GroupBy []ValExpr

func (node GroupBy) Format(buf *TrackedBuffer) {
	prefix := " group by "
	for _, n := range node {
		buf.Fprintf("%s%v", prefix, n)
		prefix = ", "
	}
}

// OrderBy represents an ORDER By clause.
type OrderBy []*Order

func (node OrderBy) Format(buf *TrackedBuffer) {
	prefix := " order by "
	for _, n := range node {
		buf.Fprintf("%s%v", prefix, n)
		prefix = ", "
	}
}

// Order represents an ordering expression.
type Order struct {
	Expr      ValExpr
	Direction string
}

// Order.Direction
const (
	AST_ASC  = "asc"
	AST_DESC = "desc"
)

func (node *Order) Format(buf *TrackedBuffer) {
	buf.Fprintf("%v %s", node.Expr, node.Direction)
}

// Limit represents a LIMIT clause.
type Limit struct {
	Offset, Rowcount ValExpr
}

func (node *Limit) RewriteLimit() (*Limit, error) {
	if node == nil {
		return nil, nil
	}

	var offset, count int64
	var err error
	newLimit := new(Limit)

	if node.Offset == nil {
		offset = 0
	} else {
		o, ok := node.Offset.(NumVal)
		if !ok {
			return nil, errors.New("Limit.offset is not number")
		}
		if offset, err = strconv.ParseInt(string([]byte(o)), 10, 64); err != nil {
			return nil, err
		}
	}

	r, ok := node.Rowcount.(NumVal)
	if !ok {
		return nil, errors.New("Limit.RowCount is not number")
	}
	if count, err = strconv.ParseInt(string([]byte(r)), 10, 64); err != nil {
		return nil, err
	}

	allRowCount := strconv.FormatInt((offset + count), 10)
	newLimit.Rowcount = NumVal(allRowCount)

	return newLimit, nil
}

func (node *Limit) Format(buf *TrackedBuffer) {
	if node == nil {
		return
	}
	buf.Fprintf(" limit ", nil)
	if node.Offset != nil {
		buf.Fprintf("%v, ", node.Offset)
	}
	buf.Fprintf("%v", node.Rowcount)
}

// UpdateExprs represents a list of update expressions.
type UpdateExprs []*UpdateExpr

func (node UpdateExprs) Format(buf *TrackedBuffer) {
	var prefix string
	for _, n := range node {
		buf.Fprintf("%s%v", prefix, n)
		prefix = ", "
	}
}

// UpdateExpr represents an update expression.
type UpdateExpr struct {
	Name *ColName
	Expr ValExpr
}

func (node *UpdateExpr) Format(buf *TrackedBuffer) {
	buf.Fprintf("%v = %v", node.Name, node.Expr)
}

// SpaceSplitExprs represents a list of space split expressions.
type SpaceSplitExprs []*SpaceSplitExpr

func (node SpaceSplitExprs) Format(buf *TrackedBuffer) {
	var prefix string
	for _, n := range node {
		buf.Fprintf("%s%v", prefix, n)
		prefix = " "
	}
}

// SpaceSplitExpr represents an space split expression.
type SpaceSplitExpr struct {
	Name string
	Expr []byte
}

func (node *SpaceSplitExpr) Format(buf *TrackedBuffer) {
	buf.Fprintf("%s %s", node.Name, node.Expr)
}

// OnDup represents an ON DUPLICATE KEY clause.
type OnDup UpdateExprs

func (node OnDup) Format(buf *TrackedBuffer) {
	if node == nil {
		return
	}
	buf.Fprintf(" on duplicate key update %v", UpdateExprs(node))
}
