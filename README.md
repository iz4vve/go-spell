# go-spell
Spellchecker for Go to get typos from single files or from file pattern (using glob).

## Build / install the binary

```
$ git clone https://github.optum.com/pmascolo/go-spell.git
$ cd go-spell
$ go build -o go-spell .
```

## Usage:

Detailed usage is described in the 
```bash
$ go-spell -h
go-spell.
        Usage:
                go-spell --model-path=<model> FILE [--target=<target>][--threshold=<threshold>]
                go-spell batch --model-path=<model> DIRECTORY [--target=<target>][--threshold=<threshold>]
                go-spell train --dictionary=<dictionary> --model-output=<modeloutput>
                go-spell -h | --help

        Options:
                -h --help                         Show this screen.
                --model-path=<model>                      Path to an existing model.
                --dictionary=<dictionary>                 Path to words list to use for training.
                --target=<target>                                 Target directory for results [default: results.json]
                --threshold=<threshold>                   Minimum count of error occurrences to be reported [default: 1]
                --model-output=<modeloutput>      Path to output model [default: wordlist.txt]
```