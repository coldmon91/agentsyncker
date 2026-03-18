package converter

import (
	"strings"
	"testing"
)

func TestMDToTOML(t *testing.T) {
	md := []byte("---\ndescription: \"Run tests with coverage\"\n---\nRun all tests.\n")
	toml, err := MDToTOML(md)
	if err != nil {
		t.Fatalf("md->toml failed: %v", err)
	}
	want := "description = \"Run tests with coverage\"\nprompt = '''\nRun all tests.\n'''\n"
	if string(toml) != want {
		t.Fatalf("unexpected toml:\n%s", string(toml))
	}
}

func TestTOMLToMD(t *testing.T) {
	toml := []byte("description = \"Run tests with coverage\"\nprompt = \"\"\"\nRun all tests.\n\"\"\"\n")
	md, err := TOMLToMD(toml)
	if err != nil {
		t.Fatalf("toml->md failed: %v", err)
	}
	want := "---\ndescription: \"Run tests with coverage\"\n---\nRun all tests.\n"
	if string(md) != want {
		t.Fatalf("unexpected markdown:\n%s", string(md))
	}
}

func TestRoundTripMarkdown(t *testing.T) {
	input := []byte("---\ndescription: \"desc\"\n---\nline1\nline2\n")
	toml, err := MDToTOML(input)
	if err != nil {
		t.Fatalf("md->toml failed: %v", err)
	}
	output, err := TOMLToMD(toml)
	if err != nil {
		t.Fatalf("toml->md failed: %v", err)
	}

	parsedInput, err := ParseMarkdown(input)
	if err != nil {
		t.Fatalf("parse input failed: %v", err)
	}
	parsedOutput, err := ParseMarkdown(output)
	if err != nil {
		t.Fatalf("parse output failed: %v", err)
	}

	if parsedInput != parsedOutput {
		t.Fatalf("roundtrip mismatch:\nin=%+v\nout=%+v", parsedInput, parsedOutput)
	}
}

func TestMDToTOMLPreservesBackslashes(t *testing.T) {
	md := []byte("---\ndescription: \"desc\"\n---\nstatus(open\\|confirmed\\|rejected)\n")
	toml, err := MDToTOML(md)
	if err != nil {
		t.Fatalf("md->toml failed: %v", err)
	}

	if !strings.Contains(string(toml), "prompt = '''\nstatus(open\\|confirmed\\|rejected)\n'''") {
		t.Fatalf("toml did not preserve prompt safely:\n%s", string(toml))
	}
}

func TestTOMLToMDMultilineLiteral(t *testing.T) {
	toml := []byte("description = \"desc\"\nprompt = '''\nline1\nline2\n'''\n")
	md, err := TOMLToMD(toml)
	if err != nil {
		t.Fatalf("toml->md failed: %v", err)
	}

	want := "---\ndescription: \"desc\"\n---\nline1\nline2\n"
	if string(md) != want {
		t.Fatalf("unexpected markdown:\n%s", string(md))
	}
}

func TestMDToTOMLFallsBackWhenPromptContainsTripleSingleQuotes(t *testing.T) {
	md := []byte("---\ndescription: \"desc\"\n---\nalpha ''' beta\n")
	toml, err := MDToTOML(md)
	if err != nil {
		t.Fatalf("md->toml failed: %v", err)
	}

	want := "description = \"desc\"\nprompt = \"alpha ''' beta\"\n"
	if string(toml) != want {
		t.Fatalf("unexpected toml:\n%s", string(toml))
	}
}

func TestRoundTripTOML(t *testing.T) {
	input := []byte("description = \"desc\"\nprompt = \"\"\"\nline1\nline2\n\"\"\"\n")
	md, err := TOMLToMD(input)
	if err != nil {
		t.Fatalf("toml->md failed: %v", err)
	}
	output, err := MDToTOML(md)
	if err != nil {
		t.Fatalf("md->toml failed: %v", err)
	}

	parsedInput, err := ParseTOML(input)
	if err != nil {
		t.Fatalf("parse input failed: %v", err)
	}
	parsedOutput, err := ParseTOML(output)
	if err != nil {
		t.Fatalf("parse output failed: %v", err)
	}

	if parsedInput != parsedOutput {
		t.Fatalf("roundtrip mismatch:\nin=%+v\nout=%+v", parsedInput, parsedOutput)
	}
}
