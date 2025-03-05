package commands

import (
	"assistant/pkg/api/context"
	"assistant/pkg/api/irc"
	"assistant/pkg/api/style"
	"assistant/pkg/config"
	"assistant/pkg/log"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"sort"
	"strings"
)

const MemeCommandName = "meme"

type MemeCommand struct {
	*commandStub
}

func NewMemeCommand(ctx context.Context, cfg *config.Config, ircs irc.IRC) Command {
	return &MemeCommand{
		commandStub: defaultCommandStub(ctx, cfg, ircs),
	}
}

func (c *MemeCommand) Name() string {
	return MemeCommandName
}

func (c *MemeCommand) Description() string {
	return "Generates meme images."
}

func (c *MemeCommand) Triggers() []string {
	return []string{"meme"}
}

func (c *MemeCommand) Usages() []string {
	return []string{"%s <meme> text:<title> [text:<subtitle>]"}
}

func (c *MemeCommand) AllowedInPrivateMessages() bool {
	return true
}

func (c *MemeCommand) CanExecute(e *irc.Event) bool {
	return c.isCommandEventValid(c, e, 2)
}

type memesResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Memes []meme `json:"memes"`
	} `json:"data"`
	ErrorMessage string `json:"error_message"`
}

type meme struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	URL      string `json:"url"`
	Captions int    `json:"captions"`
}

type captionImageResponse struct {
	Success bool `json:"success"`
	Data    struct {
		URL string `json:"url"`
	} `json:"data"`
}

var memeRegex = regexp.MustCompile(`^(.*?)\s*(?:text|t|t1|top):(.*?)(?:\s*(?:text|t|t2|bottom|b):(.*))?$`)

func (c *MemeCommand) Execute(e *irc.Event) {
	logger := log.Logger()
	logger.Infof(e, "⚡ %s [%s/%s] ", c.Name(), e.From, e.ReplyTarget())
	tokens := Tokens(e.Message())
	input := strings.TrimSpace(strings.TrimPrefix(e.Message(), tokens[0]))

	matches := memeRegex.FindStringSubmatch(input)
	if matches == nil || len(matches) < 3 {
		logger.Debugf(e, "invalid syntax, matches: %v", matches)
		usage := fmt.Sprintf(c.Usages()[0], c.Triggers()[0])
		c.Replyf(e, "Invalid syntax, usage: %s", style.Italics(usage))
		return
	}

	query := strings.TrimSpace(strings.ToLower(matches[1]))
	queryTokens := strings.Split(query, " ")
	top := matches[2]
	bottom := ""
	if len(matches) > 3 {
		bottom = matches[3]
	}

	sres, err := http.Get("https://api.imgflip.com/get_memes")
	if err != nil {
		c.Replyf(e, "Sorry, something went wrong. Please try again later.")
		logger.Errorf(e, "failed to search memes: %v", err)
		return
	}

	defer sres.Body.Close()

	sresb, err := io.ReadAll(sres.Body)
	if err != nil {
		c.Replyf(e, "Sorry, something went wrong. Please try again later.")
		logger.Errorf(e, "failed to read meme search response: %v", err)
		return
	}

	var memesResult memesResponse
	if err = json.Unmarshal(sresb, &memesResult); err != nil {
		c.Replyf(e, "Sorry, something went wrong. Please try again later.")
		logger.Errorf(e, "failed to decode memes response: %v", err)
		return
	}

	if !memesResult.Success {
		c.Replyf(e, "Sorry, something went wrong. Please try again later.")
		logger.Error(e, "memes search returned failure")
		return
	}

	if len(memesResult.Data.Memes) == 0 {
		c.Replyf(e, "Sorry, no memes found matching %s.", style.Bold(query))
		return
	}

	m, err := rankMemeResults(memesResult.Data.Memes, queryTokens)
	if err != nil {
		c.Replyf(e, "Sorry, something went wrong. Please try again later.")
		logger.Errorf(e, "failed to rank meme results: %v", err)
		return
	}

	if m == nil {
		c.Replyf(e, "Sorry, no memes found matching %s.", style.Bold(query))
		return
	}

	sd := url.Values{}
	sd.Set("template_id", m.ID)
	sd.Set("username", c.cfg.Imgflip.Username)
	sd.Set("password", c.cfg.Imgflip.Password)
	sd.Set("text0", top)
	sd.Set("text1", bottom)

	creq, err := http.NewRequest(http.MethodPost, "https://api.imgflip.com/caption_image", strings.NewReader(sd.Encode()))
	if err != nil {
		c.Replyf(e, "Sorry, something went wrong. Please try again later.")
		logger.Errorf(e, "failed to create meme caption request: %v", err)
		return
	}
	creq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	cres, err := http.DefaultClient.Do(creq)
	if err != nil {
		c.Replyf(e, "Sorry, something went wrong. Please try again later.")
		logger.Errorf(e, "failed to caption meme: %v", err)
		return
	}

	defer cres.Body.Close()

	cresb, err := io.ReadAll(cres.Body)
	if err != nil {
		c.Replyf(e, "Sorry, something went wrong. Please try again later.")
		logger.Errorf(e, "failed to read caption response: %v", err)
		return
	}

	var captionResult captionImageResponse
	if err = json.Unmarshal(cresb, &captionResult); err != nil {
		c.Replyf(e, "Sorry, something went wrong. Please try again later.")
		logger.Errorf(e, "failed to decode caption response: %v", err)
		return
	}

	if !captionResult.Success {
		c.Replyf(e, "Sorry, something went wrong. Please try again later.")
		logger.Error(e, "meme caption returned failure")
		return
	}

	logger.Debugf(e, "meme caption response: %v", captionResult)
	c.SendMessage(e, e.ReplyTarget(), captionResult.Data.URL)
}

