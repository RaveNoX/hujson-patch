package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"dario.cat/mergo"
	"github.com/mattbaird/jsonpatch"
	"github.com/spf13/pflag"
	"github.com/tailscale/hujson"
)

const (
	usageText = `
Usage: %s <input> <patch>
Use "-" to read from STDIN (only applicable to one argument)
	`
)

var (
	outputFile string
)

func init() {
	pflag.Usage = func() {
		fmt.Fprintf(os.Stderr, strings.TrimSpace(usageText), filepath.Base(os.Args[0]))
		fmt.Fprintln(os.Stderr, "\nOptions:")
		pflag.PrintDefaults()
	}

	pflag.StringVarP(&outputFile, "output", "o", "", "Write output to this file")
}

func main() {
	pflag.Parse()

	if pflag.NArg() != 2 {
		pflag.Usage()
		os.Exit(2)
	}

	inputFile := pflag.Arg(0)
	patchFile := pflag.Arg(1)

	result, err := patch(inputFile, patchFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if outputFile != "" {
		err = os.WriteFile(outputFile, []byte(result), os.ModePerm)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Cannot write output file: %v\n", err)
			os.Exit(3)
		}
	} else {
		fmt.Println(result)
	}
}

func patch(inputFile, patchFile string) (string, error) {
	var (
		inputBytes, patchBytes []byte
		err                    error
	)

	if inputFile == "-" && patchFile == "-" {
		return "", fmt.Errorf("input and patch are both from STDIN")
	}

	if inputFile == "-" {
		inputBytes, err = io.ReadAll(os.Stdin)
	} else {
		inputBytes, err = os.ReadFile(inputFile)
	}

	if err != nil {
		return "", fmt.Errorf("cannot read input: %w", err)
	}

	if patchFile == "-" {
		patchBytes, err = io.ReadAll(os.Stdin)
	} else {
		patchBytes, err = os.ReadFile(patchFile)
	}

	if err != nil {
		return "", fmt.Errorf("cannot read patch: %w", err)
	}

	inputVal, err := hujson.Parse(inputBytes)
	if err != nil {
		return "", fmt.Errorf("cannot parse input: %w", err)
	}

	patchVal, err := hujson.Parse(patchBytes)
	if err != nil {
		return "", fmt.Errorf("cannot parse patch: %w", err)
	}

	patchVal.Standardize()

	inputOrig := inputVal.Clone()
	inputOrig.Standardize()

	mergedBytes, err := mergeJSON(inputOrig.Pack(), patchVal.Pack())
	if err != nil {
		return "", fmt.Errorf("cannot merge patch: %w", err)
	}

	patchOps, err := jsonpatch.CreatePatch(inputOrig.Pack(), mergedBytes)
	if err != nil {
		return "", fmt.Errorf("cannot construct patch: %w", err)
	}

	patchBytes, err = json.MarshalIndent(patchOps, "", " ")
	if err != nil {
		return "", fmt.Errorf("cannot marshal patch: %w", err)
	}

	err = inputVal.Patch(patchBytes)
	if err != nil {
		return "", fmt.Errorf("cannot apply patch: %w", err)
	}

	inputVal.Format()

	return inputVal.String(), nil
}

func mergeJSON(srcBytes, dstBytes []byte, options ...func(*mergo.Config)) ([]byte, error) {
	var dst, src map[string]interface{}

	if err := json.Unmarshal(dstBytes, &dst); err != nil {
		return nil, fmt.Errorf("failed to unmarshal destination JSON: %w", err)
	}
	if err := json.Unmarshal(srcBytes, &src); err != nil {
		return nil, fmt.Errorf("failed to unmarshal source JSON: %w", err)
	}

	if err := mergo.Merge(&dst, src, options...); err != nil {
		return nil, fmt.Errorf("merge failed: %w", err)
	}

	result, err := json.Marshal(dst)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal merged JSON: %w", err)
	}

	return result, nil
}
