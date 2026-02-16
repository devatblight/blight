package commands

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"math"
	"strconv"
	"strings"
)

type CalcResult struct {
	Expression string
	Result     string
	Valid      bool
}

func Evaluate(input string) CalcResult {
	expr := strings.TrimSpace(input)
	if strings.HasPrefix(expr, "=") {
		expr = strings.TrimSpace(expr[1:])
	}

	if expr == "" {
		return CalcResult{Valid: false}
	}

	result, err := evalExpr(expr)
	if err != nil {
		return CalcResult{Valid: false}
	}

	formatted := formatNumber(result)
	return CalcResult{
		Expression: expr,
		Result:     formatted,
		Valid:      true,
	}
}

func IsCalcQuery(query string) bool {
	q := strings.TrimSpace(query)
	if strings.HasPrefix(q, "=") {
		return true
	}
	if len(q) < 2 {
		return false
	}
	hasDigit := false
	hasOp := false
	for _, c := range q {
		if c >= '0' && c <= '9' {
			hasDigit = true
		}
		if c == '+' || c == '-' || c == '*' || c == '/' || c == '%' || c == '^' {
			hasOp = true
		}
	}
	return hasDigit && hasOp
}

func evalExpr(expr string) (float64, error) {
	expr = strings.ReplaceAll(expr, "^", "**")
	expr = strings.ReplaceAll(expr, "**", "^")

	node, err := parser.ParseExpr(expr)
	if err != nil {
		return 0, err
	}
	return evalNode(node)
}

func evalNode(node ast.Expr) (float64, error) {
	switch n := node.(type) {
	case *ast.BasicLit:
		return strconv.ParseFloat(n.Value, 64)

	case *ast.ParenExpr:
		return evalNode(n.X)

	case *ast.UnaryExpr:
		x, err := evalNode(n.X)
		if err != nil {
			return 0, err
		}
		switch n.Op {
		case token.SUB:
			return -x, nil
		case token.ADD:
			return x, nil
		}
		return 0, fmt.Errorf("unsupported unary op: %s", n.Op)

	case *ast.BinaryExpr:
		left, err := evalNode(n.X)
		if err != nil {
			return 0, err
		}
		right, err := evalNode(n.Y)
		if err != nil {
			return 0, err
		}

		switch n.Op {
		case token.ADD:
			return left + right, nil
		case token.SUB:
			return left - right, nil
		case token.MUL:
			return left * right, nil
		case token.QUO:
			if right == 0 {
				return 0, fmt.Errorf("division by zero")
			}
			return left / right, nil
		case token.REM:
			if right == 0 {
				return 0, fmt.Errorf("modulo by zero")
			}
			return float64(int64(left) % int64(right)), nil
		case token.XOR:
			return math.Pow(left, right), nil
		}
		return 0, fmt.Errorf("unsupported op: %s", n.Op)

	case *ast.Ident:
		switch strings.ToLower(n.Name) {
		case "pi":
			return math.Pi, nil
		case "e":
			return math.E, nil
		}
		return 0, fmt.Errorf("unknown identifier: %s", n.Name)
	}

	return 0, fmt.Errorf("unsupported expression")
}

func formatNumber(f float64) string {
	if f == float64(int64(f)) && !math.IsInf(f, 0) {
		return fmt.Sprintf("%d", int64(f))
	}
	s := fmt.Sprintf("%.10f", f)
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	return s
}
