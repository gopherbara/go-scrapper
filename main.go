package main

import (
	"encoding/csv"
	"fmt"
	"github.com/gocolly/colly"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

type Job struct {
	Name          string
	Description   string
	Salary        string
	Company       string
	ContactPerson string
	Location      string
	Experience    string
	EnglishLvl    string
	JobType       string // remote/office
	CompanyType   string
}

type ConcurrentJob struct {
	sync.RWMutex
	Items []Job
}

func (cj *ConcurrentJob) AppendConcurrent(job Job) {
	cj.Lock()
	defer cj.Unlock()
	cj.Items = append(cj.Items, job)
}

func main() {
	jobs := &ConcurrentJob{
		Items: make([]Job, 0),
	}
	wg := sync.WaitGroup{}
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go ScrapDjinni(&wg, jobs, i)
	}
	wg.Wait()
	SaveToFile(jobs)
}

func ScrapDjinni(wg *sync.WaitGroup, jobs *ConcurrentJob, pageNum int) {
	defer wg.Done()
	scrapperUrl := fmt.Sprintf("https://djinni.co/jobs/?page=%v", pageNum)

	c := colly.NewCollector(colly.AllowedDomains("djinni.co"))

	c.OnRequest(func(request *colly.Request) {
		fmt.Printf("Visiting %s", request.URL)
	})

	c.OnError(func(response *colly.Response, err error) {
		fmt.Printf("Error while scrraping: %s", err.Error())
	})

	c.OnHTML("ul.list-unstyled", func(e *colly.HTMLElement) {

		e.ForEach("li.list-jobs__item", func(i int, element *colly.HTMLElement) {
			job := Job{}
			job.Description = strings.ReplaceAll(element.ChildText("div.list-jobs__description"), "\n", " ")
			job.Name = element.ChildText("a.profile span")
			job.Salary = element.ChildText("span.public-salary-item")

			companyinfo := strings.SplitN(element.ChildText("div.list-jobs__details__info a"), "\n", 2)
			job.Company = companyinfo[0]
			job.ContactPerson = strings.TrimSpace(strings.ReplaceAll(companyinfo[1], "\n", " "))

			job.Location = strings.ReplaceAll(element.ChildText("span.location-text"), "\n", " ")

			details := element.DOM.Find("nobr").Nodes
			if len(details) > 0 {
				job.EnglishLvl = CustomClear(details[len(details)-1].LastChild.Data)
				job.Experience = CustomClear(details[len(details)-2].LastChild.Data)
				job.JobType = CustomClear(details[len(details)-3].LastChild.Data)
				if len(details) > 3 {
					job.CompanyType = CustomClear(details[len(details)-4].LastChild.Data)
				}
			}
			jobs.AppendConcurrent(job)
		})

	})

	c.Visit(scrapperUrl)
}

func CustomClear(str string) string {
	return strings.TrimSpace(strings.TrimPrefix(str, "\n"))
}

func SaveToFile(jobs *ConcurrentJob) {
	fileName := "Results/" + time.Now().Format("2006-02-01") + "_jobs.csv"
	csvFile, err := os.Create(fileName)

	if err != nil {
		log.Fatalf("failed creating file: %s", err)
	}
	defer csvFile.Close()
	csvwriter := csv.NewWriter(csvFile)
	csvwriter.Write([]string{"Company", "Position", "Salary", "Experience", "English Level", "Description", "Location", "Job Type"})

	for _, job := range jobs.Items {
		row := []string{job.Company, job.Name, job.Salary, job.Experience, job.EnglishLvl, job.Description, job.Location,
			job.JobType}
		_ = csvwriter.Write(row)
	}

	csvwriter.Flush()
}
