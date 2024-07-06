package main

import (
	"bytes"
	"debug/buildinfo"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	patchr "github.com/wreulicke/go-cli-template"
	"gopkg.in/yaml.v3"
)

func mainInternal() error {
	//nolint:wrapcheck
	return NewApp().Execute()
}

func main() {
	if err := mainInternal(); err != nil {
		log.Fatal(err)
	}
}

func NewApp() *cobra.Command {
	var valuesPath string
	var commentPrefix string
	c := cobra.Command{
		Use:   "patchr [file]",
		Short: "patchr is a tool to apply patches to source code using comment directives",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var data any
			if valuesPath != "" {
				d, err := readValues(valuesPath)
				if err != nil {
					return err
				}
				data = d
			}
			f := args[0]
			s, err := os.Stat(f)
			if os.IsNotExist(err) {
				return fmt.Errorf("file not found: %s", f)
			}
			if s.IsDir() {
				return applyPatchDir(f, commentPrefix, data)
			}
			return applyPatch(f, commentPrefix, data)
		},
	}
	c.Flags().StringVarP(&valuesPath, "values", "v", "", "values file for template")
	c.Flags().StringVarP(&commentPrefix, "comment-prefix", "p", "", "overrides comment prefix")

	c.AddCommand(
		NewVersionCommand(),
	)
	return &c
}

// need a way to handle any files
var wellknownCommentPrefixMap = map[string]string{
	".go":     "//",
	".java":   "//",
	".js":     "//",
	".ts":     "//",
	".sql":    "--",
	".sh":     "#",
	".gralde": "//",
	".kt":     "//",
	".groovy": "//",
	".yaml":   "#",
	".yml":    "#",
}

func detectCommentPrefix(path string) (string, error) {
	ext := filepath.Ext(path)
	if prefix, ok := wellknownCommentPrefixMap[ext]; ok {
		return prefix, nil
	}
	return "", fmt.Errorf("unsupported file extension: %s", ext)
}

func readValues(path string) (any, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("cannot open file: %w", err)
	}
	defer f.Close()

	var b bytes.Buffer
	p, err := detectCommentPrefix(path)
	if err != nil {
		return nil, err
	}
	patcher := patchr.NewPatcher(p)
	err = patcher.Apply(&b, f, nil)
	if err != nil {
		return nil, fmt.Errorf("cannot apply patch: %w", err)
	}

	var data any
	ext := filepath.Ext(path)
	switch ext {
	case ".json":
		err = json.NewDecoder(&b).Decode(&data)
		if err != nil {
			return nil, fmt.Errorf("cannot decode json: %w", err)
		}
		return data, nil
	case ".yaml", ".yml":
		err = yaml.NewDecoder(&b).Decode(&data)
		if err != nil {
			return nil, fmt.Errorf("cannot decode yaml: %w", err)
		}
		return data, nil
	default:
		return nil, fmt.Errorf("unsupported file extension: %s", ext)
	}
}

func applyPatchDir(path string, commentPrefix string, data any) error {
	err := filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		return applyPatch(path, commentPrefix, data)
	})
	if err != nil {
		return fmt.Errorf("cannot walk: %w", err)
	}
	return nil
}

func applyPatch(path string, commentPrefix string, data any) error {
	prefix := commentPrefix
	if commentPrefix == "" {
		var err error
		prefix, err = detectCommentPrefix(path)
		if err != nil {
			return err
		}
	}
	src, err := os.OpenFile(path, os.O_RDWR, 0o644)
	if err != nil {
		return fmt.Errorf("cannot open file: %w", err)
	}
	defer src.Close()

	var b bytes.Buffer
	p := patchr.NewPatcher(prefix)

	err = p.Apply(&b, src, data)
	if err != nil {
		return fmt.Errorf("cannot apply patch: %w", err)
	}

	err = src.Truncate(0)
	if err != nil {
		return fmt.Errorf("cannot truncate: %w", err)
	}

	_, err = src.Seek(0, 0)
	if err != nil {
		return fmt.Errorf("cannot seek: %w", err)
	}

	_, err = io.Copy(src, &b)
	if err != nil {
		return fmt.Errorf("cannot copy: %w", err)
	}
	return nil
}

func NewVersionCommand() *cobra.Command {
	var detail bool
	c := &cobra.Command{
		Use:   "version",
		Short: "show version",
		RunE: func(cmd *cobra.Command, args []string) error {
			w := cmd.OutOrStdout()
			info, err := buildinfo.ReadFile(os.Args[0])
			if err != nil {
				return fmt.Errorf("cannot read buildinfo: %w", err)
			}

			fmt.Fprintf(w, "go version: %s", info.GoVersion)
			fmt.Fprintf(w, "module version: %s", info.Main.Version)
			if detail {
				fmt.Fprintln(w)
				fmt.Fprintln(w, info)
			}
			return nil
		},
	}
	c.Flags().BoolVarP(&detail, "detail", "d", false, "show details")
	return c
}
