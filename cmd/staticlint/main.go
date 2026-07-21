package main

import (
	"log"
	"regexp"

	"github.com/gostaticanalysis/nilerr"
	"github.com/max-marek-projects/shortener/cmd/staticlint/exitcheck"
	"github.com/max-marek-projects/shortener/internal/logger"
	"github.com/securego/gosec/v2/goanalysis"
	"github.com/timakin/bodyclose/passes/bodyclose"
	"go.uber.org/zap"
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
	"honnef.co/go/tools/analysis/lint"
	"honnef.co/go/tools/quickfix"
	"honnef.co/go/tools/simple"
	"honnef.co/go/tools/staticcheck"
	"honnef.co/go/tools/stylecheck"
)

func main() {
	// initialize logger for debugging
	err := logger.Initialize("DEBUG")
	if err != nil {
		log.Fatalf("Unable to initialize logger: %v", err)
	}

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

	// allowed analyzers
	var sRulePattern = regexp.MustCompile(`^(?:S|ST|SA|QF)\d+$`)
	var analyzerGroups = [][]*lint.Analyzer{
		staticcheck.Analyzers,
		simple.Analyzers,
		stylecheck.Analyzers,
		quickfix.Analyzers,
	}

	// staticcheck
	for _, analyzerGroup := range analyzerGroups {
		for _, a := range analyzerGroup {
			if sRulePattern.MatchString(a.Analyzer.Name) {
				logger.Log.Debug("Added analyzer", zap.String("name", a.Analyzer.Name))
				allAnalyzers = append(allAnalyzers, a.Analyzer)
			}
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
