// Copyright 2017 The Cockroach Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing
// permissions and limitations under the License.

// This file contains helper code to populate distsqlrun.Expressions during
// planning.

package distsqlplan

import (
	"bytes"
	"context"
	"fmt"

	"github.com/cockroachdb/cockroach/pkg/sql/distsqlrun"
	"github.com/cockroachdb/cockroach/pkg/sql/sem/tree"
	"github.com/cockroachdb/cockroach/pkg/util/log"
)

// exprFmtFlagsBase produces FmtFlags used for serializing expressions; a proper
// IndexedVar formatting function needs to be added on. It replaces placeholders
// with their values.
func exprFmtFlagsBase(evalCtx *tree.EvalContext) tree.FmtFlags {
	return tree.FmtPlaceholderFormat(
		tree.FmtCheckEquivalence,
		func(fmtCtx *tree.FmtCtx, p *tree.Placeholder) {
			d, err := p.Eval(evalCtx)
			if err != nil {
				panic(fmt.Sprintf("failed to serialize placeholder: %s", err))
			}
			d.Format(fmtCtx)
		})
}

// MakeExpression creates a distsqlrun.Expression.
//
// The distsqlrun.Expression uses the placeholder syntax (@1, @2, @3..) to refer
// to columns.
//
// The expr uses IndexedVars to refer to columns. The caller can optionally
// remap these columns by passing an indexVarMap: an IndexedVar with index i
// becomes column indexVarMap[i].
func MakeExpression(
	expr tree.TypedExpr, evalCtx *tree.EvalContext, indexVarMap []int,
) distsqlrun.Expression {
	if expr == nil {
		return distsqlrun.Expression{}
	}

	// We format the expression using the IndexedVar and Placeholder formatting interceptors.
	var f tree.FmtFlags
	if indexVarMap == nil {
		f = exprFmtFlagsBase(evalCtx)
	} else {
		f = tree.FmtIndexedVarFormat(
			exprFmtFlagsBase(evalCtx),
			func(ctx *tree.FmtCtx, idx int) {
				remappedIdx := indexVarMap[idx]
				if remappedIdx < 0 {
					panic(fmt.Sprintf("unmapped index %d", idx))
				}
				ctx.Printf("@%d", remappedIdx+1)
			},
		)
	}
	var buf bytes.Buffer
	fmtCtx := tree.MakeFmtCtx(&buf, f)
	fmtCtx.FormatNode(expr)
	if log.V(1) {
		log.Infof(context.TODO(), "Expr %s:\n%s", buf.String(), tree.ExprDebugString(expr))
	}
	return distsqlrun.Expression{Expr: buf.String()}
}
