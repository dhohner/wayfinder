package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/dhohner/wayfinder/internal/recommender"
)

const usage = "Usage: wayfinder [--prefer quality|cost|speed] \"describe the task\""

// Recommender is the behavior required by the CLI. It keeps command parsing
// independent from the recommendation implementation and easy to test.
type Recommender interface {
	RecommendWithPreference(task string, preference recommender.Preference) recommender.Recommendation
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

	preference, task, ok := parseArgs(args)
	if !ok || task == "" {
		fmt.Fprintln(stderr, usage)
		return 2
	}

	fmt.Fprintln(stdout, recommender.Format(rec.RecommendWithPreference(task, preference)))
	return 0
}

func parseArgs(args []string) (recommender.Preference, string, bool) {
	var preference recommender.Preference
	var taskParts []string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--prefer":
			if i+1 >= len(args) {
				return recommender.PreferNone, "", false
			}
			parsed, ok := recommender.ParsePreference(args[i+1])
			if !ok {
				return recommender.PreferNone, "", false
			}
			preference = parsed
			i++
		case strings.HasPrefix(arg, "--prefer="):
			parsed, ok := recommender.ParsePreference(strings.TrimPrefix(arg, "--prefer="))
			if !ok {
				return recommender.PreferNone, "", false
			}
			preference = parsed
		case strings.HasPrefix(arg, "--"):
			return recommender.PreferNone, "", false
		default:
			taskParts = append(taskParts, arg)
		}
	}

	return preference, strings.TrimSpace(strings.Join(taskParts, " ")), true
}
