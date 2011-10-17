package main

import (
	"strings"
	"url"
	"http"
	"bytes"
	"io/ioutil"
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

		j = strings.IndexAny(text[i:], " ]>)")
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

func ShortenURL(u string) string {
	resp, err := http.Get("http://krzz.de/_api/save?url=" + url.QueryEscape(u))
	if err != nil {
		return u
	}

	if resp.StatusCode != 200 {
		return u
	}

	newurl, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return u
	}

	newurl = bytes.Trim(newurl, "\r\n ")

	if len(newurl) > len(u) {
		return u
	}

	return string(newurl)
}
