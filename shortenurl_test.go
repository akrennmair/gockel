package main

import (
	"testing"
)

func TestFindURLs(t *testing.T) {
	text := FindURLs("foo bar baz quux", func(u string) string {
		t.Fatalf("found URL in text without URL: %s", u)
		return u
	})
	if text != "foo bar baz quux" {
		t.Fatal("FindURL returned different string: %s", text)
	}

	text = FindURLs("http://foobar.com/", func(u string) string {
		if u != "http://foobar.com/" {
			t.Fatalf("extracted URL other than http://foobar.com: %s", u)
		}
		return u
	})
	if text != "http://foobar.com/" {
		t.Fatalf("FindURL returned different string: %s", text)
	}

	text = FindURLs("foo https://barfoo.com/ bar", func(u string) string {
		if u != "https://barfoo.com/" {
			t.Fatalf("extracted URL other than https://barfoo.com/: %s", u)
		}
		return u
	})
	if text != "foo https://barfoo.com/ bar" {
		t.Fatalf("FindURL returned different string: %s", text)
	}

	text = FindURLs("foo http://quux.com/", func(u string) string {
		if u != "http://quux.com/" {
			t.Fatalf("extracted URL other than http://quux.com/: %s", u)
		}
		return u
	})
}
