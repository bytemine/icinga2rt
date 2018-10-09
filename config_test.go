package main

import (
	"strings"
	"testing"
)

const validCSV = `# state, old state, existing, owned, action
OK,WARNING,true,false,comment
CRITICAL,UNKNOWN,true,true,ignore
OK,WARNING,true,false,create
CRITICAL,UNKNOWN,true,true,delete`

const invalidCSVState = `,WARNING,true,false,comment`
const invalidCSVBool0 = `OK,WARNING,ŧ®üé,false,comment`
const invalidCSVBool1 = `OK,WARNING,true,fæðlſ€,comment`
const invalidCSVAction = `OK,WARNING,true,false,¢ömm€nŧ`

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
