package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/dhohner/wayfinder/internal/recommender"
)

const usage = "Usage: wayfinder [--json] [--explain] [--optimize value|cost|speed|quality] [--against gpt|claude] \"describe the task\""

// Recommender is the behavior required by the CLI. It keeps command parsing
// independent from the recommendation implementation and easy to test.
type Recommender interface {
	RecommendWithOptimizationAgainst(task string, optimization recommender.Optimization, against recommender.AgainstFamily) recommender.Recommendation
}

// Run executes the command and returns a process exit code. It never calls os.Exit.
func Run(args []string, stdout, stderr io.Writer, rec Recommender) int {
	if stdout == nil {
		stdout = io.Discard
	}
	if stderr == nil {
		stderr = io.Discard
	}
	if rec == nil {
		rec = recommender.NewService()
	}

	optimization, against, explain, jsonOutput, task, ok := parseArgs(args)
	if !ok || task == "" {
		fmt.Fprintln(stderr, usage)
		return 2
	}

	recommendation := rec.RecommendWithOptimizationAgainst(task, optimization, against)
	if jsonOutput {
		out, err := recommender.FormatJSON(recommendation, optimization, explain)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		fmt.Fprintln(stdout, out)
		return 0
	}
	if explain {
		fmt.Fprintln(stdout, recommender.FormatWithExplanation(recommendation))
	} else {
		fmt.Fprintln(stdout, recommender.Format(recommendation))
	}
	return 0
}

func parseArgs(args []string) (recommender.Optimization, recommender.AgainstFamily, bool, bool, string, bool) {
	optimization := recommender.OptimizeValue
	against := recommender.AgainstUnspecified
	explain := false
	jsonOutput := false
	var taskParts []string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--explain":
			explain = true
		case strings.HasPrefix(arg, "--explain="):
			return recommender.OptimizeValue, recommender.AgainstUnspecified, false, false, "", false
		case arg == "--json":
			jsonOutput = true
		case strings.HasPrefix(arg, "--json="):
			return recommender.OptimizeValue, recommender.AgainstUnspecified, false, false, "", false
		case arg == "--optimize":
			if i+1 >= len(args) {
				return recommender.OptimizeValue, recommender.AgainstUnspecified, false, false, "", false
			}
			parsed, ok := recommender.ParseOptimization(args[i+1])
			if !ok {
				return recommender.OptimizeValue, recommender.AgainstUnspecified, false, false, "", false
			}
			optimization = parsed
			i++
		case strings.HasPrefix(arg, "--optimize="):
			parsed, ok := recommender.ParseOptimization(strings.TrimPrefix(arg, "--optimize="))
			if !ok {
				return recommender.OptimizeValue, recommender.AgainstUnspecified, false, false, "", false
			}
			optimization = parsed
		case arg == "--against":
			if i+1 >= len(args) {
				return recommender.OptimizeValue, recommender.AgainstUnspecified, false, false, "", false
			}
			parsed, ok := recommender.ParseAgainstFamily(args[i+1])
			if !ok {
				return recommender.OptimizeValue, recommender.AgainstUnspecified, false, false, "", false
			}
			against = parsed
			i++
		case strings.HasPrefix(arg, "--against="):
			parsed, ok := recommender.ParseAgainstFamily(strings.TrimPrefix(arg, "--against="))
			if !ok {
				return recommender.OptimizeValue, recommender.AgainstUnspecified, false, false, "", false
			}
			against = parsed
		case strings.HasPrefix(arg, "--"):
			return recommender.OptimizeValue, recommender.AgainstUnspecified, false, false, "", false
		default:
			taskParts = append(taskParts, arg)
		}
	}

	return optimization, against, explain, jsonOutput, strings.TrimSpace(strings.Join(taskParts, " ")), true
}
