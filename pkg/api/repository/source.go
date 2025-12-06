package repository

import (
	"assistant/pkg/api/style"
	"assistant/pkg/api/text"
	"assistant/pkg/firestore"
	"assistant/pkg/models"
	"assistant/pkg/slicesx"
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strings"
)

var httpRegex = regexp.MustCompile(`^https?://(?:www\.)?(.*?)/`)

func AddSource(source *models.Source) error {
	return firestore.Get().CreateSource(source)
}

func FindSource(input string) (*models.Source, error) {
	input = strings.TrimSpace(strings.ToLower(input))

	src, err := findSourceByDomain(input)
	if err != nil {
		return nil, err
	}

	if src != nil {
		return src, nil
	}

	return nil, nil
}

func FindSourceIncludingKeywords(input string) (*models.Source, error) {
	src, err := FindSource(input)
	if err != nil {
		return nil, err
	}

	if src != nil {
		return src, nil
	}

	keywords := strings.Fields(input)
	if len(keywords) == 0 {
		return nil, nil
	}

	srcs, err := findSourceByKeywords(keywords, true)
	if err != nil {
		return nil, err
	}

	if len(srcs) > 0 {
		return srcs[0], nil
	}

	return nil, nil
}

func findSourceByDomain(url string) (*models.Source, error) {
	domain := url
	if httpRegex.MatchString(url) {
		m := httpRegex.FindStringSubmatch(url)
		if len(m) < 2 {
			return nil, nil
		}
		domain = m[1]
	}

	sources, err := firestore.Get().FindSourcesByDomain(domain)
	if err != nil {
		return nil, err
	}

	if len(sources) == 0 {
		return nil, nil
	}

	return sources[0], nil
}

type sourceResult struct {
	source *models.Source
	score  int
}

func findSourceByKeywords(keywords []string, requireAll bool) ([]*models.Source, error) {
	kw := make([]string, len(keywords))
	for i, k := range keywords {
		kw[i] = strings.TrimSpace(strings.ToLower(k))
	}

	sk, err := firestore.Get().FindSourcesByKeywords(kw)
	if err != nil {
		return nil, err
	}

	if len(sk) == 0 {
		return nil, nil
	}

	sr := make([]sourceResult, 0)
	for _, s := range sk {
		score := 0
		allMatch := true
		for _, k := range kw {
			if slices.Contains(s.Keywords, k) {
				score++
			} else {
				allMatch = false
			}
		}

		if score > 0 && (!requireAll || allMatch) {
			sr = append(sr, sourceResult{s, score})
		}
	}

	sort.Slice(sr, func(i, j int) bool {
		return sr[i].score > sr[j].score
	})

	sources := make([]*models.Source, 0)
	for _, r := range sr {
		sources = append(sources, r.source)
	}

	return sources, nil
}

func FullSourceSummary(source *models.Source) string {
	desc := ""

	highlyBiased := []string{
		"extreme", "conspiracy", "propaganda", "pseudoscience", "far",
	}

	cleanedBias := strings.ReplaceAll(strings.ToLower(source.Bias), "-", " ")
	ratingColor := style.ColorNone
	if strings.Contains(strings.ToLower(cleanedBias), "least biased") {
		ratingColor = style.ColorGreen
	} else if slicesx.ContainsAny(highlyBiased, strings.Fields(cleanedBias)) {
		ratingColor = style.ColorRed
	} else if strings.ToLower(source.Bias) == "left" || strings.ToLower(source.Bias) == "right" {
		ratingColor = style.ColorYellow
	}

	source.Bias = strings.ReplaceAll(source.Bias, "n/a", "N/A")

	if len(source.Bias) > 0 {
		desc += fmt.Sprintf("%s: %s", style.Underline("Bias"), style.ColorForeground(text.CapitalizeEveryWord(source.Bias, false), ratingColor))
	}

	factualColor := style.ColorNone
	if strings.Contains(strings.ToLower(source.Factuality), "high") {
		factualColor = style.ColorGreen
	} else if strings.ToLower(source.Factuality) == "mixed" {
		factualColor = style.ColorYellow
	} else if strings.Contains(strings.ToLower(source.Factuality), "low") {
		factualColor = style.ColorRed
	}

	source.Factuality = strings.ReplaceAll(source.Factuality, "n/a", "N/A")

	if len(source.Factuality) > 0 {
		if len(desc) > 0 {
			desc += ", "
		}
		desc += fmt.Sprintf("%s: %s", style.Underline("Factual reporting"), style.ColorForeground(text.CapitalizeEveryWord(source.Factuality, false), factualColor))
	}

	credibilityColor := style.ColorNone
	if strings.Contains(strings.ToLower(source.Credibility), "high") {
		credibilityColor = style.ColorGreen
	} else if strings.Contains(strings.ToLower(source.Credibility), "medium") {
		credibilityColor = style.ColorYellow
	} else if strings.Contains(strings.ToLower(source.Credibility), "low") {
		credibilityColor = style.ColorRed
	}

	source.Credibility = strings.ReplaceAll(source.Credibility, "n/a", "N/A")

	if len(source.Credibility) > 0 {
		if len(desc) > 0 {
			desc += ", "
		}
		desc += fmt.Sprintf("%s: %s", style.Underline("Credibility"), style.ColorForeground(text.CapitalizeEveryWord(source.Credibility, false), credibilityColor))
	}

	if len(desc) > 0 {
		desc = fmt.Sprintf("📊 %s: %s | %s", style.Bold(source.Title), desc, style.Italics(source.ID))
	}

	return desc
}

func ShortSourceSummary(source *models.Source) string {
	desc := ""

	credibilityColor := style.ColorNone
	if strings.Contains(strings.ToLower(source.Credibility), "high") {
		credibilityColor = style.ColorGreen
	} else if strings.Contains(strings.ToLower(source.Credibility), "medium") {
		credibilityColor = style.ColorYellow
	} else if strings.Contains(strings.ToLower(source.Credibility), "low") {
		credibilityColor = style.ColorRed
	} else if strings.Contains(strings.ToLower(source.Bias), "satire") {
		credibilityColor = style.ColorBlue
		source.Credibility = source.Bias
	}

	source.Credibility = strings.ReplaceAll(source.Credibility, "n/a", "N/A")

	if len(source.Credibility) > 0 {
		rating := text.CapitalizeEveryWord(source.Credibility, false)
		if slices.Contains([]string{"high", "medium", "low"}, strings.ToLower(source.Credibility)) {
			rating += " Credibility"
		}
		desc += fmt.Sprintf("%s", style.ColorForeground(rating, credibilityColor))
	}

	if len(desc) > 0 {
		desc = fmt.Sprintf("📊 %s: %s", style.Bold(source.Title), desc)
	}

	return desc
}
