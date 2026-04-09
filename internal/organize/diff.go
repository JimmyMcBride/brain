package organize

import "github.com/pmezard/go-difflib/difflib"

func UnifiedDiff(fromName, toName, before, after string) (string, error) {
	return difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
		A:        difflib.SplitLines(before),
		B:        difflib.SplitLines(after),
		FromFile: fromName,
		ToFile:   toName,
		Context:  3,
	})
}
