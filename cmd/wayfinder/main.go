package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/dhohner/wayfinder/internal/recommender"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: wayfinder \"describe the task\"")
		os.Exit(2)
	}

	task := strings.TrimSpace(strings.Join(os.Args[1:], " "))
	if task == "" {
		fmt.Fprintln(os.Stderr, "Usage: wayfinder \"describe the task\"")
		os.Exit(2)
	}

	rec := recommender.Recommend(task)
	fmt.Println(recommender.Format(rec))
}
