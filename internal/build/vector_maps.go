package build

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	"github.com/xgo-dev/llgo/internal/packages"
	llssa "github.com/xgo-dev/llgo/ssa"
	"golang.org/x/tools/go/ssa"
)

func validateVectorMaps(prog llssa.Program, groups ...[]*packages.Package) error {
	seen := make(map[*packages.Package]bool)
	validate := prog.VectorMapValidator()
	var firstErr error
	for _, roots := range groups {
		packages.Visit(roots, nil, func(pkg *packages.Package) {
			if firstErr != nil || seen[pkg] || pkg.TypesInfo == nil || pkg.IllTyped {
				return
			}
			seen[pkg] = true
			// Walk source order for a stable, positioned diagnostic, including
			// unused type declarations and instantiated generic containers.
			for _, file := range pkg.Syntax {
				ast.Inspect(file, func(node ast.Node) bool {
					if firstErr != nil {
						return false
					}
					expr, ok := node.(ast.Expr)
					if !ok {
						return true
					}
					if typ := pkg.TypesInfo.TypeOf(expr); typ != nil {
						if err := validate(typ); err != nil {
							firstErr = fmt.Errorf("%s: %w", pkg.Fset.Position(expr.Pos()), err)
							return false
						}
					}
					return true
				})
			}
		})
	}
	return firstErr
}

// Source types still contain type parameters inside generic bodies. Check the
// concrete SSA instances before LLVM lowering, including methods and closures.
func validateSSAVectorMaps(fn *ssa.Function, validate func(types.Type) error) error {
	check := func(typ types.Type, pos token.Pos) error {
		if err := validate(typ); err != nil {
			if !pos.IsValid() {
				pos = fn.Pos()
			}
			return fmt.Errorf("%s: %w", fn.Prog.Fset.Position(pos), err)
		}
		return nil
	}
	if err := check(fn.Signature, fn.Pos()); err != nil {
		return err
	}
	var operands []*ssa.Value
	for _, block := range fn.Blocks {
		for _, instruction := range block.Instrs {
			if value, ok := instruction.(ssa.Value); ok {
				if err := check(value.Type(), instruction.Pos()); err != nil {
					return err
				}
			}
			operands = instruction.Operands(operands[:0])
			for _, operand := range operands {
				if operand != nil && *operand != nil {
					if err := check((*operand).Type(), instruction.Pos()); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}
