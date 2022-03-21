package cue

import (
	"fmt"
	"regexp"
	"strings"

	"cuelang.org/go/cue/ast"
	"cuelang.org/go/cue/build"
	"cuelang.org/go/cue/cuecontext"
	CUEErrors "cuelang.org/go/cue/errors"
	"cuelang.org/go/cue/parser"
	"github.com/lipence/config"
)

const (
	Name            = "cue"
	configPathRegex = `.*\.cue$`
)

type extraFileEntry struct {
	path string
	data []byte
}

type Loader struct {
	extraFile []extraFileEntry
}

func (l *Loader) Clear() {}

func (l *Loader) Type() string {
	return Name
}

func (l *Loader) AllowDir() bool {
	return true
}

func (l *Loader) PathPattern() *regexp.Regexp {
	return regexp.MustCompile(configPathRegex)
}

func (l *Loader) Preload(path string, data []byte) {
	l.extraFile = append(l.extraFile, extraFileEntry{path: path, data: data})
}

func (l *Loader) Load(dir string, files map[string][]byte) (config.Value, error) {
	var err error
	var ctx = cuecontext.New()
	var instanceBuilder = build.NewContext().NewInstance(dir, nil)
	// load internal files
	for _, entry := range l.extraFile {
		if err = l.addFile(instanceBuilder, entry.path, entry.data); err != nil {
			return nil, fmt.Errorf("%w (path: %s)", readErrors(err), entry.path)
		}
	}
	// load config files
	for subPath, content := range files {
		if err = l.addFile(instanceBuilder, subPath, content); err != nil {
			return nil, fmt.Errorf("%w (path: %s)", readErrors(err), subPath)
		}
	}
	if err = instanceBuilder.Complete(); err != nil {
		return nil, readErrors(err)
	}
	var cfgInstance = ctx.BuildInstance(instanceBuilder)
	if err = cfgInstance.Err(); err != nil {
		return nil, readErrors(err)
	}
	return NewCueVal(cfgInstance), nil
}

func (l *Loader) addFile(i *build.Instance, path string, src interface{}) (err error) {
	var f *ast.File
	if f, err = parser.ParseFile(path, src, parser.AllowPartial, parser.AllErrors); err != nil {
		return err
	}
	if err = i.AddSyntax(f); err != nil {
		return err
	}
	return nil
}

type multiErrorWrapper []CUEErrors.Error

func (w multiErrorWrapper) Error() string {
	var builder = strings.Builder{}
	builder.WriteString("Multiple cue errors:\n")
	for i := len(w) - 1; i >= 0; i-- {
		e := w[i]
		m, a := e.Msg()
		builder.WriteString(fmt.Sprintf("\x20\x20%d) "+m+"\n", append([]interface{}{i}, a...)...))
		builder.WriteString(fmt.Sprintf("\x20\x20\tPath: %s\n", strings.Join(e.Path(), ".")))
		for j, p := range e.InputPositions() {
			builder.WriteString(fmt.Sprintf("\x20\x20\tPos #%d: %s\n", j, p.String()))
		}
	}
	return strings.TrimSpace(builder.String())
}

func readErrors(err error) error {
	errs := CUEErrors.Errors(err)
	switch len(errs) {
	case 0:
		return nil
	case 1:
		return err
	default:
		return multiErrorWrapper(errs)
	}
}
