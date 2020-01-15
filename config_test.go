package main

import (
	"strings"
	"testing"
)

const validCSV = `# state, old state, existing, owned, action
OK,WARNING,true,comment
CRITICAL,UNKNOWN,false,ignore
OK,WARNING,true,create
CRITICAL,UNKNOWN,false,delete
OK,WARNING,true,status:resolved`

const invalidCSVState = `,WARNING,true,comment`
const invalidCSVBool0 = `OK,WARNING,ŧ®üé,comment`
const invalidCSVBool1 = `OK,WARNING,fæðlſ€,comment`
const invalidCSVAction0 = `OK,WARNING,true,¢ömm€nŧ`
const invalidCSVAction1 = `OK,WARNING,true,status:`
const invalidCSVAction2 = `OK,WARNING,true,status`
const invalidCSVAction3 = `OK,WARNING,true,foobar:`

func TestReadMappings(t *testing.T) {
	r := strings.NewReader(validCSV)
	ms, err := readMappings(r)
	if err != nil {
		t.Error(err)
	}

	// number of valid records. can't use count of \n here as we may have comments in csv,
	// so it's hardcoded for now.
	if len(ms) != 5 {
		t.Fail()
	}

	t.Log(ms)

	for _, v := range []string{invalidCSVState, invalidCSVBool0, invalidCSVBool1, invalidCSVAction0, invalidCSVAction1, invalidCSVAction2, invalidCSVAction2} {
		r := strings.NewReader(v)
		_, err := readMappings(r)
		if err == nil {
			t.Fail()
			t.Logf("expected error while parsing invalid CSV: %v", v)
		}
	}
}
