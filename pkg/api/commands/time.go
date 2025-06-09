package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/style"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const TimeCommandName = "time"

const geocodingAPIURL = "https://maps.googleapis.com/maps/api/geocode/json?address=%s&key=%s"
const timeZoneAPIURL = "https://maps.googleapis.com/maps/api/timezone/json?location=%f,%f&timestamp=%d&key=%s"

type TimeCommand struct {
	*commandStub
}

func NewTimeCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &TimeCommand{
		commandStub: defaultCommandStub(ctx, cfg, ircs),
	}
}

func (c *TimeCommand) Name() string {
	return TimeCommandName
}

func (c *TimeCommand) Description() string {
	return "Shows date and time for the given location."
}

func (c *TimeCommand) Triggers() []string {
	return []string{"time", "date"}
}

func (c *TimeCommand) Usages() []string {
	return []string{"%s <location>"}
}

func (c *TimeCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *TimeCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 1)
}

func (c *TimeCommand) Execute(e *irc.Event) {
	logger := log.Logger()
	tokens := Tokens(e.Message())
	location := strings.Join(tokens[1:], " ")
	logger.Infof(e, "⚡ %s [%s/%s] %s", c.Name(), e.From, e.ReplyTarget(), location)

	geocoding, err := c.fetchGeocodingResponse(location)
	if err != nil {
		logger.Errorf(e, "failed to fetch geocoding data: %v", err)
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

	tz, err := c.fetchTimeZoneResponse(lat, lng)
	if err != nil {
		logger.Errorf(e, "failed to fetch timezone data: %v", err)
		c.Replyf(e, fmt.Sprintf("Error fetching timezone data for %s", style.Bold(formattedLocation)))
		return
	}

	formattedTime, err := c.formatDateTimeResponse(formattedLocation, tz)
	if err != nil {
		logger.Errorf(e, "failed to format date/time response: %v", err)
		c.Replyf(e, fmt.Sprintf("Error parsing date/time for %s", style.Bold(formattedLocation)))
		return
	}

	c.SendMessage(e, e.ReplyTarget(), formattedTime)
}

func (c *TimeCommand) fetchGeocodingResponse(location string) (*GeocodingResponse, error) {
	res, err := http.Get(fmt.Sprintf(geocodingAPIURL, location, c.cfg.GoogleCloud.MappingAPIKey))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch geocoding data, %v", err)
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch geocoding data, received status code %d", res.StatusCode)
	}

	defer res.Body.Close()

	var response GeocodingResponse
	err = json.NewDecoder(res.Body).Decode(&response)
	return &response, err
}

func (c *TimeCommand) fetchTimeZoneResponse(lat, lng float64) (*TimeZoneResponse, error) {
	res, err := http.Get(fmt.Sprintf(timeZoneAPIURL, lat, lng, time.Now().Unix(), c.cfg.GoogleCloud.MappingAPIKey))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch timezone data, %v", err)
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch timezone data, received status code %d", res.StatusCode)
	}

	defer res.Body.Close()

	var response TimeZoneResponse
	err = json.NewDecoder(res.Body).Decode(&response)
	return &response, err
}

func (c *TimeCommand) formatDateTimeResponse(formattedLocation string, tz *TimeZoneResponse) (string, error) {
	if tz == nil {
		return "", fmt.Errorf("timezone data not found for location: %s", formattedLocation)
	}

	currentTime := time.Now().In(time.FixedZone(tz.TimezoneName, tz.RawOffset+tz.DstOffset))
	return currentTime.Format(fmt.Sprintf("%s: %s on %s", style.Underline(formattedLocation), style.Bold("3:04 PM"), style.Bold("Monday, January 2, 2006"))), nil
}

type GeocodingResponse struct {
	Results []struct {
		FormattedAddress string `json:"formatted_address"`
		Geometry         struct {
			Location struct {
				Lat float64 `json:"lat"`
				Lng float64 `json:"lng"`
			} `json:"location"`
		} `json:"geometry"`
	} `json:"results"`
}

type TimeZoneResponse struct {
	DstOffset    int    `json:"dstOffset"`
	RawOffset    int    `json:"rawOffset"`
	Status       string `json:"status"`
	TimezoneID   string `json:"timeZoneId"`
	TimezoneName string `json:"timeZoneName"`
}
