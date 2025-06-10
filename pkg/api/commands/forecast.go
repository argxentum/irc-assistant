package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/repository"
	"assistant/pkg/api/style"
	"assistant/pkg/api/text"
	"assistant/pkg/config"
	"assistant/pkg/firestore"
	"assistant/pkg/log"
	"assistant/pkg/models"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const ForecastCommandName = "forecast"

const forecastAPIURL = "https://weather.googleapis.com/v1/forecast/days:lookup?location.latitude=%f&location.longitude=%f&key=%s&days=1"

type ForecastCommand struct {
	*commandStub
}

func NewForecastCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &ForecastCommand{
		commandStub: defaultCommandStub(ctx, cfg, ircs),
	}
}

func (c *ForecastCommand) Name() string {
	return ForecastCommandName
}

func (c *ForecastCommand) Description() string {
	return "Shows weather forecast for the given location."
}

func (c *ForecastCommand) Triggers() []string {
	return []string{"forecast", "fc"}
}

func (c *ForecastCommand) Usages() []string {
	return []string{"%s <location>", "%s (uses previous location)"}
}

func (c *ForecastCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *ForecastCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 0)
}

func (c *ForecastCommand) Execute(e *irc.Event) {
	logger := log.Logger()
	tokens := Tokens(e.Message())

	location := ""
	if len(tokens) > 1 {
		location = strings.Join(tokens[1:], " ")
	}

	logger.Infof(e, "⚡ %s [%s/%s] %s", c.Name(), e.From, e.ReplyTarget(), location)

	var user *models.User
	if !e.IsPrivateMessage() {
		user, _ = repository.GetUserByMask(e, e.ReplyTarget(), irc.ParseMask(e.Source), false)
	}

	if len(location) == 0 {
		if user != nil {
			location = user.Location
		} else {
			c.Replyf(e, "No previous location found. Please specify a location: %s", fmt.Sprintf(c.Usages()[0], tokens[0]))
			return
		}
	}

	geocoding, err := c.fetchGeocodingResponse(location)
	if err != nil {
		logger.Errorf(e, "failed to fetch geocoding data, %v", err)
		c.Replyf(e, fmt.Sprintf("Error fetching data for %s", style.Bold(location)))
		return
	}

	if len(geocoding.Results) == 0 {
		logger.Errorf(e, "no geocoding results found for %s", location)
		c.Replyf(e, fmt.Sprintf("No results found for %s", style.Bold(location)))
		return
	}

	formattedLocation := geocoding.Results[0].FormattedAddress
	lat := geocoding.Results[0].Geometry.Location.Lat
	lng := geocoding.Results[0].Geometry.Location.Lng

	forecast, err := c.fetchForecast(lat, lng)
	if err != nil {
		logger.Errorf(e, "failed to fetch forecast data, %v", err)
		c.Replyf(e, fmt.Sprintf("Error fetching forecast for %s", style.Bold(location)))
		return
	}

	if forecast == nil {
		logger.Errorf(e, "no forecast data found for %s", location)
		c.Replyf(e, fmt.Sprintf("No forecast data found for %s", style.Bold(location)))
		return
	}

	// update user location
	if user != nil && len(formattedLocation) > 0 {
		user.Location = formattedLocation
		if err := firestore.Get().UpdateUser(e.ReplyTarget(), user, map[string]any{"location": formattedLocation}); err != nil {
			logger.Errorf(e, "failed to update user location, %v", err)
		} else {
			logger.Debugf(e, "updated user location to %s", formattedLocation)
		}
	}

	message := c.createForecastMessage(e, forecast)
	if len(message) == 0 {
		logger.Errorf(e, "no current conditions message created for %s", location)
		c.Replyf(e, fmt.Sprintf("No current conditions data available for %s", style.Bold(location)))
		return
	}

	c.SendMessage(e, e.ReplyTarget(), fmt.Sprintf("%s - %s", style.Underline(style.Bold(formattedLocation)), message))
}

func (c *ForecastCommand) fetchGeocodingResponse(location string) (*geocodingResponse, error) {
	res, err := http.Get(fmt.Sprintf(geocodingAPIURL, url.QueryEscape(location), c.cfg.GoogleCloud.MappingAPIKey))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch geocoding data, %v", err)
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch geocoding data, received status code %d", res.StatusCode)
	}

	defer res.Body.Close()

	var response geocodingResponse
	err = json.NewDecoder(res.Body).Decode(&response)
	return &response, err
}

func (c *ForecastCommand) fetchForecast(lat, lng float64) (*forecastDaysResponse, error) {
	res, err := http.Get(fmt.Sprintf(forecastAPIURL, lat, lng, c.cfg.GoogleCloud.MappingAPIKey))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch forecast data, %v", err)
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch forecast data, received status code %d", res.StatusCode)
	}

	defer res.Body.Close()

	var response forecastDaysResponse
	err = json.NewDecoder(res.Body).Decode(&response)
	return &response, err
}

