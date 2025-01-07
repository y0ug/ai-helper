package io

import (
	"fmt"
	"os"
)

// WriteOutput writes the output to either stdout or a file
func WriteOutput(output string, outputFile string) error {
	// If no output file specified, write to stdout
	if outputFile == "" {
		fmt.Println(output)
		return nil
	}

	// Write to file
	err := os.WriteFile(outputFile, []byte(output), 0644)
	if err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	return nil
}
