/*
Copyright (c) Tobias Schäfer. All rights reserved.
Licensed under the MIT license, see LICENSE in the project root for details.
*/
package version

import (
	"fmt"
	"os"
)

var (
	GitCommit, Version string
)

// Release returns the current version of the application.
func Release() string {
	if Version == "" {
		Version = "dev"
	}

	return Version
}

// Commit returns the current git commit hash of the application.
func Commit() string {
	return GitCommit
}

// Banner returns the ASCII art banner of the application.
func Banner() string {
	return `
                       _                  _       _
  ___ ___  _ __  _ __ | |_ _ __ ____  ___| | ____| |
 / __/ _ \| '_ \| '_ \| __| '__/ _  |/ __| |/ / _  |
| (_| (_) | | | | | | | |_| | | (_| | (__|   < (_| |
 \___\___/|_| |_|_| |_|\__|_|  \__,_|\___|_|\_\__,_|
 `
}

// Print prints the colored banner, release version, and commit hash to the
// console. Coloring can be disabled by setting the NO_COLOR environment
// variable to "1" or "true".
func Print() {
	no_color, ok := os.LookupEnv("NO_COLOR")
	if ok && no_color == "1" || no_color == "true" {
		fmt.Printf("%s\n", Banner())
	} else {
		fmt.Printf("\033[34m%s\033[0m\n", Banner())
	}
	fmt.Printf("Release: %s\n", Release())
	fmt.Printf("Commit:  %s\n", Commit())
}
