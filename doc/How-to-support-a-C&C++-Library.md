How to support a C/C++ Library
=====
## Symbol Visibility
When llgo needs to link C or C++ libraries, symbol visibility is a crucial concept. It determines how C/C++ functions and methods are linked and utilized within llgo. Symbol visibility significantly impacts the handling of symbols in llgo bindings: visible symbols can typically be directly linked, while invisible symbols require wrapper functions.

The accessibility of symbols, particularly destructors, in dynamic libraries depends on their definition method. For instance, destructors that are explicitly declared in header files and implemented as non-inline functions in .cpp files typically appear in the dynamic library's symbol table, making them visible and directly linkable.

**Visible Symbols:** These are symbols that can be found in the dynamic library. They typically represent public API functions and methods.

* Example: A class constructor explicitly declared in the header and implemented in a .cpp file.

* Example: A non-inline destructor declared in the header and implemented in a .cpp file.

**Invisible Symbols:** These are symbols that cannot be found in the dynamic library. This may include inline functions, templates, or certain constructors and destructors.

* Example: A default constructor not explicitly declared.

* Example: An inline destructor or a compiler-generated default destructor.

## Support a C Library

### Install a C Library

We recommend using a package manager (such as brew, apt-get, winget, etc.) to install a C library. For example:

```bash
brew install inih
```

### Writing Go Files to Link Library Functions

1. On macOS, use `nm -gU libbar.dylib` to parse C-style symbols

```bash
0000000000003e55 T _ini_parse
```

2. Find the function prototype you want to convert in the corresponding .h file

```c
int ini_parse(const char* filename, ini_handler handler, void* user);
```

3. Create the corresponding Go file

```bash
inih/
├── _demo
   ├── inih_demo
    ├──inih_demo.go
└── inih.go
```

4. In `inih.go`, use LLGoPackage to specify the location of the third-party library so that llgo can link to the third-party library. Both `pkg-config --libs inih` and `linih` are used to specify the location of the third-party library.

```go
package inih

import (
    _ "unsafe" // unsafe is necessary when using go:linkname
)

const (
    LLGoPackage = "link: $(pkg-config --libs inih); -linih"
)
```

5. Write the corresponding function in `inih.go`

