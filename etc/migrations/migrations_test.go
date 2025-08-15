package migrations_test

import (
	"fmt"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/jsnfwlr/go11y/etc/migrations"
)

func TestValidateMigrations(t *testing.T) {
	files, err := os.ReadDir("./")
	if err != nil {
		t.Fatalf("could not read files from the directory this test is in: %s", err)
	}

	mFiles, err := migrations.Migrations.ReadDir(".")
	if err != nil {
		t.Fatalf("could not read files from the embeddedFS: %s", err)
	}

	sqlFiles := []string{}
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".sql") {
			sqlFiles = append(sqlFiles, f.Name())
		}
	}

	slices.Sort(sqlFiles)

	sqlMigs := []string{}
	for _, f := range mFiles {
		if strings.HasSuffix(f.Name(), ".sql") {
			sqlMigs = append(sqlMigs, f.Name())
		}
	}

	slices.Sort(sqlMigs)

	lTV, lI, lL, err := makeInfo(t, sqlFiles)
	if err != nil {
		t.Fatalf("could not get local information: %s", err)
	}

	mTV, mI, mL, err := makeInfo(t, sqlMigs)
	if err != nil {
		t.Fatalf("could not get migration information: %s", err)
	}

	t.Run("I want my MTV", func(t *testing.T) {
		// I want my MTV https://www.youtube.com/watch?v=wTP2RUD_cL0
		if mTV != lTV {
			t.Errorf("Target version should be %d, not %d", lTV, mTV)
		}
	})

	t.Run("Lets compare notes", func(t *testing.T) {
		if mI != lI {
			if len(mL) != len(lL) {
				t.Errorf("length of info differs: migrations - %d vs local - %d", len(mL), len(lL))
			}
			for i := range mL {
				if mL[i].Sequence != lL[i].Sequence {
					t.Errorf("%d - %d vs %d", i, mL[i].Sequence, lL[i].Sequence)
				}
				if mL[i].Name != lL[i].Name {
					t.Errorf("%d - %s vs %s", i, mL[i].Name, lL[i].Name)
				}
			}
		}
	})
}

type infoLine struct {
	Sequence int
	Name     string
}

func makeInfo(t *testing.T, files []string) (targetVersion int32, information string, lines []infoLine, fault error) {
	t.Helper()

	var tv int32
	var l []infoLine

	i := ""

	fPartReg := regexp.MustCompile(`([0-9]+)_(.*)\.sql`)
	numLeadZero := regexp.MustCompile(`^0+`)
	for _, file := range files {
		matches := fPartReg.FindAllStringSubmatch(file, -1)

		indicator := "  "

		num := numLeadZero.ReplaceAllString(matches[0][1], "")
		i += fmt.Sprintf("%2s %3s %s\n", indicator, num, matches[0][2])

		n, err := strconv.Atoi(num)
		if err != nil {
			return 0, "", []infoLine{}, fmt.Errorf("could not convert %s to int", num)
		}

		l = append(l, infoLine{Sequence: n, Name: matches[0][2]})
		if int32(n) > tv {
			tv = int32(n)
		}
	}

	return tv, i, l, nil
}
