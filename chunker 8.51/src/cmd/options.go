package main

type options struct {
	Command    string
	URL        string
	Manifest   string
	OutputDir  string
	CompileDir string
	SourceDir  string
	Threads    int
}

func parseOptions(args []string) (*options, error) {
	if len(args) == 0 {
		return nil, nil
	}

	opts := &options{Command: args[0], Threads: 4}
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "-u", "--url":
			if i+1 >= len(args) {
				return nil, errMissingValue(args[i])
			}
			opts.URL = args[i+1]
			i++
		case "-m", "--manifest":
			if i+1 >= len(args) {
				return nil, errMissingValue(args[i])
			}
			opts.Manifest = args[i+1]
			i++
		case "-o", "--output":
			if i+1 >= len(args) {
				return nil, errMissingValue(args[i])
			}
			opts.OutputDir = args[i+1]
			i++
		case "-c", "--compile":
			if i+1 >= len(args) {
				return nil, errMissingValue(args[i])
			}
			opts.CompileDir = args[i+1]
			i++
		case "-s", "--source":
			if i+1 >= len(args) {
				return nil, errMissingValue(args[i])
			}
			opts.SourceDir = args[i+1]
			i++
		case "-t", "--threads":
			if i+1 >= len(args) {
				return nil, errMissingValue(args[i])
			}
			opts.Threads = 4
			i++
		default:
			return nil, errUnknownArg(args[i])
		}
	}

	return opts, nil
}

func errMissingValue(arg string) error {
	return &argError{arg: arg, msg: "missing value"}
}

func errUnknownArg(arg string) error {
	return &argError{arg: arg, msg: "unknown argument"}
}

type argError struct {
	arg string
	msg string
}

func (e *argError) Error() string {
	return e.arg + ": " + e.msg
}
