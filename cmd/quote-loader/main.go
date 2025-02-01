package main

import (
	"assistant/pkg/api/context"
	"assistant/pkg/config"
	"assistant/pkg/firestore"
	"assistant/pkg/models"
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

func main() {
	panic("Remove panic in main to load again...")

	ctx := context.NewContext()

	channel := ""
	start := 0
	logFilename := ""
	configFilename := ""

	if len(os.Args) > 1 {
		logFilename = os.Args[1]
	}
	if len(os.Args) > 2 {
		configFilename = os.Args[2]
	}
	if len(os.Args) > 3 {
		channel = os.Args[3]
	}
	if len(os.Args) > 4 {
		start, _ = strconv.Atoi(os.Args[3])
	}

	if len(logFilename) == 0 {
		panic("log filename is required")
	}

	if len(configFilename) == 0 {
		panic("config filename is required")
	}

	if len(channel) == 0 {
		panic("channel is required")
	}

	cfg, err := config.ReadConfig(configFilename)
	if err != nil {
		panic(err)
	}

	initializeFirestore(ctx, cfg)
	defer firestore.Get().Close()

	processFile(cfg, logFilename, channel, start)
}

func initializeFirestore(ctx context.Context, cfg *config.Config) {
	_, err := firestore.Initialize(ctx, cfg)
	if err != nil {
		panic(fmt.Errorf("error initializing firestore, %s", err))
	}
}

func readLineCount(logFilename string) int {
	file, err := os.Open(logFilename)
	if err != nil {
		panic(fmt.Errorf("error opening log file, %s", err))
	}
	if file == nil {
		panic(fmt.Errorf("error opening log file, file is nil"))
	}

	defer file.Close()

	reader := bufio.NewReader(file)
	lineCount := 0

	for {
		_, err := reader.ReadString('\n')
		if err == io.EOF {
			break
		} else if err != nil {
			panic(fmt.Errorf("error reading line: %s", err))
		}
		lineCount++
	}

	return lineCount
}

const chunkSize = 300

func processFile(cfg *config.Config, logFilename, channel string, start int) error {
	lines := readLineCount(logFilename)
	chunks := lines / chunkSize

	file, err := os.Open(logFilename)
	if err != nil {
		panic(fmt.Errorf("error opening log file, %s", err))
	}
	if file == nil {
		panic(fmt.Errorf("error opening log file, file is nil"))
	}

	defer file.Close()

	reader := bufio.NewReader(file)
	previousChunk := make([]string, 0, chunkSize)
	currentChunk := make([]string, 0, chunkSize)

	i := 0
	grabs := make([]grab, 0)
	adds := make([]add, 0)
	fs := firestore.Get()

	for {
		line, err := reader.ReadString('\n')

		if err == io.EOF {
			if len(currentChunk) > 0 {
				processChunk(i, chunks, lines, currentChunk, previousChunk, &grabs, &adds)
			}
			break
		} else if err != nil {
			return fmt.Errorf("error reading line: %w", err)
		}

		currentChunk = append(currentChunk, line)

		if len(currentChunk) == chunkSize {
			processChunk(i, chunks, lines, currentChunk, previousChunk, &grabs, &adds)
			previousChunk = currentChunk
			currentChunk = make([]string, 0, chunkSize)
			i++
		}
	}

	for j, g := range grabs {
		fmt.Printf("[%d/%d] %s grabbed message from %s: %s\n", j, len(grabs), g.grabber, g.message.nick, g.message.content)

		if j < start {
			fmt.Printf("Skipping...\n")
			continue
		}

		q := models.NewQuote(g.message.nick, g.grabber, g.message.content, g.message.date)
		if err = fs.CreateQuote(channel, q); err != nil {
			fmt.Println("Error creating quote")
		}
	}

	return nil
}

var messageRegex = regexp.MustCompile(`^\[(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2})] <(.+)> (.+)\s*$`)
var grabRegex = regexp.MustCompile(`^\.grab (.+)$`)
var quoteAddRegex = regexp.MustCompile(`^\.q add (.+) (.+)$`)

type message struct {
	date    time.Time
	nick    string
	content string
}

type grab struct {
	grabber string
	message message
}

type add struct {
	grabber string
	message message
}

var loc, _ = time.LoadLocation("America/Chicago")
var ignoredNicks = []string{"ChanServ", "NickServ", "gonzobot"}

func processChunk(i, chunks, lines int, currentChunk []string, previousChunk []string, grabs *[]grab, adds *[]add) {
	messages := make([]message, 0)

	j := 0

	for _, line := range currentChunk {
		line = strings.TrimSpace(line)
		var m message

		if messageRegex.MatchString(line) {
			matches := messageRegex.FindStringSubmatch(line)
			if len(matches) != 4 {
				fmt.Println("Error parsing message")
				j++
				continue
			}

			if slices.Contains(ignoredNicks, matches[2]) {
				j++
				continue
			}

			d, err := time.ParseInLocation("2006-01-02 15:04:05", matches[1], loc)
			if err != nil {
				fmt.Println("Error parsing date")
				j++
				continue
			}

			m = message{
				date:    d,
				nick:    matches[2],
				content: matches[3],
			}

			messages = append(messages, m)
		}

		if grabRegex.MatchString(m.content) {
			matches := grabRegex.FindStringSubmatch(m.content)
			if len(matches) != 2 {
				fmt.Println("Error parsing grab")
				j++
				continue
			}

			author := matches[1]
			if author == m.nick {
				j++
				continue
			}

			g := grab{
				grabber: m.nick,
			}

			for k := len(messages) - 1; k >= 0; k-- {
				if messages[k].nick == author {
					g.message = messages[k]
					break
				}
			}

			if len(g.message.content) == 0 {
				if pm, ok := searchChunkForMessageFrom(previousChunk, author); ok {
					g.message = pm
				}
			}

			if len(g.message.content) > 0 {
				fmt.Printf("[%d/%d] %s parsed message from %s: %s\n", 1+j+(i*chunkSize), lines, g.grabber, g.message.nick, g.message.content)
				*grabs = append(*grabs, g)
			}
		}

		//if quoteAddRegex.MatchString(m.content) {
		//	matches := quoteAddRegex.FindStringSubmatch(m.content)
		//	if len(matches) != 3 {
		//		fmt.Println("Error parsing quote add")
		//		j++
		//		continue
		//	}
		//
		//	author := matches[1]
		//	if author == m.nick {
		//		j++
		//		continue
		//	}
		//
		//	a := add{
		//		grabber: m.nick,
		//	}
		//
		//	if len(a.message.content) > 0 {
		//		*adds = append(*adds, a)
		//	}
		//}

		j++
	}
}

func searchChunkForMessageFrom(chunk []string, author string) (message, bool) {
	for i := len(chunk) - 1; i >= 0; i-- {
		if messageRegex.MatchString(chunk[i]) {
			matches := messageRegex.FindStringSubmatch(chunk[i])
			if len(matches) != 4 {
				fmt.Println("Error parsing message")
				continue
			}

			d, err := time.ParseInLocation("2006-01-02 15:04:05", matches[1], loc)
			if err != nil {
				fmt.Println("Error parsing date")
				continue
			}

			m := message{
				date:    d,
				nick:    matches[2],
				content: matches[3],
			}

			if slices.Contains(ignoredNicks, m.nick) {
				continue
			}

			if m.nick == author {
				return m, true
			}
		}
	}

	return message{}, false
}
