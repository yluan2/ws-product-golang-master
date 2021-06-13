package main

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type counters struct {
	sync.Mutex
	view  int
	click int
}

var (
	c           = counters{}
	content     = []string{"sports", "entertainment", "business", "education"}
	mStore      = map[string]counters{}
	tStore      = map[string]*counters{} //avoid creating multiple copies of map
	tempStorage = []string{}
	limiter     = rate.NewLimiter(1, 3)
)

func welcomeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Welcome to EQ Works 😎")
}

func viewHandler(w http.ResponseWriter, r *http.Request) {
	data := content[rand.Intn(len(content))]
	dt := time.Now().Format("01-02-2006 15:04:05")
	newKey := data + ":" + dt //new key with time stamp
	// fmt.Println(newKey)

	c.Lock()
	if !keyExist(data, tStore) {
		tStore[data] = &counters{
			view:  0,
			click: 0,
		}
	}
	tStore[data].view++
	c.view++
	c.Unlock()

	err := processRequest(r)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(400)
		return
	}

	// simulate random click call
	if rand.Intn(100) < 50 {
		processClick(data)
	}

	keyInfo := fmt.Sprintf("%s;%d;%d", newKey, tStore[data].view, tStore[data].click)
	tempStorage = append(tempStorage, keyInfo)

	//output the new Time-Stamped Key and corresponding view, click count
	fmt.Println(newKey, "view time", tStore[data].view, "click time", tStore[data].click)

}

func keyExist(data string, m map[string]*counters) bool {
	for k := range m {
		if data == strings.Split(k, ":")[0] {
			return true
		}
	}
	return false
}

func processRequest(r *http.Request) error {
	time.Sleep(time.Duration(rand.Int31n(50)) * time.Millisecond)
	return nil
}

func processClick(data string) error {
	c.Lock()
	tStore[data].click++
	c.click++
	c.Unlock()

	return nil
}

func statsHandler(w http.ResponseWriter, r *http.Request) {
	if !isAllowed() {
		w.WriteHeader(429)
		return
	}

	c := make(chan bool)

	// when no click was created in 5s, the auto uploaded will be stopped
	for len(tempStorage) != 0 {
		go uploadCounters(tempStorage, c)
		<-c
		tempStorage = nil
		//check if the data has been uploaded into the map successfully
		fmt.Println(len(mStore))
		time.Sleep(5 * time.Second)
	}
	io.WriteString(w, "the auto updated has been paused!\n")

}

func isAllowed() bool {
	return true
}

func uploadCounters(kList []string, c chan bool) error {
	for _, k := range kList {
		newKey := strings.Split(k, ";")[0]
		viewCOunt, _ := strconv.Atoi(strings.Split(k, ";")[1])
		clickCount, _ := strconv.Atoi(strings.Split(k, ";")[2])

		mStore[newKey] = counters{
			view:  viewCOunt,
			click: clickCount,
		}
	}
	c <- true
	// print the number of the new data was created during the sleep time.
	fmt.Println("len:", len(kList))

	return nil
}

func limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if limiter.Allow() == false {
			http.Error(w, http.StatusText(429), http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", welcomeHandler)
	mux.HandleFunc("/view/", viewHandler)
	mux.HandleFunc("/stats/", statsHandler)

	//Test Command line
	//vegeta attack -duration=10s -rate=100 -targets=vegeta.conf | vegeta report

	log.Fatal(http.ListenAndServe(":8080", limit(mux)))

}