Note that the basic C function type mapping to Go function type can be found at [https://github.com/xgo-dev/llgo/blob/main/doc/Type-Mapping-between-C-and-Go.md](https://github.com/xgo-dev/llgo/blob/main/doc/Type-Mapping-between-C-and-Go.md). Some types requiring special handling are listed at the end of this document for reference.

```go
//go:linkname Parse C.ini_parse
func Parse(filename *c.Char, handler func(user c.Pointer, section *c.Char, name *c.Char, value *c.Char) c.Int, user c.Pointer) c.Int
```

6. Write the function call in `inih_demo.go`

```go
package main

import (
"github.com/goplus/lib/c"
"github.com/goplus/lib/cpp/inih"
)

func main() {
filename := c.Str("path/to/yourIniFile")

if inih.Parse(filename, func(user c.Pointer, section *c.Char, name *c.Char, value *c.Char) c.Int {
        println("section:", c.GoString(section), "name:", c.GoString(name), "value:", c.GoString(value))
        return 1
    }, nil) < 0 {
        println("Error parsing config file")
        return
    }
}
```
   
7. Use llgo to run the demo

```bash
cd inih/_demo/inih_demo
llgo run .
```

### Handling Special Types

#### Handling Enum Values in C

Use const to implement enum values

```go
/*
typedef enum {
    BLEND_ALPHA = 0,                // Blend textures considering alpha (default)
    BLEND_ADDITIVE,                 // Blend textures adding colors
    BLEND_MULTIPLIED,               // Blend textures multiplying colors
    BLEND_ADD_COLORS,               // Blend textures adding colors (alternative)
    BLEND_SUBTRACT_COLORS,          // Blend textures subtracting colors (alternative)
    BLEND_ALPHA_PREMULTIPLY,        // Blend premultiplied textures considering alpha
    BLEND_CUSTOM,                   // Blend textures using custom src/dst factors (use rlSetBlendFactors())
    BLEND_CUSTOM_SEPARATE           // Blend textures using custom rgb/alpha separate src/dst factors (use rlSetBlendFactorsSeparate())
} BlendMode;
*/

type BlendMode c.Int

const (
    BLEND_ALPHA             BlendMode = iota // Blend textures considering alpha (default)
    BLEND_ADDITIVE                           // Blend textures adding colors
    BLEND_MULTIPLIED                         // Blend textures multiplying colors
    BLEND_ADD_COLORS                         // Blend textures adding colors (alternative)
    BLEND_SUBTRACT_COLORS                    // Blend textures subtracting colors (alternative)
    BLEND_ALPHA_PREMULTIPLY                  // Blend premultiplied textures considering alpha
    BLEND_CUSTOM                             // Blend textures using custom src/dst factors (use rlSetBlendFactors())
    BLEND_CUSTOM_SEPARATE                    // Blend textures using custom rgb/alpha separate src/dst factors (use rlSetBlendFactorsSeparate())
)
```

#### Handling Structs in C

```go
// If you need to use class member variables, like llgo/c/raylib
/*
//Vector4, 4 components
typedef struct Vector4 {
    float x;                // Vector x component
    float y;                // Vector y component
    float z;                // Vector z component
    float w;                // Vector w component
} Vector4;
*/

type Vector4 struct {
    X float32 // Vector x component
    Y float32 // Vector y component
    Z float32 // Vector z component
    W float32 // Vector w component
}

// If class member variables don't need to be exposed, like llgo/c/cjson, wrap functions that use these member variables as methods of the class. Example:

/*
typedef struct cJSON
{
    // next/prev allow you to walk array/object chains. Alternatively, use GetArraySize/GetArrayItem/GetObjectItem
    struct cJSON *next;
    struct cJSON *prev;
    // An array or object item will have a child pointer pointing to a chain of the items in the array/object.
    struct cJSON *child;

    // The type of the item, as above.
    int type;

    // The item's string, if type==cJSON_String  and type == cJSON_Raw
    char *valuestring;
    /* writing to valueint is DEPRECATED, use cJSON_SetNumberValue instead
    int valueint;
    // The item's number, if type==cJSON_Number
    double valuedouble;

    // The item's name string, if this item is the child of, or is in the list of subitems of an object.
    char *string;
} cJSON;
*/

// llgo:type C
type JSON struct {
    Unused [0]byte
}
// llgo:link (*JSON).AddItem C.cJSON_AddItemToArray
func (o *JSON) AddItem(item *JSON) c.Int { return 0 }
```

For the size of Unused, if the methods bound to the structure do not need to create objects, i.e., the receiver of the Go methods bound to this structure is of pointer type, you can declare `Unused [0]byte`. Otherwise, you need to write a simple C file using the `sizeof` operator to calculate the size of the structure. Assuming the structure size is 38 bytes, then declare `Unused [38]byte`.

#### Handling Function Pointers in C

```go
// Convert function pointers to Go style and then declare function pointer types using aliases

//llgo:type C
type Comp func(a c.Int)

```

#### Handling char ** Type in C

Handle char ** as `[]*c.Char`

```go
// void printStrings(const char * strings[], int size);
//
//llgo:link PrintStrings C.printStrings
func PrintStrings(strings **c.Char, size c.Int) {}

```

```go
package main

import (
	"unsafe"

	"github.com/goplus/lib/c"
	"github.com/goplus/lib/c/ptrtest"
)

func main() {
    strings := make([]*c.Char, 4)
    strings[0] = c.Str("hello")
    strings[1] = c.Str("world")
    strings[2] = c.Str("ni")
    strings[3] = c.Str("hao")
    ptrtest.PrintStrings(unsafe.SliceData(strings), c.Int(4))
}
```

## LLGO for C++ Third-Party Libraries

Using the C++ part of the inih library as an example

### Installation

Same as installing C libraries

### File Structure

After migrating the C part of the inih library, just continue creating files in the same directory.

```
inih/
├── _demo
    ├── inih_demo
	    ├──inih_demo.go
	  ├── INIReader_demo
		  ├──reader_demo.go
├── _wrap/cpp_wrap.cpp (optional)
└── inih.go
└── reader.go
```

### Writing Go Files to Link Library Functions

#### Migrating Ordinary Functions

Since the inih library does not have C++ style ordinary functions, we'll use an ordinary method of a class as an example. The specific process is the same.

Ordinary functions can be directly linked using the corresponding symbol in the dylib. Translate C++ symbols to readable form in the dylib directory.

```bash
nm -gU $(brew --prefix inih)/lib/libINIReader.dylib > output.txt
c++filt <output.txt> symbol.txt
```

Function prototype

```cpp
int ParseError() const;
```

Example of `symbol.txt`

```bash
0000000000002992 T INIReader::ParseError() const
```

Example of `output.txt`

```bash
0000000000002992 T __ZNK9INIReader10ParseErrorEv
```

Find the offset of the function you want to use in `symbol.txt`, then go back to `output.txt` and find the symbol corresponding to that offset.

For functions, generally use `go:linkname` to link. Here, refer to the migration method of C library functions, but bind the symbol to the C++ style symbol. Note that since C. represents a `_`, `__ZNK9INIReader10ParseErrorEv` should be written as `C._ZNK9INIReader10ParseErrorEv`.

```go
// The inih library currently does not involve ordinary functions, this is for demonstration purposes only and is not needed for migrating inih
//go:linkname ParseError C.__ZNK9INIReader10ParseErrorEv
func ParseError() c.Int
```

#### Migrating Classes

- Use a struct to map the class. The writing method is the same as migrating a struct in the C library migration:

  ```go
  // llgo:type C
  type Reader struct {
      Unused [32]byte
  }
  ```
- Class Methods

  For general methods of the class, directly use `llgo:link` to link:

  ```go
  // llgo:link (*Reader).GetInteger C._ZNK9INIReader10GetIntegerERKNSt3__112basic_stringIcNS0_11char_traitsIcEENS0_9allocatorIcEEEES8_l
  func (r *Reader) GetInteger(section *std.String, name *std.String, defaultValue c.Long) c.Long {
  	return 0
  }
  ```
- Constructor

  - Explicitly Constructor:

    ```cpp
    class INIReader {
    public:
        // Construct INIReader and parse given filename.
        INI_API explicit INIReader(const std::string &filename);
    }
    ```
    Bind to the `InitFromFile` method of the struct and call it in the `NewReaderFile` function to initialize the class and return the class for Go to use.

    The following long string starting with `_ZN9INI` is the corresponding function prototype in the symbol table for `INIReader(const std::string &filename)`
    ```go
    // llgo:link (*Reader).InitFromFile C._ZN9INIReaderC1ERKNSt3__112basic_stringIcNS0_11char_traitsIcEENS0_9allocatorIcEEEE
    func (r *Reader) InitFromFile(fileName *std.String) {}

    // NewReaderFile creates a new INIReader instance.
    func NewReaderFile(fileName *std.String) (ret Reader) {
        ret.InitFromFile(fileName)
        return
    }
    ```
  - Implicitly Constructor 
  
    In typical implementations of the inih library, directly invoking implicit constructors to instantiate reader objects is not recommended. For detailed examples of how bindings effectively handle non-exported symbols, please refer to the [Templates and Inlines](#templates-and-inlines) section.

- Destructor

    Explicitly declared and non-inline destructors can be directly linked, consistent with the linking method of general class methods (see "Class Methods" section). For destructors that do not appear in the dynamic library's symbol table, a wrapper layer implementation is required. This wrapper in the C++ wrapper file (e.g., cppWrap.cpp) looks like:

    ```cpp
    extern "C" {
        void INIReaderDispose(INIReader* r) {
            r->~INIReader();
        }
    } // extern "C"
    ```
    This wrapper function explicitly calls the object's destructor. By using extern "C", we ensure that this function can be called by C code, allowing Go to link to it.
    In the Go file:
    ```go
    // llgo:link (*Reader).Dispose C.INIReaderDispose
    func (r *Reader) Dispose() {}
    ```
    Here we link the Go Dispose method to the C++ wrapped INIReaderDispose function.
    In actual usage:
    We use defer to ensure that the Dispose method is called when the reader object goes out of scope, thus properly releasing resources.
    ```go
    reader := inih.NewReader(c.Str(buf), uintptr(len(buf)))
    defer reader.Dispose()
    ```
    This situation is analogous to the handling of inline functions and templates described in the following section.

#### Templates and Inlines

Templates or inlines do not generate symbols in dynamic libraries (dylib) (default constructors and destructors). To ensure that you can use C style symbols to link template or inline functions, create a C++ file and wrap it with `extern "C"`, then bind the functions directly in Go.

```cpp
// Using std::string as an example, not needed for migrating inih
extern "C" void stdStringInitFromCStrLen(std::string* s, const char* cstr, size_t len) {
	new(s) std::string(cstr, len);
}
```

Then use LLGoFiles to link in Go: the writing of standard library's `LLGoFiles` and `LLGoPackage` is slightly different from third-party libraries. Using `std::string` and `spdlog` library as examples, inih does not involve this step:

```go
//  std::string
const (
    LLGoFiles   = "_wrap/string.cpp"
    LLGoPackage = "link: c++"
  )
// llgo:link (*String).InitFromCStrLen C.stdStringInitFromCStrLen
func (s *String) InitFromCStrLen(cstr *c.Char, n uintptr) {}
//  If it is a third-party library, specify pkg-config, using spdlog as an example:
const (
LLGoFiles   = "$(pkg-config --cflags spdlog): cppWrap/cppWrap.cpp"
LLGoPackage = "link: $(pkg-config --libs spdlog); -lspdlog -pthread -lfmt")
```

### Writing and Running the Demo

```go
package main

import (
	"github.com/goplus/lib/c"
	"github.com/goplus/lib/cpp/inih"
	"github.com/goplus/lib/cpp/std"
)

func demoFromBuffer() {
	buf := `[settings]
                username=admin
                timeout=100`   
	reader := inih.NewReader(c.Str(buf), uintptr(len(buf)))
	defer reader.Dispose()

	println(reader.ParseError())
	sec := std.Str("settings")
	name := std.Str("timeout")
	value := reader.GetInteger(sec, name, 0)
	println("value:", value)
}

func demoFromFile() {
	reader := inih.NewReaderFile(std.Str("config.ini"))
	defer reader.Dispose()

	if ret := reader.ParseError(); ret != 0 {
		println("Error parsing config file:", ret)
		return
	}
	isDatabaseEnabled := reader.GetBoolean(std.Str("database"), std.Str("enabled"), false)
	port := reader.GetInteger(std.NewString("database"), std.Str("port"), 0)
	s := reader.GetString(std.Str("database"), std.Str("server"), std.Str("unknown"))
	println("s:", s.Str())
	println("isDatabaseEnabled:", isDatabaseEnabled, "port:", port)
}

func main() {
	demoFromBuffer()
	demoFromFile()
}
```

Use `llgo run .` to run in the directory where the demo is written.
