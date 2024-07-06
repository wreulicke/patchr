package main

import (
	"bytes"
	"debug/buildinfo"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	patchr "github.com/wreulicke/go-cli-template"
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
	c := cobra.Command{
		Use:   "patchr [file]",
		Short: "patchr is a tool to apply patches to source code using comment directives",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			f := args[0]
			s, err := os.Stat(f)
			if os.IsNotExist(err) {
				return fmt.Errorf("file not found: %s", f)
			}
			if s.IsDir() {
				// todo support dir
				return fmt.Errorf("file is a directory: %s", f)
			}
			return applyPatch(f, nil) // TODO add data
		},
	}

	c.AddCommand(
		NewVersionCommand(),
	)
	return &c
}

func detectCommentPrefix(path string) (string, error) {
	ext := filepath.Ext(path)
	if ext == ".go" {
		return "//", nil
	}
	return "", fmt.Errorf("unsupported file extension: %s", ext)
}

func applyPatch(path string, data any) error {
	prefix, err := detectCommentPrefix(path)
	if err != nil {
		return err
	}
	src, err := os.OpenFile(path, os.O_RDWR, 0644)
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
	return err
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