type memeSearchResult struct {
	score int
	meme  meme
}

func rankMemeResults(in []meme, keywords []string) (*meme, error) {
	topMatches := make([]meme, 0)
	all := append(popular, in...)

	sr := make([]memeSearchResult, 0)
	for _, m := range all {
		nameTokens := strings.Split(strings.ToLower(m.Name), " ")
		score := 0
		allMatch := true
		for _, k := range keywords {
			if slices.Contains(nameTokens, strings.ToLower(k)) {
				score++
			} else {
				allMatch = false
			}
		}

		if allMatch {
			topMatches = append(topMatches, m)
		}

		if score > 0 {
			sr = append(sr, memeSearchResult{score, m})
		}
	}

	if len(topMatches) > 0 {
		return &topMatches[0], nil
	}

	sort.Slice(sr, func(i, j int) bool {
		return sr[i].score > sr[j].score
	})

	memes := make([]meme, 0)
	for _, r := range sr {
		memes = append(memes, r.meme)
	}

	if len(memes) == 0 {
		return nil, nil
	}

	return &memes[0], nil
}

// https://imgflip.com/popular-meme-ids
var popular = []meme{
	{ID: "181913649", Name: "Drake Hotline Bling"},
	{ID: "112126428", Name: "Distracted Boyfriend"},
	{ID: "87743020", Name: "Two Buttons"},
	{ID: "124822590", Name: "Left Exit 12 Off Ramp"},
	{ID: "129242436", Name: "Change My Mind"},
	{ID: "438680", Name: "Batman Slapping Robin"},
	{ID: "217743513", Name: "UNO Draw 25 Cards"},
	{ID: "131087935", Name: "Running Away Balloon"},
	{ID: "61579", Name: "One Does Not Simply"},
	{ID: "93895088", Name: "Expanding Brain"},
	{ID: "4087833", Name: "Waiting Skeleton"},
	{ID: "102156234", Name: "Mocking Spongebob"},
	{ID: "1035805", Name: "Boardroom Meeting Suggestion"},
	{ID: "97984", Name: "Disaster Girl"},
	{ID: "188390779", Name: "Woman Yelling At Cat"},
	{ID: "91538330", Name: "Everywhere"},
	{ID: "101470", Name: "Ancient Aliens"},
	{ID: "247375501", Name: "Buff Doge vs Cheems"},
	{ID: "131940431", Name: "Gru's Plan"},
	{ID: "89370399", Name: "Roll Safe Think About It"},
	{ID: "222403160", Name: "Bernie I Am Once Again Asking For Your Support"},
	{ID: "119139145", Name: "Blank Nut Button"},
	{ID: "61520", Name: "Futurama Fry"},
	{ID: "178591752", Name: "Tuxedo Winnie The Pooh"},
	{ID: "155067746", Name: "Surprised Pikachu"},
	{ID: "114585149", Name: "Inhaling Seagull"},
	{ID: "5496396", Name: "Leonardo Dicaprio Cheers"},
	{ID: "135256802", Name: "Epic Handshake"},
	{ID: "27813981", Name: "Hide the Pain Harold"},
	{ID: "80707627", Name: "Sad Pablo Escobar"},
	{ID: "123999232", Name: "The Scroll Of Truth"},
	{ID: "100777631", Name: "Is This A Pigeon"},
	{ID: "21735", Name: "The Rock Driving"},
	{ID: "61532", Name: "The Most Interesting Man In The World"},
	{ID: "148909805", Name: "Monkey Puppet"},
	{ID: "226297822", Name: "Panik Kalm Panik"},
	{ID: "124055727", Name: "Y'all Got Any More Of That"},
	{ID: "28251713", Name: "Oprah You Get A"},
	{ID: "252600902", Name: "Always Has Been"},
	{ID: "161865971", Name: "Marked Safe From"},
	{ID: "8072285", Name: "Doge"},
	{ID: "61585", Name: "Bad Luck Brian"},
	{ID: "101288", Name: "Third World Skeptical Kid"},
	{ID: "134797956", Name: "American Chopper Argument"},
	{ID: "61539", Name: "First World Problems"},
	{ID: "91545132", Name: "Trump Bill Signing"},
	{ID: "180190441", Name: "They're The Same Picture"},
	{ID: "110163934", Name: "I Bet He's Thinking About Other Women"},
	{ID: "61556", Name: "Grandma Finds The Internet"},
	{ID: "6235864", Name: "Finding Neverland"},
	{ID: "175540452", Name: "Unsettled Tom"},
	{ID: "84341851", Name: "Evil Kermit"},
	{ID: "61527", Name: "Y U No"},
	{ID: "3218037", Name: "This Is Where I'd Put My Trophy If I Had One"},
	{ID: "61544", Name: "Success Kid"},
	{ID: "55311130", Name: "This Is Fine"},
	{ID: "14371066", Name: "Star Wars Yoda"},
	{ID: "563423", Name: "That Would Be Great"},
	{ID: "135678846", Name: "Who Killed Hannibal"},
	{ID: "61546", Name: "Brace Yourselves is Coming"},
	{ID: "79132341", Name: "Bike Fall"},
	{ID: "196652226", Name: "Spongebob Ight Imma Head Out"},
	{ID: "405658", Name: "Grumpy Cat"},
	{ID: "61582", Name: "Creepy Condescending Wonka"},
	{ID: "16464531", Name: "But That's None Of My Business"},
	{ID: "61533", Name: "All The Things"},
	{ID: "101511", Name: "Don't You Squidward"},
	{ID: "195515965", Name: "Clown Applying Makeup"},
	{ID: "1509839", Name: "Captain Picard Facepalm"},
	{ID: "101287", Name: "Third World Success Kid"},
	{ID: "235589", Name: "Evil Toddler"},
	{ID: "99683372", Name: "Sleeping Shaq"},
	{ID: "61516", Name: "Philosoraptor"},
	{ID: "100947", Name: "Matrix Morpheus"},
	{ID: "259237855", Name: "Laughing Leo"},
	{ID: "14230520", Name: "Black Girl Wat"},
	{ID: "132769734", Name: "Hard To Swallow Pills"},
	{ID: "245898", Name: "Picard Wtf"},
	{ID: "922147", Name: "Laughing Men In Suits"},
	{ID: "101910402", Name: "Who Would Win?"},
	{ID: "101716", Name: "Yo Dawg Heard You"},
	{ID: "61580", Name: "Too Damn High"},
	{ID: "101440", Name: "10 Guy"},
	{ID: "40945639", Name: "Dr Evil Laser"},
	{ID: "109765", Name: "I'll Just Wait Here"},
	{ID: "259680", Name: "Am I The Only One Around Here"},
	{ID: "9440985", Name: "Face You Make Robert Downey Jr"},
	{ID: "61581", Name: "Put It Somewhere Else Patrick"},
	{ID: "29617627", Name: "Look At Me"},
	{ID: "163573", Name: "Imagination Spongebob"},
	{ID: "56225174", Name: "Be Like Bill"},
	{ID: "12403754", Name: "Bad Pun Dog"},
	{ID: "21604248", Name: "Mugatu So Hot Right Now"},
	{ID: "460541", Name: "Jack Sparrow Being Chased"},
	{ID: "1367068", Name: "I Should Buy A Boat Cat"},
	{ID: "195389", Name: "Sparta Leonidas"},
	{ID: "6531067", Name: "See Nobody Cares"},
	{ID: "766986", Name: "Aaaaand Its Gone"},
	{ID: "444501", Name: "Maury Lie Detector"},
	{ID: "100955", Name: "Confession Bear"},
}
