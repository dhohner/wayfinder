package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/dhohner/wayfinder/internal/recommender"
)

const usage = "Usage: wayfinder [--optimize value|cost|speed|quality] \"describe the task\""

// Recommender is the behavior required by the CLI. It keeps command parsing
// independent from the recommendation implementation and easy to test.
type Recommender interface {
	RecommendWithOptimization(task string, optimization recommender.Optimization) recommender.Recommendation
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

	optimization, task, ok := parseArgs(args)
	if !ok || task == "" {
		fmt.Fprintln(stderr, usage)
		return 2
	}

	fmt.Fprintln(stdout, recommender.Format(rec.RecommendWithOptimization(task, optimization)))
	return 0
}

func parseArgs(args []string) (recommender.Optimization, string, bool) {
	optimization := recommender.OptimizeValue
	var taskParts []string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--optimize":
			if i+1 >= len(args) {
				return recommender.OptimizeValue, "", false
			}
			parsed, ok := recommender.ParseOptimization(args[i+1])
			if !ok {
				return recommender.OptimizeValue, "", false
			}
			optimization = parsed
			i++
		case strings.HasPrefix(arg, "--optimize="):
			parsed, ok := recommender.ParseOptimization(strings.TrimPrefix(arg, "--optimize="))
			if !ok {
				return recommender.OptimizeValue, "", false
			}
			optimization = parsed
		case strings.HasPrefix(arg, "--"):
			return recommender.OptimizeValue, "", false
		default:
			taskParts = append(taskParts, arg)
		}
	}

	return optimization, strings.TrimSpace(strings.Join(taskParts, " ")), true
}
