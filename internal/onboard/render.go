package onboard

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
)

func RenderProviderTF() string {
	return `terraform {
  required_providers {
    peakhour = {
      source = "peakhour-io/peakhour"
    }
  }
}

provider "peakhour" {}

`
}

func RenderImportsTF(targets []ImportTarget) string {
	if len(targets) == 0 {
		return ""
	}

	// Ensure deterministic output regardless of caller ordering.
	sortedTargets := append([]ImportTarget(nil), targets...)
	sort.SliceStable(sortedTargets, func(i, j int) bool {
		if sortedTargets[i].TypeName != sortedTargets[j].TypeName {
			return sortedTargets[i].TypeName < sortedTargets[j].TypeName
		}
		return sortedTargets[i].Name < sortedTargets[j].Name
	})

	typeNameCounts := make(map[string]map[string]int)

	var buf bytes.Buffer
	for _, target := range sortedTargets {
		base := sanitizeTFName(target.Name)
		if base == "" {
			base = "x"
		}

		if _, ok := typeNameCounts[target.TypeName]; !ok {
			typeNameCounts[target.TypeName] = make(map[string]int)
		}

		typeNameCounts[target.TypeName][base]++
		ordinal := typeNameCounts[target.TypeName][base]

		localName := base
		if ordinal > 1 {
			localName = fmt.Sprintf("%s_%d", base, ordinal)
		}

		fmt.Fprintf(&buf, "import {\n  to = %s.%s\n  id = %q\n}\n\n", target.TypeName, localName, target.ImportID)
	}

	return buf.String()
}

func sanitizeTFName(input string) string {
	in := strings.ToLower(strings.TrimSpace(input))

	var b strings.Builder
	b.Grow(len(in))

	lastUnderscore := false
	for _, r := range in {
		isAlpha := r >= 'a' && r <= 'z'
		isDigit := r >= '0' && r <= '9'
		isUnderscore := r == '_'

		if isAlpha || isDigit || isUnderscore {
			if isUnderscore {
				if lastUnderscore {
					continue
				}
				lastUnderscore = true
			} else {
				lastUnderscore = false
			}
			b.WriteRune(r)
			continue
		}

		if lastUnderscore {
			continue
		}
		b.WriteByte('_')
		lastUnderscore = true
	}

	out := strings.Trim(b.String(), "_")
	out = strings.TrimLeft(out, "0123456789_")
	out = strings.Trim(out, "_")
	if out == "" {
		return "x"
	}
	return out
}
