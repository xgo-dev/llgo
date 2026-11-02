package main

import (
	"fmt"
	"os"
	"strings"
	"unsafe"

	"github.com/goplus/lib/c"
	"github.com/llarhub/clang-c"
)

type Data struct {
	Depth c.Uint
	Unit  clang.TranslationUnit
}

var accessMap = map[clang.CXXAccessSpecifier]string{
	clang.CXXInvalidAccessSpecifier: "invalid",
	clang.CXXPublic:                 "public",
	clang.CXXProtected:              "protected",
	clang.CXXPrivate:                "private",
}

func printIndent(depth c.Uint) {
	fmt.Print(strings.Repeat("   ", int(depth)))
}

func accessToString(spec clang.CXXAccessSpecifier) string {
	if str, ok := accessMap[spec]; ok {
		return str
	}
	return "unknown"
}

func visit(cursor, parent clang.Cursor, ClientData c.Pointer) clang.ChildVisitResult {
	data := (*Data)(ClientData)
	printAST(cursor, data)
	return clang.ChildVisit_Continue
}

func printType(t clang.Type, data *Data) {
	printIndent(data.Depth)

	typeSpell := t.Spelling()
	typeKind := t.Kind.Spelling()

	if t.Kind == clang.Type_Invalid {
	} else if t.Kind == clang.Type_Unexposed {
		c.Printf(c.Str("<UnexposedType|%s>: %s\n"), typeKind.CStr(), typeSpell.CStr())
	} else if t.Kind >= clang.Type_FirstBuiltin && t.Kind <= clang.Type_LastBuiltin {
		c.Printf(c.Str("<BuiltinType|%s>: %s\n"), typeKind.CStr(), typeSpell.CStr())
	} else if t.Kind > clang.Type_Complex {
		c.Printf(c.Str("<ComplexType|%s>: %s\n"), typeKind.CStr(), typeSpell.CStr())
	}

	data.Depth++
	switch t.Kind {
	case clang.Type_Pointer:
		printType(t.Pointee(), data)
	case clang.Type_IncompleteArray, clang.Type_VariableArray, clang.Type_DependentSizedArray, clang.Type_ConstantArray:
		printType(t.ArrayElement(), data)
	case clang.Type_Typedef:
		printType(t.Declaration().TypedefDeclUnderlyingType(), data)
	case clang.Type_Elaborated:
		printType(t.Named(), data)
	case clang.Type_FunctionProto:
		printType(t.Result(), data)
		for i := 0; i < int(t.NumArgTypes()); i++ {
			printType(t.Arg(c.Uint(i)), data)
		}
	}
	data.Depth--

	typeKind.Dispose()
	typeSpell.Dispose()
}

func printLocation(cursor clang.Cursor) {
	loc := cursor.Location()
	var file clang.File
	var line, column c.Uint

	loc.Spelling(&file, &line, &column, nil)
	filename := file.Name()
	defer filename.Dispose()

	c.Printf(c.Str("(Loc:%s:%d:%d)\n"), filename.CStr(), line, column)
}

func printAccess(cursor clang.Cursor) {
	kind := cursor.Kind.Spelling()
	spell := cursor.Spelling()
	defer kind.Dispose()
	defer spell.Dispose()

	c.Printf(c.Str("%s: %s %s"), kind.CStr(), spell.CStr(), c.AllocaCStr(accessToString(cursor.CXXAccessSpecifier())))
	printLocation(cursor)
}

func printMacro(cursor clang.Cursor, unit clang.TranslationUnit) {
	kind := cursor.Kind.Spelling()
	defer kind.Dispose()

	c.Printf(c.Str("%s: "), kind.CStr())
	ran := cursor.Extent()
	var numTokens c.Uint
	var tokens *clang.Token
	unit.Tokenize(ran, &tokens, &numTokens)
	defer unit.DisposeTokens(tokens, numTokens)

	tokensSlice := unsafe.Slice(tokens, int(numTokens))
	for _, tok := range tokensSlice {
		tokStr := unit.TokenSpelling(tok)
		c.Printf(c.Str("%s "), tokStr.CStr())
		tokStr.Dispose()
	}
	printLocation(cursor)
}

func printFunc(cursor clang.Cursor, data *Data) {
	kind := cursor.Kind.Spelling()
	spell := cursor.Spelling()
	symbol := cursor.Mangling()
	defer symbol.Dispose()
	defer kind.Dispose()
	defer spell.Dispose()

	c.Printf(c.Str("%s: %s (Symbol: %s)"), kind.CStr(), spell.CStr(), symbol.CStr())
	printLocation(cursor)
	printType(cursor.Type(), data)
}

func printEnumConstant(cursor clang.Cursor) {
	kind := cursor.Kind.Spelling()
	spell := cursor.Spelling()
	defer kind.Dispose()
	defer spell.Dispose()

	c.Printf(c.Str("%s: %s:%lld"), kind.CStr(), spell.CStr(), cursor.EnumConstantDeclValue())
	printLocation(cursor)
}

func printDefault(cursor clang.Cursor, data *Data) {
	kind := cursor.Kind.Spelling()
	spell := cursor.Spelling()
	defer kind.Dispose()
	defer spell.Dispose()

	// node which has type
	if cursor.Type().Kind != clang.Type_Invalid {
		c.Printf(c.Str("%s: %s"), kind.CStr(), spell.CStr())
		printLocation(cursor)
		printType(cursor.Type(), data)
	} else {
		c.Printf(c.Str("%s: %s\n"), kind.CStr(), spell.CStr())
	}
}

func printAST(cursor clang.Cursor, data *Data) {
	kind := cursor.Kind.Spelling()
	spell := cursor.Spelling()

	printIndent(data.Depth)

	switch cursor.Kind {
	case clang.Cursor_CXXAccessSpecifier:
		printAccess(cursor)
	case clang.Cursor_MacroDefinition:
		printMacro(cursor, data.Unit)
	case clang.Cursor_FunctionDecl, clang.Cursor_CXXMethod, clang.Cursor_Constructor, clang.Cursor_Destructor:
		printFunc(cursor, data)
	case clang.Cursor_EnumConstantDecl:
		printEnumConstant(cursor)
	default:
		printDefault(cursor, data)
	}

	data.Depth++
	clang.VisitChildren(cursor, visit, c.Pointer(data))
	data.Depth--

	kind.Dispose()
	spell.Dispose()
}

func main() {
	if c.Argc != 2 {
		fmt.Fprintln(os.Stderr, "Usage: castdump <headerFile>")
		return
	}

	args := make([]*c.Char, 3)
	args[0] = c.Str("-x")
	args[1] = c.Str("c++")
	args[2] = c.Str("-std=c++11")

	sourceFile := *c.Advance(c.Argv, 1)
	index := clang.CreateIndex(0, 0)
	unit := index.ParseTranslationUnit(
		sourceFile,
		unsafe.SliceData(args), 3,
		nil, 0,
		c.Uint(clang.TranslationUnit_DetailedPreprocessingRecord),
	)
	defer index.Dispose()
	defer unit.Dispose()

	if unit == nil {
		println("Unable to parse translation unit. Quitting.")
		c.Exit(1)
	}

	cursor := unit.Cursor()

	Data := &Data{
		Depth: 0,
		Unit:  unit,
	}
	printAST(cursor, Data)
}
