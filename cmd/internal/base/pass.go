package base

import (
	"flag"
	"fmt"
	"strings"
)

type stringValue struct {
	p    *PassArgs
	name string
}

func (p *stringValue) String() string {
	return ""
}

func (p *stringValue) Set(v string) error {
	p.p.Args = append(p.p.Args, fmt.Sprintf("-%v=%v", p.name, v))
	return nil
}

type boolValue struct {
	p    *PassArgs
	name string
}

func (p *boolValue) String() string {
	return ""
}

func (p *boolValue) Set(v string) error {
	p.p.Args = append(p.p.Args, fmt.Sprintf("-%v=%v", p.name, v))
	return nil
}

func (p *boolValue) IsBoolFlag() bool {
	return true
}

type PassArgs struct {
	Args []string
	Flag *flag.FlagSet
}

func (p *PassArgs) Tags() string {
	for _, v := range p.Args {
		if strings.HasPrefix(v, "-tags=") {
			return v[6:]
		}
	}
	return ""
}

func (p *PassArgs) Var(names ...string) {
	for _, name := range names {
		p.Flag.Var(&stringValue{p: p, name: name}, name, "")
	}
}

func (p *PassArgs) Bool(names ...string) {
	for _, name := range names {
		p.Flag.Var(&boolValue{p: p, name: name}, name, "")
	}
}

func NewPassArgs(flag *flag.FlagSet) *PassArgs {
	p := &PassArgs{Flag: flag}
	return p
}

func PassBuildFlags(cmd *Command) *PassArgs {
	p := NewPassArgs(&cmd.Flag)
	cmd.PassArgs = p
	p.Bool("n")
	// Note: "a" flag removed - now handled by flags.AddBuildFlags()
	p.Bool("linkshared", "race", "msan", "asan",
		"trimpath", "work")
	p.Var("p", "asmflags", "compiler",
		"gcflags", "gccgoflags", "installsuffix",
		"ldflags", "pkgdir", "toolexec", "buildvcs")
	return p
}
