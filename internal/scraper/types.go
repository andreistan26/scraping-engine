package scraper

import "regexp"

type Attribute struct {
    //Selector    string      `json:"selector,omitempty"`
    Key         string      `json:"key,omitempty"`
    Value       string      `json:"value,omitempty"`
    RegexMatch  string      `json:"regex,omitempty"`
    Re          *regexp.Regexp
}

type Return struct {
    Key         string      `json:"key,omitempty"`
    Attribute   Attribute   `json:"attribute,omitempty"`
    Type        string      `json:"type,omitempty"`
    Query       string      `json:"query_attribute,omitempty"`
}

type Condition struct {
    Attribute   Attribute   `json:"attribute,omitempty"`
    Comp        string      `json:"comp,omitempty"`
    Comaparand  string      `json:"comparand,omitempty"`
    Type        string      `json:"type,omitempty"`
}

type Item struct {
    Selector    string      `json:"selector,omitempty"`
    Conditions  []Condition `json:"conditions,omitempty"`
    Returns     []Return    `json:"return,omitempty"`
}

type ScraperJobConfig struct {
    Entry   string      `json:"entry,omitempty"`
    Item    Item        `json:"item,omitempty"`
}

type ScraperResults struct {
    Items []map[string]interface{} `json:"items"`
}

type ScraperJob struct {
    Results ScraperResults 
    Config  ScraperJobConfig
}

func NewScraperJob(Config ScraperJobConfig) ScraperJob {
    job := ScraperJob {Config: Config}
    job.Results.Items = []map[string]interface{}{}

    return job
}
