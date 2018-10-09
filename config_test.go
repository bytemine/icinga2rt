package main

import (
	"strings"
	"testing"
)

const validCSV = `# state, old state, existing, owned, action
OK,WARNING,true,comment
CRITICAL,UNKNOWN,false,ignore
OK,WARNING,true,create
CRITICAL,UNKNOWN,false,delete`

const invalidCSVState = `,WARNING,true,comment`
const invalidCSVBool0 = `OK,WARNING,ŧ®üé,comment`
const invalidCSVBool1 = `OK,WARNING,fæðlſ€,comment`
const invalidCSVAction = `OK,WARNING,true,¢ömm€nŧ`

func TestReadMappings(t *testing.T) {
	r := strings.NewReader(validCSV)
	ms, err := readMappings(r)
	if err != nil {
		t.Error(err)
	}

	if len(ms) != 4 {
		t.Fail()
	}

	t.Log(ms)

	for _, v := range []string{invalidCSVState, invalidCSVBool0, invalidCSVBool1, invalidCSVAction} {
		r := strings.NewReader(v)
		_, err := readMappings(r)
		if err == nil {
			t.Fail()
			t.Logf("expected error while parsing invalid CSV: %v", v)
		}
	}
}
