package main

import (
	"bytes"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
)

var forbiddenTags = map[string]bool{
	"script":   true,
	"style":    true,
	"meta":     true,
	"svg":      true,
	"iframe":   true,
	"source":   true,
	"link":     true,
	"a":        true,
	"input":    true,
	"textarea": true,
	"button":   true,
	"select":   true,
	"option":   true,
	"form":     true,
	"noscript": true,
}

var forbiddenAttrs = map[string]bool{
	"class":    true,
	"style":    true,
	"id":       true,
	"width":    true,
	"height":   true,
	"loading":  true,
	"rel":      true,
	"tabindex": true,
	"role":     true,
	"size":     true,
	"name":     true,
}

func cleanNode(n *html.Node) *html.Node {
	// Drop comment nodes entirely
	if n.Type == html.CommentNode {
		return nil
	}

	// Drop forbidden elements completely
	if n.Type == html.ElementNode && forbiddenTags[strings.ToLower(n.Data)] {
		return nil
	}

	// Filter attributes if it's an element node
	if n.Type == html.ElementNode {
		var attrs []html.Attribute
		for _, a := range n.Attr {
			if !forbiddenAttrs[strings.ToLower(a.Key)] {
				keyLower := strings.ToLower(a.Key)
				if forbiddenAttrs[keyLower] || strings.HasPrefix(keyLower, "data-") || strings.HasPrefix(keyLower, "aria-") {
					continue
				}
				attrs = append(attrs, a)
			}
		}
		n.Attr = attrs
	}

	// Recursively process children
	for c := n.FirstChild; c != nil; {
		next := c.NextSibling
		if cleaned := cleanNode(c); cleaned == nil {
			n.RemoveChild(c)
		}
		c = next
	}

	return n
}

func RemoveDangerousTagsAndAttrs(htmlStr string) string {
	if !strings.HasPrefix(htmlStr, "<!DOCTYPE html>") && !strings.HasPrefix(htmlStr, "<html") {
		return htmlStr
	}
	doc, err := html.Parse(strings.NewReader(htmlStr))
	if err != nil {
		return htmlStr // fallback to original if broken HTML
	}

	cleaned := cleanNode(doc)

	var buf bytes.Buffer
	if err := html.Render(&buf, cleaned); err != nil {
		return htmlStr
	}
	out := buf.String()
	out = strings.TrimSpace(out)
	out = strings.ReplaceAll(out, "\n", " ")
	out = strings.ReplaceAll(out, "\t", " ")
	spaceRe := regexp.MustCompile(`\s+`)
	out = spaceRe.ReplaceAllString(out, " ")
	return collapseSpaces(out)
}

func CleanHTML(html string) (string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return "", err
	}

	var clean func(*goquery.Selection) bool
	clean = func(sel *goquery.Selection) bool {
		hasText := strings.TrimSpace(sel.Text()) != ""

		// Clean children first
		sel.Children().Each(func(i int, child *goquery.Selection) {
			if !clean(child) {
				child.Remove()
			}
		})

		// If element has no text and no children, drop it
		if !hasText && sel.Children().Length() == 0 {
			return false
		}

		// If element has no attributes and no direct text -> unwrap it completely
		if len(sel.Nodes[0].Attr) == 0 && !directText(sel) {
			sel.Contents().Unwrap()
		}

		return true
	}

	// Start from <body> children if exists, otherwise from root
	root := doc.Selection
	doc.Find("body").Children().Each(func(i int, s *goquery.Selection) {
		if !clean(s) {
			s.Remove()
		}
	})
	if doc.Find("body").Length() == 0 {
		root.Children().Each(func(i int, s *goquery.Selection) {
			if !clean(s) {
				s.Remove()
			}
		})
	}

	// Extract cleaned HTML
	htmlOut, err := doc.Html()
	if err != nil {
		return "", err
	}
	return htmlOut, nil
}

// directText checks if selection contains non-empty text nodes directly
func directText(sel *goquery.Selection) bool {
	if len(sel.Nodes) == 0 {
		return false
	}
	for n := range sel.Nodes[0].ChildNodes() {
		if n.Type == html.TextNode && strings.TrimSpace(n.Data) != "" {
			return true
		}
	}
	return false
}

func collapseSpaces(s string) string {
	// collapse any whitespace to a single space
	spaceRe := regexp.MustCompile(`\s+`)
	s = spaceRe.ReplaceAllString(s, " ")
	// remove spaces between tags: ">   <" -> "><"
	betweenTags := regexp.MustCompile(`>\s+<`)
	s = betweenTags.ReplaceAllString(s, "><")
	return strings.TrimSpace(s)
}
