package main

import (
	"github.com/gostaticanalysis/nilerr"
	"github.com/max-marek-projects/shortener/cmd/staticlint/exitcheck"
	"github.com/securego/gosec/v2/goanalysis"
	"github.com/timakin/bodyclose/passes/bodyclose"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/appends"
	"golang.org/x/tools/go/analysis/passes/asmdecl"
	"golang.org/x/tools/go/analysis/passes/assign"
	"golang.org/x/tools/go/analysis/passes/atomic"
	"golang.org/x/tools/go/analysis/passes/bools"
	"golang.org/x/tools/go/analysis/passes/buildtag"
	"golang.org/x/tools/go/analysis/passes/cgocall"
	"golang.org/x/tools/go/analysis/passes/composite"
	"golang.org/x/tools/go/analysis/passes/copylock"
	"golang.org/x/tools/go/analysis/passes/defers"
	"golang.org/x/tools/go/analysis/passes/directive"
	"golang.org/x/tools/go/analysis/passes/errorsas"
	"golang.org/x/tools/go/analysis/passes/framepointer"
	"golang.org/x/tools/go/analysis/passes/hostport"
	"golang.org/x/tools/go/analysis/passes/httpresponse"
	"golang.org/x/tools/go/analysis/passes/ifaceassert"
	"golang.org/x/tools/go/analysis/passes/loopclosure"
	"golang.org/x/tools/go/analysis/passes/lostcancel"
	"golang.org/x/tools/go/analysis/passes/nilfunc"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/shift"
	"golang.org/x/tools/go/analysis/passes/sigchanyzer"
	"golang.org/x/tools/go/analysis/passes/slog"
	"golang.org/x/tools/go/analysis/passes/stdmethods"
	"golang.org/x/tools/go/analysis/passes/stdversion"
	"golang.org/x/tools/go/analysis/passes/stringintconv"
	"golang.org/x/tools/go/analysis/passes/structtag"
	"golang.org/x/tools/go/analysis/passes/testinggoroutine"
	"golang.org/x/tools/go/analysis/passes/tests"
	"golang.org/x/tools/go/analysis/passes/timeformat"
	"golang.org/x/tools/go/analysis/passes/unmarshal"
	"golang.org/x/tools/go/analysis/passes/unreachable"
	"golang.org/x/tools/go/analysis/passes/unsafeptr"
	"golang.org/x/tools/go/analysis/passes/unusedresult"
	"golang.org/x/tools/go/analysis/passes/waitgroup"
	"honnef.co/go/tools/staticcheck"
)

func main() {
	// parse flags
	// var includeStaticCheckRaw, excludeStaticCheckRaw string
	// flag.StringVar(&includeStaticCheckRaw, "staticcheck-include", "",
	// 	"список анализаторов staticcheck для включения (через запятую), например: S1000,ST1000")
	// flag.StringVar(&excludeStaticCheckRaw, "staticcheck-exclude", "",
	// 	"список анализаторов staticcheck для исключения (через запятую)")
	// flag.Parse()

	// includeStaticCheck := parseList(includeStaticCheckRaw)
	// excludeStaticCheck := parseList(excludeStaticCheckRaw)

	// standard array obtained from https://go.googlesource.com/tools/+/28ff1811c64c77737ccead3a5fc3b8bcb9dfaef4/go/analysis/suite/vet/vet.go
	var allAnalyzers = []*analysis.Analyzer{
		appends.Analyzer,
		asmdecl.Analyzer,
		assign.Analyzer,
		atomic.Analyzer,
		bools.Analyzer,
		buildtag.Analyzer,
		cgocall.Analyzer,
		composite.Analyzer,
		copylock.Analyzer,
		defers.Analyzer,
		directive.Analyzer,
		errorsas.Analyzer,
		// fieldalignment.Analyzer omitted: too noisy
		framepointer.Analyzer,
		httpresponse.Analyzer,
		hostport.Analyzer,
		ifaceassert.Analyzer,
		loopclosure.Analyzer,
		lostcancel.Analyzer,
		nilfunc.Analyzer,
		printf.Analyzer,
		// shadow.Analyzer omitted: too noisy
		shift.Analyzer,
		sigchanyzer.Analyzer,
		slog.Analyzer,
		stdmethods.Analyzer,
		stdversion.Analyzer,
		stringintconv.Analyzer,
		structtag.Analyzer,
		tests.Analyzer,
		testinggoroutine.Analyzer,
		timeformat.Analyzer,
		unmarshal.Analyzer,
		unreachable.Analyzer,
		unsafeptr.Analyzer,
		unusedresult.Analyzer,
		waitgroup.Analyzer,
	}

	// staticcheck
	for _, a := range staticcheck.Analyzers {
		// add SA analyzers
		if len(a.Analyzer.Name) >= 2 && a.Analyzer.Name[:2] == "SA" {
			allAnalyzers = append(allAnalyzers, a.Analyzer)
			continue
		}
		// add ST analyzers
		if len(a.Analyzer.Name) >= 2 && a.Analyzer.Name[:2] == "ST" {
			allAnalyzers = append(allAnalyzers, a.Analyzer)
			continue
		}
		// add S analyzers
		if len(a.Analyzer.Name) >= 1 && a.Analyzer.Name[:1] == "S" {
			allAnalyzers = append(allAnalyzers, a.Analyzer)
			continue
		}
		// add QF analyzers
		if len(a.Analyzer.Name) >= 2 && a.Analyzer.Name[:2] == "QF" {
			allAnalyzers = append(allAnalyzers, a.Analyzer)
			continue
		}
	}

	// other analyzers
	allAnalyzers = append(allAnalyzers, bodyclose.Analyzer)
	allAnalyzers = append(allAnalyzers, nilerr.Analyzer)
	allAnalyzers = append(allAnalyzers, goanalysis.Analyzer)

	// os.Exit analyzer
	allAnalyzers = append(allAnalyzers, exitcheck.Analyzer)

	multichecker.Main(allAnalyzers...)
}
