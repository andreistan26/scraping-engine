package scraper

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"regexp"
	"scraper/internal/types"
	"scraper/internal/utils"

	"github.com/Knetic/govaluate"
	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/debug"
)

/*
   The scraper gets a json file as an input and then it does its job.
*/
func DoJob(rawJob types.Job) error {
    scraperJobConfig := ScraperJobConfig{}
    dec := json.NewDecoder(bytes.NewReader(rawJob.Config))
    if err := dec.Decode(&scraperJobConfig); err != nil {
        fmt.Printf("%v", rawJob.Config)
        utils.Fail(err, "fail on marshal")
    }

    job := NewScraperJob(scraperJobConfig)

    allowedDomain, err := toAllowWebsiteFormat(job.Config.Entry)
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

    log.Println(job.Config.Item.Selector)

    c.OnHTML(job.Config.Item.Selector, func(h *colly.HTMLElement) {
        for _, cond := range job.Config.Item.Conditions {
            res := cond.Evaluate(h)
            if res == false {
                return 
            }
        }

        result := make(map[string]interface{})
        job.GetReturns(h, result)

        job.Results.Items = append(job.Results.Items, result)
    })

    c.Visit(job.Config.Entry) 
    c.Wait()

    fmt.Printf("%v\n", job.Results.Items)

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
            log.Printf("%v", err)
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
