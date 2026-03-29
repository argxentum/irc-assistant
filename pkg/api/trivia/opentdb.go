package trivia

import (
	"encoding/json"
	"fmt"
	"html"
	"math/rand"
	"net/http"
	"time"
)

const apiURL = "https://opentdb.com/api.php"

type Question struct {
	Question     string   `json:"question"`
	Answers      []string `json:"answers"`
	CorrectIndex int      `json:"correct_index"` // 1-based
	Category     string   `json:"category"`
	Difficulty   string   `json:"difficulty"`
}

type openTDBResponse struct {
	ResponseCode int             `json:"response_code"`
	Results      []openTDBResult `json:"results"`
}

type openTDBResult struct {
	Category         string   `json:"category"`
	Type             string   `json:"type"`
	Difficulty       string   `json:"difficulty"`
	Question         string   `json:"question"`
	CorrectAnswer    string   `json:"correct_answer"`
	IncorrectAnswers []string `json:"incorrect_answers"`
}

func FetchQuestions(count int, category, difficulty string) ([]Question, error) {
	url := fmt.Sprintf("%s?amount=%d&type=multiple", apiURL, count)
	if category != "" && category != "any" {
		url += "&category=" + category
	}
	if difficulty != "" && difficulty != "any" {
		url += "&difficulty=" + difficulty
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error fetching trivia questions: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("opentdb returned status %d", resp.StatusCode)
	}

	var result openTDBResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("error decoding trivia response: %w", err)
	}

	if result.ResponseCode != 0 {
		return nil, fmt.Errorf("opentdb error code %d", result.ResponseCode)
	}

	questions := make([]Question, 0, len(result.Results))
	for _, r := range result.Results {
		q := Question{
			Question:   html.UnescapeString(r.Question),
			Category:   html.UnescapeString(r.Category),
			Difficulty: r.Difficulty,
		}

		correct := html.UnescapeString(r.CorrectAnswer)
		answers := make([]string, 0, len(r.IncorrectAnswers)+1)
		for _, a := range r.IncorrectAnswers {
			answers = append(answers, html.UnescapeString(a))
		}
		answers = append(answers, correct)

		rand.Shuffle(len(answers), func(i, j int) {
			answers[i], answers[j] = answers[j], answers[i]
		})

		for i, a := range answers {
			if a == correct {
				q.CorrectIndex = i + 1
				break
			}
		}
		q.Answers = answers

		questions = append(questions, q)
	}

	return questions, nil
}

// Categories maps OpenTDB category IDs to display names.
var Categories = map[string]string{
	"9":  "General Knowledge",
	"10": "Entertainment: Books",
	"11": "Entertainment: Film",
	"12": "Entertainment: Music",
	"13": "Entertainment: Musicals & Theatres",
	"14": "Entertainment: Television",
	"15": "Entertainment: Video Games",
	"16": "Entertainment: Board Games",
	"17": "Science & Nature",
	"18": "Science: Computers",
	"19": "Science: Mathematics",
	"20": "Mythology",
	"21": "Sports",
	"22": "Geography",
	"23": "History",
	"24": "Politics",
	"25": "Art",
	"26": "Celebrities",
	"27": "Animals",
	"28": "Vehicles",
	"29": "Entertainment: Comics",
	"30": "Science: Gadgets",
	"31": "Entertainment: Japanese Anime & Manga",
	"32": "Entertainment: Cartoon & Animations",
}
