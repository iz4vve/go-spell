// Package main contains the relevant functions to perform spellcheck
// on files or sets of files and report errors and how often they occur
package main

// TODO review ALL variable names!!!
// TODO implement thresholding

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/schollz/progressbar"

	docopt "github.com/docopt/docopt-go"
	"github.com/sajari/fuzzy"
)

func main() {
	start := time.Now()
	defer func() {
		fmt.Println("--------")
		timeTrack(start, "main")
	}()

	usage := `go-spell.
	Usage:
		go-spell --model-path=<model> FILE [--target=<target>][--threshold=<threshold>]
		go-spell batch --model-path=<model> DIRECTORY [--target=<target>][--threshold=<threshold>]
		go-spell train --dictionary=<dictionary> --model-output=<modeloutput>
		go-spell -h | --help

	Options:
		-h --help                     	  Show this screen.
		--model-path=<model>			  Path to an existing model.
		--dictionary=<dictionary>		  Path to words list to use for training.
		--target=<target>				  Target directory for results [default: results.json]
		--threshold=<threshold>			  Minimum count of error occurrences to be reported [default: 1]
		--model-output=<modeloutput>	  Path to output model [default: wordlist.txt]`

	arguments, _ := docopt.ParseDoc(usage)

	// TRAIN
	if train, _ := arguments.Bool("train"); train {
		dictionary, err := arguments.String("--dictionary")
		failOnError(err)
		modelOutput, err := arguments.String("--model-output")
		failOnError(err)
		fmt.Printf("Training model using dictionary file %s...\n", dictionary)
		fmt.Printf("Model will be saved in %s\n", modelOutput)
		trainModel(dictionary, modelOutput)
		return
	}

	// DIRECTORY
	if batch, _ := arguments.Bool("batch"); batch {
		directory, err := arguments.String("DIRECTORY")
		failOnError(err)
		if !checkFile(directory) {
			log.Fatal("path does not exist")
		}
		model := loadModel(arguments)
		errors, errs := spellcheckDir(directory, model)
		if len(errs) != 0 {
			fmt.Printf("Errors occurred: %s\n", errs)
		}
		fmt.Println("Batch errors calculated, saving results...")
		errors = updateCounts(errors)
		saveResults(errors, arguments)
		return
	}

	// FILE
	file, err := arguments.String("FILE")
	failOnError(err)
	if !checkFile(file) {
		log.Fatal("path does not exist")
	}
	model := loadModel(arguments)
	errors, err := spellcheckFile(file, model, false)
	failOnError(err)
	saveResults(errors, arguments)
}

// loadModel loads a model from a file
// Models have to be trained and saved using this script's training utility to be
// in a format that is consumable by the rest of the functions.
func loadModel(arguments docopt.Opts) *fuzzy.Model {
	defer timeTrack(time.Now(), "loadModel")
	modelPath, err := arguments.String("--model-path")
	failOnError(err)
	fmt.Printf("Loading model '%s'\n", modelPath)
	model, err := fuzzy.Load(modelPath)
	if err != nil {
		fmt.Println("Could not load a valid model. Please train a valid model and load it.")
		log.Fatal(err)
	}
	return model
}

// spellcheckDir runs spellchecking on all files that match the glob
// pattern specified by the user in the command line.
func spellcheckDir(directory string, model *fuzzy.Model) ([]ErrorCounts, []error) {
	files, err := filepath.Glob(directory)
	failOnError(err)
	fmt.Printf("Running batch job on %d files\n", len(files))

	globalErrors := []ErrorCounts{}
	errs := []error{}

	bar := progressbar.New(len(files))
	for _, file := range files {
		errors, err := spellcheckFile(file, model, true)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		for _, item := range errors {
			globalErrors = append(globalErrors, item)
		}
		bar.Add(1)
	}
	fmt.Println()
	return globalErrors, errs
}

