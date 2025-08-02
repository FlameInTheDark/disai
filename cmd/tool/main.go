package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/advancedlogic/GoOse"
	mcp_golang "github.com/metoro-io/mcp-golang"
	"github.com/metoro-io/mcp-golang/transport/http"
	"resty.dev/v3"
)

type SearchWebArguments struct {
	Query string `json:"query" jsonschema:"required,description=The query to search for"`
}

type URLFucntionArguments struct {
	URL string `json:"url" jsonschema:"required,description=The URL of the article to fetch"`
}

type SearchResults struct {
	Query   string      `json:"query"`
	Results []SearchRow `json:"results"`
}

type SearchRow struct {
	URL     string `json:"url"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

func main() {
	transport := http.NewHTTPTransport("/mcp")
	transport.WithAddr(":8089")

	server := mcp_golang.NewServer(transport, mcp_golang.WithName("Local MCP Server"), mcp_golang.WithVersion("1.0.0"))
	err := server.RegisterTool("search", "Search the internet with given query.", func(arguments SearchWebArguments) (*mcp_golang.ToolResponse, error) {
		slog.Info("search initiated", slog.String("query", arguments.Query))
		c := resty.New()
		defer c.Close()
		resp, err := c.R().SetQueryParam("q", arguments.Query).SetQueryParam("format", "json").Get("http://192.168.1.129:8888")
		if err != nil {
			return mcp_golang.NewToolResponse(mcp_golang.NewTextContent(fmt.Sprintf("Error searching for '%s': %s", arguments.Query, err.Error()))), nil
		}
		var results SearchResults
		err = json.Unmarshal(resp.Bytes(), &results)
		if err != nil {
			return mcp_golang.NewToolResponse(mcp_golang.NewTextContent(fmt.Sprintf("Error searching for '%s': %s", arguments.Query, err.Error()))), nil
		}
		var respStr string
		for i, r := range results.Results {
			var first string
			if i == 0 {
				resp, err := c.R().Get(r.URL)
				if err != nil {
					slog.Error("unable to open first url from the results", slog.String("error", err.Error()))
				} else {
					if resp.StatusCode() != 200 {
						first = fmt.Sprintf("Error opening first url from the results: status code %s\n\n", resp.Status())
					}
					g := goose.New()
					a, err := g.ExtractFromRawHTML(string(resp.Bytes()), r.URL)
					if err != nil {
						slog.Error("unable to extract content from first url from the results", slog.String("error", err.Error()))
					} else {
						first = fmt.Sprintf("First page results ---\nTitle: %s\nContent: %s\nDescription: %s, URLs: %s\n\nAll search results --- \n", a.Title, a.CleanedText, a.MetaDescription, strings.Join(a.Links, "\n"))
					}
				}
			}
			respStr += fmt.Sprintf("%s%s\n%s\n%s\n\n", first, r.Title, r.URL, r.Content)
		}
		return mcp_golang.NewToolResponse(mcp_golang.NewTextContent(respStr)), nil
	})
	if err != nil {
		panic(err)
	}

	err = server.RegisterTool("fetch_url", "Fetches article text from a given URL.", func(arguments URLFucntionArguments) (*mcp_golang.ToolResponse, error) {
		slog.Info("url opening", slog.String("url", arguments.URL))
		c := resty.New()
		defer c.Close()
		resp, err := c.R().Get(arguments.URL)
		if err != nil {
			return mcp_golang.NewToolResponse(mcp_golang.NewTextContent(fmt.Sprintf("Unable to open URL %s: %s", arguments.URL, err.Error()))), nil
		}

		if resp.StatusCode() != 200 {
			return mcp_golang.NewToolResponse(mcp_golang.NewTextContent(fmt.Sprintf("Unable to open URL %s: status code %s", arguments.URL, resp.Status()))), nil
		}

		g := goose.New()
		a, err := g.ExtractFromRawHTML(string(resp.Bytes()), arguments.URL)
		if err != nil {
			return mcp_golang.NewToolResponse(mcp_golang.NewTextContent(fmt.Sprintf("Unable to extract content from URL %s: %s", arguments.URL, err.Error()))), nil
		}
		return mcp_golang.NewToolResponse(mcp_golang.NewTextContent(fmt.Sprintf("Title: %s\nContent: %s\nDescription: %s, URLs: %s", a.Title, a.CleanedText, a.MetaDescription, strings.Join(a.Links, "\n")))), nil
	})
	if err != nil {
		panic(err)
	}

	err = server.Serve()
	if err != nil {
		panic(err)
	}

	slog.Info("Up and running")
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)
	<-signalCh
}
