package main

import (
	"bufio"
	"bytes"
	"debug/buildinfo"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/wreulicke/patchr"
	"gopkg.in/yaml.v3"
)

type patchConfig struct {
	valuesPath    string
	commentPrefix string
	inputs        map[string]string
	dryRunOutput  io.Writer
	dryRun        bool
}

func mainInternal() error {
	//nolint:wrapcheck
	return NewApp(map[string]string{}).Execute()
}

func main() {
	if err := mainInternal(); err != nil {
		log.Fatal(err)
	}
}

func NewApp(inputs map[string]string) *cobra.Command {
	var config patchConfig
	config.inputs = inputs

	c := cobra.Command{
		Use:   "patchr [file or directory]",
		Short: "patchr is a tool to apply patches to source code using comment directives",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if config.dryRun {
				config.dryRunOutput = cmd.OutOrStdout()
			}
			return apply(args[0], &config)
		},
	}
	c.Flags().StringVarP(&config.valuesPath, "values", "v", "", "values file for template")
	c.Flags().StringVarP(&config.commentPrefix, "comment-prefix", "p", "", "overrides comment prefix")
	c.Flags().BoolVarP(&config.dryRun, "dry-run", "d", false, "dry run")

	c.AddCommand(
		NewVersionCommand(),
	)
	return &c
}

// need a way to handle any files.
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

var commentPrefixDirective = "patchr:comment-prefix"

func detectCommentPrefix(f *os.File, path string) (string, error) {
	ext := filepath.Ext(path)
	if prefix, ok := wellknownCommentPrefixMap[ext]; ok {
		return prefix, nil
	}

	var err error
	// check shebang
	r := bufio.NewReader(f)
	line, _, err := r.ReadLine()
	if err != nil {
		return "", fmt.Errorf("cannot try to read shebang: %w", err)
	}

	_, err = f.Seek(0, 0)
	if err != nil {
		return "", fmt.Errorf("cannot seek: %w", err)
	}

	// check shebang
	if strings.HasPrefix(string(line), "#!") {
		return "#", nil
	}
	// check comment prefix directive
	index := strings.Index(string(line), commentPrefixDirective)
	if index > 0 {
		return strings.TrimSpace(string(line[:index])), nil
	}
	return "", fmt.Errorf("unsupported file extension: %s", ext)
}

func readValues(config *patchConfig) (any, error) {
	f, err := os.Open(config.valuesPath)
	if err != nil {
		return nil, fmt.Errorf("cannot open file: %w", err)
	}
	defer f.Close()

	p, err := detectCommentPrefix(f, config.valuesPath)
	if err != nil {
		return nil, err
	}

	var b bytes.Buffer
	patcher := patchr.NewPatcher(p, config.inputs)
	err = patcher.Apply(&b, f, nil)
	if err != nil {
		return nil, fmt.Errorf("cannot apply patch: %w", err)
	}

	var data any
	ext := filepath.Ext(config.valuesPath)
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

func apply(targetPath string, config *patchConfig) error {
	var data any
	if config.valuesPath != "" {
		d, err := readValues(config)
		if err != nil {
			return err
		}
		data = d
	}
	s, err := os.Stat(targetPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("file not found: %s", targetPath)
	}
	if s.IsDir() {
		return applyPatchDir(targetPath, config, data)
	}
	return applyPatch(targetPath, config, data)
}

func applyPatchDir(targetPath string, config *patchConfig, data any) error {
	err := filepath.WalkDir(targetPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		// copy config
		return applyPatch(path, config, data)
	})
	if err != nil {
		return fmt.Errorf("cannot walk: %w", err)
	}
	return nil
}

func applyPatch(targetPath string, config *patchConfig, data any) error {
	src, err := os.OpenFile(targetPath, os.O_RDWR, 0o644)
	if err != nil {
		return fmt.Errorf("cannot open file: %w", err)
	}
	defer src.Close()

	prefix := config.commentPrefix
	if prefix == "" {
		var err error
		prefix, err = detectCommentPrefix(src, targetPath)
		if err != nil {
			return err
		}
	}

	var b bytes.Buffer
	p := patchr.NewPatcher(prefix, config.inputs)

	err = p.Apply(&b, src, data)
	if err != nil {
		return fmt.Errorf("cannot apply patch: %w", err)
	}

	if config.dryRun {
		fmt.Fprintf(config.dryRunOutput, "=== %s === start ===\n", targetPath)
		_, err = io.Copy(config.dryRunOutput, &b)
		if err != nil {
			return fmt.Errorf("cannot copy to dry run output: %w", err)
		}
		fmt.Fprintf(config.dryRunOutput, "=== %s === end ===\n", targetPath)
		return nil
	}

	// replace file content
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

			fmt.Fprintf(w, "go version: %s\n", info.GoVersion)
			fmt.Fprintf(w, "path: %s\n", info.Path)
			fmt.Fprintf(w, "mod: %s\n", info.Main.Path)
			fmt.Fprintf(w, "module version: %s\n", info.Main.Version)
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