// spellcheckFile runs spellcheck on a single file and saves the results
// as a JSON array to a file.
// The default file name for the results is 'results.json'
func spellcheckFile(file string, model *fuzzy.Model, batch bool) ([]ErrorCounts, error) {
	if !batch {
		defer timeTrack(time.Now(), "spellcheckFile")
	}
	fileContents, err := ioutil.ReadFile("test")
	if err != nil {
		return []ErrorCounts{}, err
	}
	sentences := strings.Replace(string(fileContents), "\n", " ", -1)
	errors := errorMap{}

	for _, word := range strings.Split(sentences, " ") {
		checked := model.SpellCheck(word)
		if len(checked) == 0 {
			continue
		}
		if len(checked) > 2 {
			// remove the trailing mysterious byte
			checked = checked[:len(checked)-1]
		}
		if strings.ToLower(word) != strings.ToLower(string(checked)) {
			addError(Typo{word, checked}, errors)
		}
	}

	errorList := []ErrorCounts{}
	for k, v := range errors {
		errorList = append(errorList, ErrorCounts{k, v})
	}
	sort.Slice(errorList, func(i, j int) bool { return errorList[i].Count > errorList[j].Count })
	return errorList, nil
}

// trainModel trains and persists a spellcheck model from a wordlist
func trainModel(dict string, output string) {
	defer timeTrack(time.Now(), "trainModel")
	model := fuzzy.NewModel()
	model.SetThreshold(1)
	model.SetDepth(3)

	dictionary, err := ioutil.ReadFile(dict)
	if err != nil {
		panic(err)
	}

	allWords := strings.Split(string(dictionary), "\n")
	// words := filterWords(allWords)
	fmt.Printf("Training on %d words. This may take a few minutes...\n", len(allWords))
	model.Train(allWords)
	model.Save(output)
	fmt.Println("Training complete")
}

// updateCounts updates the occurrence of errors across spellcheck results
// for multiple files
func updateCounts(errors []ErrorCounts) []ErrorCounts {
	errM := errorMap{}
	updated := []ErrorCounts{}

	for _, item := range errors {
		typo := Typo{item.Wrong, item.Correct}
		if _, ok := errM[typo]; !ok {
			errM[typo] = item.Count
			continue
		}
		errM[typo] = errM[typo] + item.Count
	}

	for k, v := range errM {
		updated = append(updated, ErrorCounts{k, v})
	}
	sort.Slice(updated, func(i, j int) bool { return updated[i].Count > updated[j].Count })
	return updated
}

// saveResults saves a JSON file containing the spellcheck results
func saveResults(errors []ErrorCounts, arguments docopt.Opts) {
	filename, err := arguments.String("--target")
	failOnError(err)
	jsonErrors, _ := json.Marshal(errors)
	ioutil.WriteFile(filename, jsonErrors, 0644)
}

// addError adds a single error to an errorMap object taking into account
// the number of times it occurred
func addError(typo Typo, errors errorMap) errorMap {
	_, ok := errors[typo]
	if !ok {
		errors[typo] = 1
		return errors
	}

	errors[typo]++
	return errors
}

// checkFile checks whether a file exists or a glob pattern returns
// a non-empty list of files
func checkFile(file string) bool {
	glob, err := filepath.Glob(file)
	if err != nil {
		return false
	}
	if _, err := os.Stat(file); os.IsNotExist(err) && len(glob) == 0 {
		return false
	}
	return true
}

// filterWords removes words containing apostrophes from a dictionary
func filterWords(w []string) []string {
	ret := []string{}
	for _, word := range w {
		if !strings.Contains(word, "'") {
			ret = append(ret, word)
		}
	}
	return ret
}

// timeTrack reports execution time for functions
func timeTrack(start time.Time, name string) {
	elapsed := time.Since(start)
	fmt.Printf("'%s' took %.3fs\n", name, elapsed.Seconds())
}

// failOnError panics when errors occur
func failOnError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

// Typo represents a typo in a text and its spellcheck
type Typo struct {
	Wrong   string `json:"wrong"`
	Correct string `json:"correct"`
}

// errorMap represents a counter of Typos
type errorMap map[Typo]int

// ErrorCounts represents Typos and their number of occurrences
type ErrorCounts struct {
	Typo
	Count int `json:"counts"`
}
