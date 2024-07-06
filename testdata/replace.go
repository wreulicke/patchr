package main

import (
	"log"

	"github.com/spf13/cobra"
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
		// patchr:replace Use:   "{{ .Name }}",
		Use:   "patchr",
		Short: "patchr",
		RunE: func(cmd *cobra.Command, args []string) error {
			// nop
			return nil
		},
	}

	return &c
}
