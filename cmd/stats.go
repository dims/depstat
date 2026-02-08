/*
Copyright 2021 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var dir string
var jsonOutput bool
var csvOutput bool
var verbose bool
var mainModules []string
var splitTestOnly bool
var excludeModules []string
var statsCompare bool
var compareSetA string
var compareSetB string
var compareMainModulesA []string
var compareMainModulesB []string

type Chain []string

// statsCmd represents the statsDeps command
var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Shows metrics about dependency chains",
	Long: `Provides the following metrics:
	1. Direct Dependencies: Total number of dependencies required by the mainModule(s) directly
	2. Transitive Dependencies: Total number of transitive dependencies (dependencies which are further needed by direct dependencies of the project)
	3. Total Dependencies: Total number of dependencies of the mainModule(s)
	4. Max Depth of Dependencies: Length of the longest chain starting from the first mainModule; defaults to length from the first module encountered in "go mod graph" output`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) != 0 {
			return fmt.Errorf("stats does not take any arguments")
		}
		if statsCompare {
			return runStatsCompare(cmd)
		}
		result, err := computeStatsSnapshot(mainModules, excludeModules, splitTestOnly)
		if err != nil {
			return err
		}
		return renderStatsSnapshot(result, mainModules, excludeModules)
	},
}

type StatsSnapshot struct {
	DirectDeps    int      `json:"directDependencies"`
	TransDeps     int      `json:"transitiveDependencies"`
	TotalDeps     int      `json:"totalDependencies"`
	MaxDepth      int      `json:"maxDepthOfDependencies"`
	TestOnlyDeps  *int     `json:"testOnlyDependencies,omitempty"`
	NonTestOnly   *int     `json:"nonTestOnlyDependencies,omitempty"`
	MainModules   []string `json:"mainModules,omitempty"`
	ExcludeValues []string `json:"excludeModules,omitempty"`
}

type StatsCompareResult struct {
	SetA    string        `json:"setA"`
	SetB    string        `json:"setB"`
	Before  StatsSnapshot `json:"before"`
	After   StatsSnapshot `json:"after"`
	Delta   StatsSnapshot `json:"delta"`
	OnlyInB []string      `json:"onlyInB"`
}

func computeStatsSnapshot(mods []string, excludes []string, includeSplit bool) (*StatsSnapshot, error) {
	excludeModules = excludes
	defer func() {
		excludeModules = nil
	}()
	depGraph := getDepInfo(mods)
	if len(depGraph.MainModules) == 0 {
		return nil, fmt.Errorf("no main modules remain after exclusions; adjust --exclude-modules or --mainModules")
	}
	var longestChain Chain
	if len(depGraph.MainModules) > 0 {
		var temp Chain
		longestChain = getLongestChain(depGraph.MainModules[0], depGraph.Graph, temp, map[string]Chain{})
	}
	maxDepth := len(longestChain)
	directDeps := len(depGraph.DirectDepList)
	transitiveDeps := len(depGraph.TransDepList)
	allDeps := getAllDeps(depGraph.DirectDepList, depGraph.TransDepList)
	totalDeps := len(allDeps)

	result := &StatsSnapshot{
		DirectDeps:    directDeps,
		TransDeps:     transitiveDeps,
		TotalDeps:     totalDeps,
		MaxDepth:      maxDepth,
		MainModules:   depGraph.MainModules,
		ExcludeValues: excludes,
	}

	if includeSplit {
		testOnlySet, err := classifyTestDeps(allDeps)
		if err != nil {
			return nil, fmt.Errorf("failed to classify dependencies as test-only/non-test: %w", err)
		}
		testOnlyDeps := len(filterDepsByTestStatus(allDeps, testOnlySet, true))
		nonTestOnlyDeps := len(filterDepsByTestStatus(allDeps, testOnlySet, false))
		result.TestOnlyDeps = &testOnlyDeps
		result.NonTestOnly = &nonTestOnlyDeps
	}

	return result, nil
}

func renderStatsSnapshot(result *StatsSnapshot, mods []string, excludes []string) error {
	if !jsonOutput && !csvOutput {
		fmt.Printf("Direct Dependencies: %d \n", result.DirectDeps)
		fmt.Printf("Transitive Dependencies: %d \n", result.TransDeps)
		fmt.Printf("Total Dependencies: %d \n", result.TotalDeps)
		fmt.Printf("Max Depth Of Dependencies: %d \n", result.MaxDepth)
		if result.TestOnlyDeps != nil && result.NonTestOnly != nil {
			fmt.Printf("Test-only Dependencies: %d \n", *result.TestOnlyDeps)
			fmt.Printf("Non-test Dependencies: %d \n", *result.NonTestOnly)
		}
	}
	if verbose {
		fmt.Println("All dependencies:")
		excludeModules = excludes
		defer func() {
			excludeModules = nil
		}()
		depGraph := getDepInfo(mods)
		printDeps(getAllDeps(depGraph.DirectDepList, depGraph.TransDepList))
	}
	if jsonOutput {
		outputObj := struct {
			DirectDeps   int  `json:"directDependencies"`
			TransDeps    int  `json:"transitiveDependencies"`
			TotalDeps    int  `json:"totalDependencies"`
			MaxDepth     int  `json:"maxDepthOfDependencies"`
			TestOnlyDeps *int `json:"testOnlyDependencies,omitempty"`
			NonTestOnly  *int `json:"nonTestOnlyDependencies,omitempty"`
		}{
			DirectDeps:   result.DirectDeps,
			TransDeps:    result.TransDeps,
			TotalDeps:    result.TotalDeps,
			MaxDepth:     result.MaxDepth,
			TestOnlyDeps: result.TestOnlyDeps,
			NonTestOnly:  result.NonTestOnly,
		}
		outputRaw, err := json.MarshalIndent(outputObj, "", "\t")
		if err != nil {
			return err
		}
		fmt.Print(string(outputRaw))
	}
	if csvOutput {
		if result.TestOnlyDeps != nil && result.NonTestOnly != nil {
			fmt.Println("Direct,Transitive,Total,MaxDepth,TestOnly,NonTestOnly")
			fmt.Printf("%d,%d,%d,%d,%d,%d\n", result.DirectDeps, result.TransDeps, result.TotalDeps, result.MaxDepth, *result.TestOnlyDeps, *result.NonTestOnly)
		} else {
			fmt.Println("Direct,Transitive,Total,MaxDepth")
			fmt.Printf("%d,%d,%d,%d\n", result.DirectDeps, result.TransDeps, result.TotalDeps, result.MaxDepth)
		}
	}
	return nil
}

func runStatsCompare(cmd *cobra.Command) error {
	if splitTestOnly {
		return fmt.Errorf("--compare cannot be combined with --split-test-only")
	}
	modsA := mainModules
	modsB := mainModules
	if len(compareMainModulesA) > 0 {
		modsA = compareMainModulesA
	}
	if len(compareMainModulesB) > 0 {
		modsB = compareMainModulesB
	}
	setA := compareSetA
	if setA == "" {
		setA = "A"
	}
	setB := compareSetB
	if setB == "" {
		setB = "B"
	}
	before, err := computeStatsSnapshot(modsA, excludeModules, false)
	if err != nil {
		return err
	}
	after, err := computeStatsSnapshot(modsB, excludeModules, false)
	if err != nil {
		return err
	}
	result := StatsCompareResult{
		SetA:   setA,
		SetB:   setB,
		Before: *before,
		After:  *after,
		Delta: StatsSnapshot{
			DirectDeps: after.DirectDeps - before.DirectDeps,
			TransDeps:  after.TransDeps - before.TransDeps,
			TotalDeps:  after.TotalDeps - before.TotalDeps,
			MaxDepth:   after.MaxDepth - before.MaxDepth,
		},
		OnlyInB: diffSlices(getAllDeps(before.MainModules, nil), getAllDeps(after.MainModules, nil)),
	}

	if jsonOutput {
		out, err := json.MarshalIndent(result, "", "\t")
		if err != nil {
			return err
		}
		fmt.Print(string(out))
		return nil
	}
	if csvOutput {
		fmt.Printf("Set,Direct,Transitive,Total,MaxDepth\n")
		fmt.Printf("%s,%d,%d,%d,%d\n", setA, before.DirectDeps, before.TransDeps, before.TotalDeps, before.MaxDepth)
		fmt.Printf("%s,%d,%d,%d,%d\n", setB, after.DirectDeps, after.TransDeps, after.TotalDeps, after.MaxDepth)
		fmt.Printf("Delta,%d,%d,%d,%d\n", result.Delta.DirectDeps, result.Delta.TransDeps, result.Delta.TotalDeps, result.Delta.MaxDepth)
		if len(result.OnlyInB) > 0 {
			fmt.Printf("OnlyIn%s,%s\n", setB, strings.Join(result.OnlyInB, ";"))
		}
		return nil
	}
	fmt.Printf("Stats compare (%s -> %s)\n", setA, setB)
	fmt.Printf("Direct Dependencies: %d -> %d (delta %+d)\n", before.DirectDeps, after.DirectDeps, result.Delta.DirectDeps)
	fmt.Printf("Transitive Dependencies: %d -> %d (delta %+d)\n", before.TransDeps, after.TransDeps, result.Delta.TransDeps)
	fmt.Printf("Total Dependencies: %d -> %d (delta %+d)\n", before.TotalDeps, after.TotalDeps, result.Delta.TotalDeps)
	fmt.Printf("Max Depth Of Dependencies: %d -> %d (delta %+d)\n", before.MaxDepth, after.MaxDepth, result.Delta.MaxDepth)
	if len(result.OnlyInB) > 0 {
		fmt.Printf("Only in %s: %s\n", setB, strings.Join(result.OnlyInB, ", "))
	}
	return nil
}

// get the longest chain starting from currentDep
func getLongestChain(currentDep string, graph map[string][]string, currentChain Chain, longestChains map[string]Chain) Chain {
	// fmt.Println(strings.Repeat("  ", len(currentChain)), currentDep)

	// already computed
	if longestChain, ok := longestChains[currentDep]; ok {
		return longestChain
	}

	deps := graph[currentDep]

	if len(deps) == 0 {
		// we have no dependencies, our longest chain is just us
		longestChains[currentDep] = Chain{currentDep}
		return longestChains[currentDep]
	}

	if contains(currentChain, currentDep) {
		// we've already been visited in the current chain, avoid cycles but also don't record a longest chain for currentDep
		return nil
	}

	currentChain = append(currentChain, currentDep)
	// find the longest dependency chain
	var longestDepChain Chain
	for _, dep := range deps {
		depChain := getLongestChain(dep, graph, currentChain, longestChains)
		if len(depChain) > len(longestDepChain) {
			longestDepChain = depChain
		}
	}
	// prepend ourselves to the longest of our dependencies' chains and persist
	longestChains[currentDep] = append(Chain{currentDep}, longestDepChain...)
	return longestChains[currentDep]
}

func init() {
	rootCmd.AddCommand(statsCmd)
	statsCmd.Flags().StringVarP(&dir, "dir", "d", "", "Directory containing the module to evaluate. Defaults to the current directory.")
	statsCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Get additional details")
	statsCmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "Get the output in JSON format")
	statsCmd.Flags().BoolVarP(&csvOutput, "csv", "c", false, "Get the output in CSV format")
	statsCmd.Flags().BoolVar(&splitTestOnly, "split-test-only", false, "Split dependency totals into test-only and non-test sections using `go mod why -m`")
	statsCmd.Flags().BoolVar(&statsCompare, "compare", false, "Compare stats between two module sets")
	statsCmd.Flags().StringVar(&compareSetA, "set-a", "", "Label for the first comparison set")
	statsCmd.Flags().StringVar(&compareSetB, "set-b", "", "Label for the second comparison set")
	statsCmd.Flags().StringSliceVar(&compareMainModulesA, "main-modules-a", []string{}, "Main modules for comparison set A")
	statsCmd.Flags().StringSliceVar(&compareMainModulesB, "main-modules-b", []string{}, "Main modules for comparison set B")
	statsCmd.Flags().StringSliceVar(&excludeModules, "exclude-modules", []string{}, "Exclude module path patterns (repeatable, supports * wildcard)")
	statsCmd.Flags().StringSliceVarP(&mainModules, "mainModules", "m", []string{}, "Enter modules whose dependencies should be considered direct dependencies; defaults to the first module encountered in `go mod graph` output")
}
