package main

import (
	"strings"
)

func FindURLs(text string, replacefunc func(string) string) string {
	newtext := ""

	i := 0
	j := 0
	for {
		if i >= len(text) {
			break
		}

		for !(strings.HasPrefix(text[j:], "http:") || strings.HasPrefix(text[j:], "https:")) && j < len(text) {
			j++
		}

		newtext = newtext + text[i:j]

		if j >= len(text) {
			break
		}

		i = j

		j = strings.IndexAny(text[i:], " ]>)\r\n\t")
		if j < 0 {
			j = len(text)
		} else {
			j += i
		}

		replaceurl := replacefunc(text[i:j])

		newtext = newtext + replaceurl

		i = j
	}

	return newtext
}
