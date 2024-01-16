package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"
)

var baseUrl = "https://jsonplaceholder.typicode.com"
var limit = 100

type Post struct {
	ID     int    `json:"id"`
	UserID int    `json:"userId"`
	Title  string `json:"title"`
	Body   string `json:"body"`
}

type Result struct {
	Success       int           `json:"success"`
	Fail          int           `json:"fail"`
	TotalDuration time.Duration `json:"totalDuration"`
	SuccessIDs    []int         `json:"successIds"`
	FailIDs       []int         `json:"failIds"`
}

type ErrorPost struct {
	ID  int
	Err error
}

func main() {
	args := os.Args
	if args[1] == "" {
		fmt.Println("arguments are missing")
		return
	}

	if args[1] == "--without-concurrent" {
		withoutConcurrent()
		return
	}

	if args[1] == "--with-concurrent" {
		withConcurrent()
		return
	}
}

func withoutConcurrent() {
	fmt.Println("start API calls without concurrent, please wait...")
	start := time.Now()
	result := Result{
		SuccessIDs: make([]int, 0),
		FailIDs:    make([]int, 0),
	}

	for i := 1; i <= limit; i++ {
		_, err := fetchPost(i)
		if err != nil {
			result.Fail++
			result.FailIDs = append(result.FailIDs, i)
			continue
		}

		result.SuccessIDs = append(result.SuccessIDs, i)
		result.Success++
	}

	result.TotalDuration = time.Since(start)

	fmt.Println("API calls complete")
	fmt.Printf("Total success: %d \n", result.Success)
	fmt.Printf("Total fail: %d \n", result.Fail)
	fmt.Printf("Total duration: %s\n", result.TotalDuration.String())
	fmt.Println("Success IDs:")
	fmt.Println(result.SuccessIDs)
	fmt.Println("Fail IDs:")
	fmt.Println(result.FailIDs)
}

func withConcurrent() {
	fmt.Println("start API calls with concurrent, please wait...")
	errChan := make(chan ErrorPost)
	postChan := make(chan Post)
	doneChan := make(chan bool)
	wg := sync.WaitGroup{}
	var count int

	start := time.Now()
	for i := 1; i <= limit; i++ {
		wg.Add(1)
		count++

		go func(wg *sync.WaitGroup, id int) {
			defer wg.Done()

			post, err := fetchPost(id)
			if err != nil {
				errChan <- ErrorPost{
					ID:  id,
					Err: err,
				}
			}

			postChan <- post
		}(&wg, i)
	}

	go func(wg *sync.WaitGroup) {
		wg.Wait()

		if count == limit {
			close(doneChan)
		}
	}(&wg)

	result := Result{
		SuccessIDs: make([]int, 0),
		FailIDs:    make([]int, 0),
	}

	for {
		select {
		case post := <-postChan:
			result.SuccessIDs = append(result.SuccessIDs, post.ID)
			result.Success++
		case err := <-errChan:
			fmt.Println(err.Err)
			result.FailIDs = append(result.FailIDs, err.ID)
			result.Fail++
		case <-doneChan:
			result.TotalDuration = time.Since(start)

			fmt.Println("API calls complete")
			fmt.Printf("Total success: %d \n", result.Success)
			fmt.Printf("Total fail: %d \n", result.Fail)
			fmt.Printf("Total duration: %s\n", result.TotalDuration.String())
			fmt.Println("Success IDs:")
			fmt.Println(result.SuccessIDs)
			fmt.Println("Fail IDs:")
			fmt.Println(result.FailIDs)
			return
		}
	}
}

func fetchPost(id int) (Post, error) {
	httpClient := &http.Client{}

	finalUrl := fmt.Sprintf("%s/posts/%d", baseUrl, id)
	res, err := httpClient.Get(finalUrl)
	if err != nil {
		return Post{}, err
	}
	defer res.Body.Close()

	var post Post
	if err := json.NewDecoder(res.Body).Decode(&post); err != nil {
		return Post{}, err
	}

	return post, nil
}
