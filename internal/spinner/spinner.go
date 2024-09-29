package utils

import (
	"fmt"
	"time"

	"github.com/briandowns/spinner"
)

// Global spinner instance
var s *spinner.Spinner

// StartSpinner starts the spinner
func StartSpinner() {
	s = spinner.New(spinner.CharSets[36], 100*time.Millisecond)
	s.Start()
}

// StopSpinner stops the spinner and clears its output
func StopSpinner() {
	s.Stop()
	clearSpinnerOutput()
}

// clearSpinnerOutput clears the spinner from the command line
func clearSpinnerOutput() {
	fmt.Print("\r")           // Move cursor to the beginning of the line
	for i := 0; i < 40; i++ { // Adjust the number of spaces as needed
		fmt.Print(" ")
	}
	fmt.Print("\r") // Move cursor back to the beginning again
}
