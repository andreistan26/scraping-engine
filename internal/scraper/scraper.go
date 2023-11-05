package scraper

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"

	"scraper/internal/types"
	"scraper/internal/log"
	"scraper/internal/utils"

	"github.com/Knetic/govaluate"
	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/debug"
)

func (worker *WorkerConfig) DoJob(ctx context.Context, jobEntry *types.Job) error {
	scraperJobConfig := &ScraperJobConfig{}
	dec := json.NewDecoder(bytes.NewReader(jobEntry.Config))
	if err := dec.Decode(&scraperJobConfig); err != nil {
		log.FromContext(ctx).Debugf("%v", jobEntry.Config)
		utils.Fail(ctx, err, "fail on marshal")
	}

	runningJob := NewScraperJob(scraperJobConfig, jobEntry)

	allowedDomain, err := toAllowWebsiteFormat(runningJob.Config.Entry)
	if err != nil {
		return err
	}

	c := colly.NewCollector(
		colly.AllowedDomains(
			allowedDomain,
		),
		colly.Async(true),
		colly.Debugger(&debug.LogDebugger{}),
	)

	log.FromContext(ctx).Debugf(runningJob.Config.Item.Selector)

	c.OnHTML(runningJob.Config.Item.Selector, func(h *colly.HTMLElement) {
		for _, cond := range runningJob.Config.Item.Conditions {
			res := cond.Evaluate(h)
			if res == false {
				return
			}
		}

		result := make(map[string]interface{})
		runningJob.GetReturns(h, result)

		runningJob.Results.Items = append(runningJob.Results.Items, result)
	})

	c.Visit(runningJob.Config.Entry)
	c.Wait()

	worker.insertResults(ctx, runningJob)

	return nil
}

func toAllowWebsiteFormat(URL string) (string, error) {
	str, err := url.Parse(URL)
	if err != nil {
		return "", err
	}

	return str.Hostname(), nil
}

func (attribute Attribute) InElement(h *colly.HTMLElement) bool {
	return h.ChildAttr(attribute.Key, attribute.Key) == attribute.Value
}

func (attribute Attribute) TextInElement(h *colly.HTMLElement) string {
	text := h.ChildText(attribute.Value)
	if attribute.RegexMatch != "" {
		if attribute.Re == nil {
			attribute.Re = regexp.MustCompile(attribute.RegexMatch)
		}
		idxes := attribute.Re.FindStringIndex(text)
		if idxes != nil {
			text = text[idxes[0]:idxes[1]]
		} else {
			text = ""
		}
	}
	return text
}

func (condition Condition) Evaluate(h *colly.HTMLElement) bool {
	left := condition.Attribute.TextInElement(h)
	if left == "" {
		return false
	}
	right := condition.Comaparand
	switch condition.Type {
	case "numeric":
		exp, err := govaluate.NewEvaluableExpression(fmt.Sprintf("%s%s%s", toNumericFormat(left), condition.Comp, right))
		if err != nil {
			//log.FromContext(ctx("%v", err)
		}
		res, _ := exp.Evaluate(nil)
		return res.(bool)
	default:
		return false
	}
}

func (job ScraperJob) GetReturns(h *colly.HTMLElement, result map[string]interface{}) {
	for _, ret := range job.Config.Item.Returns {
		switch ret.Type {
		case "text":
			result[ret.Key] = ret.Attribute.TextInElement(h)
			break
		case "attribute":
			result[ret.Key] = ret.Attribute.siblingAttrValue(h, ret.Query)
		}
	}
}

/* Untested function */
func (attribute Attribute) siblingAttrValue(h *colly.HTMLElement, attr string) string {
	res := h.Attr(attr)
	if res != "" {
		return res
	}

	h.DOM.Find(attribute.Key).EachWithBreak(func(i int, s *goquery.Selection) bool {
		if val, _ := s.Attr(attribute.Key); val == attribute.Value {
			res, _ = s.Attr(attr)
			return true
		}
		return false
	})

	return res
}

func toNumericFormat(number string) string {
	res := []rune{}
	for _, c := range number {
		if c <= '9' && c >= '0' || c == '.' {
			res = append(res, c)
		}
	}
	return string(res)
}

func (worker *WorkerConfig) insertResults(ctx context.Context, scraperJob *ScraperJob) {
	conn, err := worker.DbPool.Acquire(ctx)
	utils.Fail(ctx, err, "Failed to acquire db connection")
	defer conn.Release()

	for _, result := range scraperJob.Results.Items {
		if _, ok := result["name"]; ok == false {
			log.FromContext(ctx).Debugf("Could not find name for result job")
			continue
		}

		itemData, err := json.Marshal(result)
		utils.Fail(ctx, err, "Failed to marshal result map into json")

		_, err = conn.Exec(ctx, "INSERT INTO items (item_name, item_data, item_job_id) VALUES ($1, $2, $3)",
			result["name"], itemData, scraperJob.JobId)
		utils.Fail(ctx, err, "Could not insert items into db")
	}
}
