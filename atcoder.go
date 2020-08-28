package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

type AtCoderSubmission struct {
	Time      int64  `json:"epoch_second"`
	ProblemID string `json:"problem_id"`
	ContestID string `json:"contest_id"`
	Result    string `json:"result"`
}

type ProblemID struct {
	problemID string
	contestID string
}

func getAtCoderUserSubmissions(userID string) ([]AtCoderSubmission, error) {
	var err error
	var submissions []AtCoderSubmission
	url := "https://kenkoooo.com/atcoder/atcoder-api/results?user=" + userID
	res, err := http.Get(url)
	log.Println("Taking a break...")
	time.Sleep(3 * time.Second)
	if err != nil {
		return submissions, err
	}
	bytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return submissions, err
	}
	err = json.Unmarshal(bytes, &submissions)
	if err != nil {
		return submissions, err
	}
	return submissions, nil
}

func filterAtCoderSubmissionsByUniqueAC(submissions []AtCoderSubmission) map[ProblemID]int64 {
	filteredSubmissions := make(map[ProblemID]int64)
	for _, submission := range submissions {
		if submission.Result == "AC" {
			key := ProblemID{submission.ProblemID, submission.ContestID}
			submissionTime, exist := filteredSubmissions[key]
			if exist {
				if submission.Time < submissionTime {
					filteredSubmissions[key] = submission.Time
				}
			} else {
				filteredSubmissions[key] = submission.Time
			}
		}
	}
	return filteredSubmissions
}

func filterAtCoderSubmissionsByDate(submissions map[ProblemID]int64, epochSeconds int64) map[ProblemID]int64 {
	filteredSubmissions := make(map[ProblemID]int64)
	for problemID, seconds := range submissions {
		if seconds >= epochSeconds {
			filteredSubmissions[problemID] = seconds
		}
	}
	return filteredSubmissions
}
