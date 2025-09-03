package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/FlameInTheDark/disai/cmd/tool/location"
	"github.com/FlameInTheDark/disai/cmd/tool/weather"
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

type WeatherForecastArguments struct {
	CityName string `json:"city_name" jsonschema:"required,description=Name of the city to get weather forecast for. Eg: London"`
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

var LocationClient *location.Client
var WeatherClient *weather.WeatherClient

func main() {
	transport := http.NewHTTPTransport("/mcp")
	transport.WithAddr(":8089")

	cfg := NewConfig("./tool.yaml")

	LocationClient = location.NewClient(cfg.GeonamesUsername, nil)
	WeatherClient = weather.NewWeatherClient(cfg.WeatherKey)

	server := mcp_golang.NewServer(transport, mcp_golang.WithName("Local MCP Server"), mcp_golang.WithVersion("1.0.0"))
	err := server.RegisterTool("search", "Search the internet with given query.", func(arguments SearchWebArguments) (*mcp_golang.ToolResponse, error) {
		slog.Info("search initiated", slog.String("query", arguments.Query))
		c := resty.New()
		defer c.Close()
		resp, err := c.R().SetQueryParam("q", arguments.Query).SetQueryParam("format", "json").Get("http://192.168.1.129:8888")
		if err != nil {
			slog.Error("unable to search", slog.String("error", err.Error()))
			return mcp_golang.NewToolResponse(mcp_golang.NewTextContent(fmt.Sprintf("Error searching for '%s': %s", arguments.Query, err.Error()))), nil
		}
		var results SearchResults
		err = json.Unmarshal(resp.Bytes(), &results)
		if err != nil {
			return mcp_golang.NewToolResponse(mcp_golang.NewTextContent(fmt.Sprintf("Error searching for '%s': %s", arguments.Query, err.Error()))), nil
		}
		var respStr string
		for i, r := range results.Results {
			respStr += fmt.Sprintf("[%[1]d] Title: %[2]s\n[%[1]d] URL Source: %[3]s\n[%[1]d] Description: %[4]s\n\n", i+1, r.Title, r.URL, r.Content)
		}
		return mcp_golang.NewToolResponse(mcp_golang.NewTextContent(respStr)), nil
	})
	if err != nil {
		panic(err)
	}

	err = server.RegisterTool("fetch_url", "Fetches minified raw HTML from the given URL.", func(arguments URLFucntionArguments) (*mcp_golang.ToolResponse, error) {
		slog.Info("url opening", slog.String("url", arguments.URL))
		c := resty.New()
		defer c.Close()
		resp, err := c.R().Get(arguments.URL)
		if err != nil {
			slog.Error("unable to open URL", slog.String("error", err.Error()))
			return mcp_golang.NewToolResponse(mcp_golang.NewTextContent(fmt.Sprintf("Unable to open URL %s: %s", arguments.URL, err.Error()))), nil
		}

		if resp.StatusCode() != 200 {
			slog.Error("unable to open URL", slog.Int("status", resp.StatusCode()))
			return mcp_golang.NewToolResponse(mcp_golang.NewTextContent(fmt.Sprintf("Unable to open URL %s: status code %s", arguments.URL, resp.Status()))), nil
		}
		return mcp_golang.NewToolResponse(mcp_golang.NewTextContent(RemoveDangerousTagsAndAttrs(resp.String()))), nil
	})
	if err != nil {
		panic(err)
	}

	err = server.RegisterTool("jina_fetch_url", "Fetches article text from a given URL using Jina.ai in Markdown syntax", func(arguments URLFucntionArguments) (*mcp_golang.ToolResponse, error) {
		slog.Info("jina url opening", slog.String("url", arguments.URL))
		c := resty.New()
		defer c.Close()
		resp, err := c.R().Get(fmt.Sprintf("https://r.jina.ai/%s", arguments.URL))
		if err != nil {
			slog.Error("unable to open URL", slog.String("error", err.Error()))
			return mcp_golang.NewToolResponse(mcp_golang.NewTextContent(fmt.Sprintf("Unable to open URL %s: %s", arguments.URL, err.Error()))), nil
		}

		if resp.StatusCode() != 200 {
			slog.Error("unable to open URL", slog.Int("status", resp.StatusCode()))
			return mcp_golang.NewToolResponse(mcp_golang.NewTextContent(fmt.Sprintf("Unable to open URL %s: status code %s", arguments.URL, resp.Status()))), nil
		}
		return mcp_golang.NewToolResponse(mcp_golang.NewTextContent(resp.String())), nil
	})
	if err != nil {
		panic(err)
	}

	err = server.RegisterTool("get_weather_forecast", "Get weather forecast for specific city. If you need weather forecast you should use this tool!", func(arguments WeatherForecastArguments) (*mcp_golang.ToolResponse, error) {
		slog.Info("get forecast", slog.String("city", arguments.CityName))

		loc, err := LocationClient.Search(context.Background(), location.SearchParams{
			Q:       arguments.CityName,
			MaxRows: 1,
		})
		if err != nil {
			slog.Error("unable to search location", slog.String("error", err.Error()))
			return mcp_golang.NewToolResponse(mcp_golang.NewTextContent(fmt.Sprintf("Unable to get location: %s", err.Error()))), nil
		}

		if len(loc.Geonames) < 1 {
			slog.Error("unable to search location", slog.String("error", "no location found"))
			return mcp_golang.NewToolResponse(mcp_golang.NewTextContent("No location found")), nil
		}

		fc, err := WeatherClient.GetForecast(loc.Geonames[0].Latitude, loc.Geonames[0].Longitude, 3)
		if err != nil {
			slog.Error("unable to get weather forecast", slog.String("error", err.Error()))
			return mcp_golang.NewToolResponse(mcp_golang.NewTextContent(fmt.Sprintf("Unable to get weather forecast: %s", err.Error()))), nil
		}

		var b strings.Builder
		b.WriteString(fmt.Sprintf("# Weather forecast for %s:\n", fc.Location.Name))
		b.WriteString(fmt.Sprintf("Region: %s\nCountry: %s\nCurrent local time: %s\nWind direction: %s\n",
			fc.Location.Region, fc.Location.Country, fc.Location.Localtime, fc.Current.WindDir))

		for _, tfc := range fc.Forecast.ForecastDay {
			b.WriteString(fmt.Sprintf("\n\n## Forecast for %s:\n", tfc.Date))
			b.WriteString(fmt.Sprintf(
				"Max temp: %.1f°C\nMin temp: %.1f°C\nAvg temp: %.1f°C\nAvg visibility: %.1f km\n"+
					"Avg humidity: %.0f%%\nTotal precipitation: %.1f mm\nTotal snow: %.1f cm\n"+
					"Chance of rain: %d%%\nChance of snow: %d%%\nCondition: %s\n",
				tfc.Day.MaxTempC, tfc.Day.MinTempC, tfc.Day.AvgTempC, tfc.Day.AvgVisKm,
				tfc.Day.AvgHumidity, tfc.Day.TotalPrecipMm, tfc.Day.TotalSnowCm,
				tfc.Day.DailyChanceOfRain, tfc.Day.DailyChanceOfSnow, tfc.Day.Condition.Text))
		}

		return mcp_golang.NewToolResponse(mcp_golang.NewTextContent(b.String())), nil
	})
	if err != nil {
		panic(err)
	}

	err = server.Serve()
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