func (c *ForecastCommand) createForecastMessage(e *irc.Event, forecast *forecastDaysResponse) string {
	if len(forecast.ForecastDays) == 0 {
		return ""
	}

	f := forecast.ForecastDays[0]
	m := style.Bold("High") + ": " + fmt.Sprintf("%.0f°F / %.0f°C", convertCelsiusToFahrenheit(f.MaxTemperature.Degrees), f.MaxTemperature.Degrees) + ". " + style.Bold("Low") + ": " + fmt.Sprintf("%.0f°F / %.0f°C", convertCelsiusToFahrenheit(f.MinTemperature.Degrees), f.MinTemperature.Degrees) + ". "

	d := c.createConditionsMessage(e, f.DaytimeForecast)
	if len(d) > 0 {
		m += style.Bold("Today") + ": " + d
	}

	n := c.createConditionsMessage(e, f.NighttimeForecast)
	if len(n) > 0 {
		if !strings.HasSuffix(m, ".") {
			m += ". "
		} else {
			m += " "
		}
		m += style.Bold("Tonight") + ": " + n
	}

	if len(f.SunEvents.Sunrise) > 0 {
		t, _ := time.Parse(time.RFC3339Nano, f.SunEvents.Sunrise)
		if !t.IsZero() {
			if !strings.HasSuffix(m, ".") {
				m += ". "
			} else {
				m += " "
			}
			m += fmt.Sprintf("Sunrise: %s", t.Format("3:04 PM"))
		}
	}

	if len(f.SunEvents.Sunset) > 0 {
		t, _ := time.Parse(time.RFC3339Nano, f.SunEvents.Sunset)
		if !t.IsZero() {
			if !strings.HasSuffix(m, ".") {
				m += ". "
			} else {
				m += " "
			}
			m += fmt.Sprintf("Sunset: %s", t.Format("3:04 PM"))
		}
	}

	if len(f.MoonEvents.MoonPhase) > 0 {
		if !strings.HasSuffix(m, ".") {
			m += ". "
		} else {
			m += " "
		}
		if phase, ok := moonphases[f.MoonEvents.MoonPhase]; ok {
			m += fmt.Sprintf("Moon: %s", strings.ToLower(phase))
		} else {
			m += fmt.Sprintf("Moon: %s", strings.ToLower(strings.Replace(f.MoonEvents.MoonPhase, "_", " ", -1)))
		}
	}

	if !strings.HasSuffix(m, ".") {
		m += "."
	}

	return m
}

func (c *ForecastCommand) createConditionsMessage(e *irc.Event, conditions WeatherConditions) string {
	m := ""

	if len(conditions.WeatherCondition.Description.Text) > 0 {
		m += strings.ToLower(conditions.WeatherCondition.Description.Text)
	} else {
		m += strings.ToLower(text.Capitalize(strings.Replace(conditions.WeatherCondition.Type, "_", " ", -1), true))
	}

	if conditions.Temperature.Degrees != 0 {
		celsius := conditions.Temperature.Degrees
		fahrenheit := convertCelsiusToFahrenheit(celsius)
		m += fmt.Sprintf(". Temperature: %.0f°F / %.0f°C", fahrenheit, celsius)
	}

	if conditions.FeelsLikeTemperature.Degrees != 0 {
		celsius := conditions.FeelsLikeTemperature.Degrees
		fahrenheit := convertCelsiusToFahrenheit(celsius)
		m += fmt.Sprintf(", feels like %.0f°F / %.0f°C", fahrenheit, celsius)
	}

	if conditions.Precipitation.Probability.Percent > 0 {
		if precipitationType, ok := precipitationTypes[conditions.Precipitation.Probability.Type]; ok {
			m += fmt.Sprintf(". Chance of %s: %d%%", strings.ToLower(precipitationType), conditions.Precipitation.Probability.Percent)
		} else {
			m += fmt.Sprintf(". Chance of %s: %d%%", strings.ToLower(strings.Replace(conditions.Precipitation.Probability.Type, "_", " ", -1)), conditions.Precipitation.Probability.Percent)
		}
	}

	if conditions.Wind.Direction.Cardinal != "" {
		if direction, ok := directionCardinalsShort[conditions.Wind.Direction.Cardinal]; ok {
			m += fmt.Sprintf(". Wind: %s at %d %s (%d %s)", direction, convertKilometersToMiles(conditions.Wind.Speed.Value), "mph", conditions.Wind.Speed.Value, "km/h")
		} else {
			m += fmt.Sprintf(". Wind: %s at %d %s (%d %s)", text.Capitalize(strings.Replace(conditions.Wind.Direction.Cardinal, "_", " ", -1), true), convertKilometersToMiles(conditions.Wind.Speed.Value), "mph", conditions.Wind.Speed.Value, "km/h")
		}
	}

	if conditions.RelativeHumidity > 0 {
		m += fmt.Sprintf(". Humidity: %.0f%%", conditions.RelativeHumidity)
	}

	if conditions.UVIndex > 0 {
		m += fmt.Sprintf(". UV index: %d", conditions.UVIndex)
	}

	if len(m) == 0 {
		return ""
	}

	if !strings.HasSuffix(m, ".") {
		m += "."
	}

	return m
}

type forecastDaysResponse struct {
	ForecastDays []struct {
		DaytimeForecast   WeatherConditions
		NighttimeForecast WeatherConditions
		MaxTemperature    struct {
			Degrees float64
			Unit    string
		}
		MinTemperature struct {
			Degrees float64
			Unit    string
		}
		SunEvents struct {
			Sunrise string
			Sunset  string
		}
		MoonEvents struct {
			MoonriseTimes []string
			MoonsetTimes  []string
			MoonPhase     string
		}
	}
}

var moonphases = map[string]string{
	"MOON_PHASE_UNSPECIFIED": "Unknown",
	"NEW_MOON":               "New moon",
	"WAXING_CRESCENT":        "Waxing crescent",
	"FIRST_QUARTER":          "First quarter",
	"WAXING_GIBBOUS":         "Waxing gibbous",
	"FULL_MOON":              "Full moon",
	"WANING_GIBBOUS":         "Waning gibbous",
	"LAST_QUARTER":           "Last quarter",
	"WANING_CRESCENT":        "Waning crescent",
}
