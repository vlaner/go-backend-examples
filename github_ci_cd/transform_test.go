package main

import "testing"

func TestTransformFunction(t *testing.T) {
	inputStr := "test string"
	expected := "d5579c46dfcc7f18207013e65b44e4cb4e2c2298f4ac457ba8f82743f31e930b"

	transformed := TransformString(inputStr)
	if expected != transformed {
		t.Errorf("%s != %s", expected, transformed)
	}
}

func TestOnlyNumersInString(t *testing.T) {
	inputStr := "1234"
	expected := "03ac674216f3e15c761ee1a5e255f067953623c8b388b4459e13f978d7c846f4"

	transformed := TransformString(inputStr)
	if expected != transformed {
		t.Errorf("%s != %s", expected, transformed)
	}
}

func TestEmptyString(t *testing.T) {
	inputStr := ""
	expected := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

	transformed := TransformString(inputStr)
	if expected != transformed {
		t.Errorf("%s != %s", expected, transformed)
	}
}
