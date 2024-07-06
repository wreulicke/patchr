package patchr

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/influxdata/go-prompt"
)

var errNotMetDirective = errors.New("not met directive")

// patchCommentDirective is a comment directive for patching.
type patchCommentDirective string

const (
	// replace next line.
	commentDirectiveReplace patchCommentDirective = "patchr:replace "
	// add next line.
	commentDirectiveAdd = "patchr:add "
	// remove next line.
	commentDirectiveRemove = "patchr:remove"
	// start of template block.
	commentDirectiveTemplateStart = "patchr:template-start"
	// end of template block.
	commentDirectiveTemplateEnd = "patchr:template-end"
	// start of skip block.
	commentDirectiveSkipStart = "patchr:skip-start"
	// end of skip block.
	commentDirectiveSkipEnd = "patchr:skip-end"
)

var commentDirectives = []patchCommentDirective{
	commentDirectiveReplace,
	commentDirectiveAdd,
	commentDirectiveRemove,
	commentDirectiveTemplateStart,
	commentDirectiveTemplateEnd,
	commentDirectiveSkipStart,
	commentDirectiveSkipEnd,
}

type Patcher struct {
	commentPrefix string
	directives    map[patchCommentDirective]string
	inputs        map[string]string
}

func NewPatcher(commentPrefix string, inputs map[string]string) *Patcher {
	p := &Patcher{}
	p.commentPrefix = commentPrefix + " "
	p.directives = make(map[patchCommentDirective]string, len(commentDirectives))
	p.inputs = inputs

	for _, directive := range commentDirectives {
		d := buildDirective(commentPrefix, directive)
		p.directives[directive] = d
	}

	return p
}

func (p *Patcher) Apply(dst io.Writer, src io.Reader, data any) error {
	s := bufio.NewScanner(src)
	d := bufio.NewWriter(dst)
	defer d.Flush()

	for s.Scan() {
		t := s.Text()

		err := p.visit(t, d, s, data)
		if errors.Is(err, errNotMetDirective) {
			err = writeNewLine(t, d)
		}
		if err != nil {
			return err
		}
	}

	if err := s.Err(); err != nil {
		return fmt.Errorf("scanner error: %w", err)
	}

	return nil
}

func (p *Patcher) visit(line string, dst *bufio.Writer, scanner *bufio.Scanner, data any) error {
	for directiveType, directive := range p.directives {
		if strings.Contains(line, directive) {
			switch directiveType {
			case commentDirectiveReplace:
				return p.visitReplace(line, dst, scanner, data)
			case commentDirectiveAdd:
				return p.visitAdd(line, dst, scanner, data)
			case commentDirectiveRemove:
				return p.visitRemove(line, dst, scanner, data)
			case commentDirectiveTemplateStart:
				return p.visitTemplateStart(line, dst, scanner, data)
			case commentDirectiveTemplateEnd:
				return errors.New("unexpected end directive")
			case commentDirectiveSkipStart:
				return p.visitSkipStart(line, dst, scanner, data)
			case commentDirectiveSkipEnd:
				return errors.New("unexpected end directive")
			}
		}
	}

	return errNotMetDirective
}

func (p *Patcher) visitReplace(line string, dst *bufio.Writer, scanner *bufio.Scanner, data any) error {
	// skip next line
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return fmt.Errorf("unexpected scan error: %w", err)
		}
		return errors.New("unexpected EOF")
	}

	d := p.directives[commentDirectiveReplace]
	index := strings.Index(line, d)

	replaced, err := p.executeTemplate(line[index+len(d):], data)
	if err != nil {
		return err
	}

	indent := line[:index]
	return writeNewLine(indent+replaced, dst)
}

func (p *Patcher) visitAdd(line string, dst *bufio.Writer, scanner *bufio.Scanner, data any) error {
	d := p.directives[commentDirectiveAdd]
	index := strings.Index(line, d)

	replaced, err := p.executeTemplate(line[index+len(d):], data)
	if err != nil {
		return err
	}

	indent := line[:index]
	return writeNewLine(indent+replaced, dst)
}

func (p *Patcher) visitRemove(line string, dst *bufio.Writer, scanner *bufio.Scanner, data any) error {
	// skip next line
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return fmt.Errorf("unexpected scan error: %w", err)
		}
		return errors.New("unexpected EOF")
	}

	return nil
}

func (p *Patcher) visitTemplateStart(line string, dst *bufio.Writer, scanner *bufio.Scanner, data any) error {
	index := strings.Index(line, p.directives[commentDirectiveTemplateStart])
	//nolint:gocritic // false positive
	indent := line[:index]
	var b bytes.Buffer
	for scanner.Scan() {
		t := scanner.Text()
		if strings.Contains(t, p.directives[commentDirectiveTemplateEnd]) {
			break
		}
		t = strings.TrimPrefix(t, p.commentPrefix)
		err := writeNewLine(indent+t, &b)
		if err != nil {
			return err
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("unexpected scan error: %w", err)
	}

	t, err := p.executeTemplate(b.String(), data)
	if err != nil {
		return err
	}

	return writeNewLine(t, dst)
}

func (p *Patcher) visitSkipStart(line string, dst *bufio.Writer, scanner *bufio.Scanner, data any) error {
	for scanner.Scan() {
		t := scanner.Text()
		if strings.Contains(t, p.directives[commentDirectiveSkipEnd]) {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("unexpected scan error: %w", err)
	}

	return nil
}

func buildDirective(prefix string, directive patchCommentDirective) string {
	return prefix + " " + string(directive)
}

func writeNewLine(line string, dst io.Writer) error {
	_, err := dst.Write([]byte(line))
	if err != nil {
		return fmt.Errorf("cannot write to dst: %w", err)
	}
	_, err = dst.Write([]byte("\n"))
	if err != nil {
		return fmt.Errorf("cannot write to dst: %w", err)
	}

	return nil
}

func (p *Patcher) executeTemplate(text string, data any) (string, error) {
	var b bytes.Buffer
	t, err := template.New("replace").Funcs(p.funcMap()).Parse(text)
	if err != nil {
		return "", fmt.Errorf("cannot parse template: %w", err)
	}
	err = t.Execute(&b, data)
	if err != nil {
		return "", fmt.Errorf("cannot execute template: %w", err)
	}

	return b.String(), nil
}

func (p *Patcher) input(name string) string {
	if i, ok := p.inputs[name]; ok {
		return i
	}

	i := prompt.Input(name+": ", func(d prompt.Document) []prompt.Suggest {
		return []prompt.Suggest{}
	}, prompt.OptionPrefixTextColor(prompt.Green))
	p.inputs[name] = i

	return i
}

func (p *Patcher) choose(name string, opts ...string) string {
	if i, ok := p.inputs[name]; ok {
		return i
	}

	i := prompt.Choose(name+": ", opts, prompt.OptionPrefixTextColor(prompt.Green))
	p.inputs[name] = i

	return i
}

func (p *Patcher) funcMap() template.FuncMap {
	m := sprig.TxtFuncMap()
	m["i"] = p.input
	m["input"] = p.input
	m["choose"] = p.choose
	m["select"] = p.choose
	return m
}
