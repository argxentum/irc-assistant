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
	"math"
	"net/http"
	"net/url"
	"strings"
)

const WeatherCommandName = "weather"

const currentConditionsAPIURL = "https://weather.googleapis.com/v1/currentConditions:lookup?location.latitude=%f&location.longitude=%f&key=%s"

type WeatherCommand struct {
	*commandStub
}

func NewWeatherCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &WeatherCommand{
		commandStub: defaultCommandStub(ctx, cfg, ircs),
	}
}

func (c *WeatherCommand) Name() string {
	return WeatherCommandName
}

func (c *WeatherCommand) Description() string {
	return "Shows current weather for the given location."
}

func (c *WeatherCommand) Triggers() []string {
	return []string{"weather", "we"}
}

func (c *WeatherCommand) Usages() []string {
	return []string{"%s <location>"}
}

func (c *WeatherCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *WeatherCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 0)
}

func (c *WeatherCommand) Execute(e *irc.Event) {
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

	currentConditions, err := c.fetchCurrentConditions(lat, lng)
	if err != nil {
		logger.Errorf(e, "failed to fetch current conditions data, %v", err)
		c.Replyf(e, fmt.Sprintf("Error fetching current conditions for %s", style.Bold(location)))
		return
	}

	if currentConditions == nil {
		logger.Errorf(e, "no current conditions data found for %s", location)
		c.Replyf(e, fmt.Sprintf("No current conditions data found for %s", style.Bold(location)))
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

	message := c.createCurrentConditionsMessage(e, currentConditions)
	if len(message) == 0 {
		logger.Errorf(e, "no current conditions message created for %s", location)
		c.Replyf(e, fmt.Sprintf("No current conditions data available for %s", style.Bold(location)))
		return
	}

	c.SendMessage(e, e.ReplyTarget(), fmt.Sprintf("%s: %s", style.Underline(style.Bold(formattedLocation)), message))
}

func (c *WeatherCommand) fetchGeocodingResponse(location string) (*geocodingResponse, error) {
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

func (c *WeatherCommand) fetchCurrentConditions(lat, lng float64) (*currentConditionsResponse, error) {
	res, err := http.Get(fmt.Sprintf(currentConditionsAPIURL, lat, lng, c.cfg.GoogleCloud.MappingAPIKey))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch current conditions data, %v", err)
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch current conditions data, received status code %d", res.StatusCode)
	}

	defer res.Body.Close()

	var response currentConditionsResponse
	err = json.NewDecoder(res.Body).Decode(&response)
	return &response, err
}

func (c *WeatherCommand) createCurrentConditionsMessage(e *irc.Event, conditions *currentConditionsResponse) string {
	m := "Currently "

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

	/*
		if conditions.CurrentConditionsHistory.TemperatureChange.Degrees != 0 {
			change := conditions.CurrentConditionsHistory.TemperatureChange.Degrees
			if change < 0 {
				m += fmt.Sprintf(" (colder than yesterday by %.0f°F / %.0f°C)", convertCentigradeToFahrenheitDegrees(-change), -change)
			} else if change > 0 {
				m += fmt.Sprintf(" (warmer than yesterday by %.0f°F / %.0f°C)", convertCentigradeToFahrenheitDegrees(change), change)
			}
		}
	*/

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

	if !strings.HasSuffix(m, ".") {
		m += "."
	}

	return m
}

func convertCelsiusToFahrenheit(celsius float64) float64 {
	return (celsius * 9 / 5) + 32
}

func convertCentigradeToFahrenheitDegrees(celsius float64) float64 {
	fahrenheitDegreesPerCentigrade := 1.8
	return math.Round(celsius * fahrenheitDegreesPerCentigrade)
}

func convertKilometersToMiles(kilometersPerHour int) int {
	return int(math.Round(float64(kilometersPerHour) * 0.621371))
}

type currentConditionsResponse struct {
	WeatherCondition struct {
		Type        string
		Description struct {
			Text string
		}
	}
	Temperature struct {
		Degrees float64
	}
	FeelsLikeTemperature struct {
		Degrees float64
	}
	Precipitation struct {
		Probability struct {
			Percent int
			Type    string
		}
	}
	CurrentConditionsHistory struct {
		TemperatureChange struct {
			Degrees float64
		}
	}
	Wind struct {
		Direction struct {
			Cardinal string
		}
		Speed struct {
			Value int
			Unit  string
		}
	}
	UVIndex                 int
	ThunderstormProbability int
	CloudCover              int
	RelativeHumidity        float64
}

var weatherConditionTypes = map[string]string{
	"CLEAR":                   "Clear",
	"MOSTLY_CLEAR":            "Mostly clear",
	"PARTLY_CLOUDY":           "Partly cloudy",
	"MOSTLY_CLOUDY":           "Mostly cloudy",
	"CLOUDY":                  "Cloudy",
	"WINDY":                   "Windy",
	"WIND_AND_RAIN":           "Wind and rain",
	"LIGHT_RAIN_SHOWERS":      "Light rain showers",
	"CHANCE_OF_SHOWERS":       "Chance of showers",
	"SCATTERED_SHOWERS":       "Scattered showers",
	"RAIN_SHOWERS":            "Rain showers",
	"HEAVY_RAIN_SHOWERS":      "Heavy rain showers",
	"LIGHT_TO_MODERATE_RAIN":  "Light to moderate rain",
	"MODERATE_TO_HEAVY_RAIN":  "Moderate to heavy rain",
	"RAIN":                    "Rain",
	"LIGHT_RAIN":              "Light rain",
	"HEAVY_RAIN":              "Heavy rain",
	"RAIN_PERIODICALLY_HEAVY": "Rain, periodically heavy",
	"LIGHT_SNOW_SHOWERS":      "Light snow showers",
	"CHANCE_OF_SNOW_SHOWERS":  "Chance of snow showers",
	"SCATTERED_SNOW_SHOWERS":  "Scattered snow showers",
	"SNOW_SHOWERS":            "Snow showers",
	"HEAVY_SNOW_SHOWERS":      "Heavy snow showers",
	"LIGHT_TO_MODERATE_SNOW":  "Light to moderate snow",
	"MODERATE_TO_HEAVY_SNOW":  "Moderate to heavy snow",
	"SNOW":                    "Snow",
	"LIGHT_SNOW":              "Light snow",
	"HEAVY_SNOW":              "Heavy snow",
	"SNOWSTORM":               "Snow, with possible thunder and lightning",
	"SNOW_PERIODICALLY_HEAVY": "Snow, periodically heavy",
	"HEAVY_SNOW_STORM":        "Heavy snow, with possible thunder and lightning",
	"BLOWING_SNOW":            "Blowing snow",
	"RAIN_AND_SNOW":           "Rain and snow mix",
	"HAIL":                    "Hail",
	"HAIL_SHOWERS":            "Hail showers",
	"THUNDERSTORM":            "Thunderstorms",
	"THUNDERSHOWER":           "Rain showers, with possible thunder and lightning",
	"LIGHT_THUNDERSTORM_RAIN": "Light rain, with possible thunder and lightning",
	"SCATTERED_THUNDERSTORMS": "Scattered thunderstorms",
	"HEAVY_THUNDERSTORM":      "Heavy thunderstorms",
}

var precipitationTypes = map[string]string{
	"PRECIPITATION_TYPE_UNSPECIFIED": "Unspecified",
	"NONE":                           "None",
	"SNOW":                           "Snow",
	"RAIN":                           "Rain",
	"LIGHT_RAIN":                     "Light rain",
	"HEAVY_RAIN":                     "Heavy rain",
	"RAIN_AND_SNOW":                  "Rain and snow",
	"SLEET":                          "Sleet",
	"FREEZING_RAIN":                  "Freezing rain",
}

var directionCardinals = map[string]string{
	"CARDINAL_DIRECTION_UNSPECIFIED": "Unspecified",
	"NORTH":                          "North",
	"NORTH_NORTHEAST":                "North-Northeast",
	"NORTHEAST":                      "Northeast",
	"EAST_NORTHEAST":                 "East-Northeast",
	"EAST":                           "East",
	"EAST_SOUTHEAST":                 "East-Southeast",
	"SOUTHEAST":                      "Southeast",
	"SOUTH_SOUTHEAST":                "South-Southeast",
	"SOUTH":                          "South",
	"SOUTH_SOUTHWEST":                "South-Southwest",
	"SOUTHWEST":                      "Southwest",
	"WEST_SOUTHWEST":                 "West-Southwest",
	"WEST":                           "West",
	"WEST_NORTHWEST":                 "West-Northwest",
	"NORTHWEST":                      "Northwest",
	"NORTH_NORTHWEST":                "North-Northwest",
}

var directionCardinalsShort = map[string]string{
	"CARDINAL_DIRECTION_UNSPECIFIED": "N/A",
	"NORTH":                          "N",
	"NORTH_NORTHEAST":                "NNE",
	"NORTHEAST":                      "NE",
	"EAST_NORTHEAST":                 "ENE",
	"EAST":                           "E",
	"EAST_SOUTHEAST":                 "ESE",
	"SOUTHEAST":                      "SE",
	"SOUTH_SOUTHEAST":                "SSE",
	"SOUTH":                          "S",
	"SOUTH_SOUTHWEST":                "SSW",
	"SOUTHWEST":                      "SW",
	"WEST_SOUTHWEST":                 "WSW",
	"WEST":                           "W",
	"WEST_NORTHWEST":                 "WNW",
	"NORTHWEST":                      "NW",
	"NORTH_NORTHWEST":                "NNW",
}

var speedUnits = map[string]string{
	"SPEED_UNIT_UNSPECIFIED": "Unspecified",
	"KILOMETERS_PER_HOUR":    "Kilometers per hour",
	"MILES_PER_HOUR":         "Miles per hour",
}

var speedUnitsShort = map[string]string{
	"SPEED_UNIT_UNSPECIFIED": "N/A",
	"KILOMETERS_PER_HOUR":    "km/h",
	"MILES_PER_HOUR":         "mph",
}
