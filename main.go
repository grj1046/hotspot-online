package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/text/encoding/simplifiedchinese"
)

type JsonModel struct {
	Name string
	Url  string
}

type Page struct {
	Title string
	Body  []byte
}

//百度今日热点事件排行榜
const baidu_today = "http://top.baidu.com/buzz?b=341"

//实时热点排行榜
const baidu_ssrd = "http://top.baidu.com/buzz?b=1"
const jsonPath = "json"

func decodeToGBK(text string) (string, error) {

	dst := make([]byte, len(text)*2)
	tr := simplifiedchinese.GB18030.NewDecoder()
	nDst, _, err := tr.Transform(dst, []byte(text), true)
	if err != nil {
		return text, err
	}

	return string(dst[:nDst]), nil
}

/************** Get Data **************/
//baidu
func parse_baidu(baidu_url string, fileName string) {
	fp, err := os.OpenFile(path.Join(jsonPath, fileName), os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		fmt.Println("open file failed", fileName, err)
		return
	}
	defer fp.Close()

	client := &http.Client{}
	request, _ := http.NewRequest(http.MethodGet, baidu_url, nil)
	// request.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9")
	// request.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8,ja;q=0.7,zh-TW;q=0.6")
	response, err := client.Do(request)
	if err != nil {
		fmt.Println("NewRequest", err)
		return
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		fmt.Printf("status code error: %d %s", response.StatusCode, response.Status)
		return
	}

	doc, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		fmt.Println("goquery new document failed", err)
		return
	}
	result := make([]JsonModel, 0)
	doc.Find("a.list-title").Each(func(i int, s *goquery.Selection) {
		//foreach item found
		host_name := s.Text()
		host_url, exists := s.Attr("href")
		if exists {
			gbkhost_name, _ := decodeToGBK(host_name)
			model := JsonModel{gbkhost_name, host_url}
			result = append(result, model)
		}
	})
	content, err := json.Marshal(result)
	if err != nil {
		fmt.Println("Marshal failed", err)
		return
	}
	fp.WriteString(string(content))
}

func GetHotspot() {
	for {
		parse_baidu(baidu_ssrd, "baidurd.json")
		parse_baidu(baidu_today, "baidusj.json")
		time.Sleep(600)
	}
}

/************** Http **************/
func handlerHome(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hi there, I love %s!", r.URL.Path[1:])
}

func handlerHotspot(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hi there, I love %s!", r.URL.Path[1:])
}

/************** main func **************/
func main() {
	//Create json Folder
	_, err := os.Stat(jsonPath)
	if err != nil {
		os.Mkdir(jsonPath, os.ModePerm)
	}
	//get data from website
	GetHotspot()
	//start http server
	// http.HandleFunc("/hotspot", handlerHotspot)
	// http.HandleFunc("/", handlerHome)
	//http.ListenAndServe(":83", nil)
}
