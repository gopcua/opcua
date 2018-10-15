package opc

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func ReadNodeIDs(filename string) (map[string]int, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	csvr := csv.NewReader(f)
	recs, err := csvr.ReadAll()
	if err != nil {
		return nil, err
	}

	const suffix = "_Encoding_DefaultBinary"

	m := map[string]int{}
	for _, r := range recs {
		// we are only interested in the object ids for the binary encoding
		name := r[0]
		if !strings.HasSuffix(name, suffix) {
			continue
		}
		name = strings.TrimSuffix(name, suffix)

		id, err := strconv.Atoi(r[1])
		if err != nil {
			return nil, fmt.Errorf("cannot parse %s of %s: %s", r[1], name, err)
		}

		m[name] = id
	}
	return m, nil
}
