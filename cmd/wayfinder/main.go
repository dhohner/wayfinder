package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/dhohner/wayfinder/internal/recommender"
)

func main() {
	preference, task, ok := parseArgs(os.Args[1:])
	if !ok || task == "" {
		fmt.Fprintln(os.Stderr, "Usage: wayfinder [--prefer quality|cost|speed] \"describe the task\"")
		os.Exit(2)
	}

	rec := recommender.RecommendWithPreference(task, preference)
	fmt.Println(recommender.Format(rec))
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
