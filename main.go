package main

import (
	"bytes"
	"crypto/tls"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/text/encoding/simplifiedchinese"
)

type JsonModel struct {
	Name string
	Url  string
}
type HotspotModel struct {
	FileName string
	Content  []JsonModel
}

type ZhihuHot struct {
	FreshText string `json:"fresh_text"`
	Data      []struct {
		ID     string `json:"id"`
		Target struct {
			ID    int    `json:"id"`
			Title string `json:"title"`
			Url   string `json:"url"`
		} `json:"target"`
	} `json:"data"`
}

type HotSiteModel struct {
	WebURL           string
	FileName         string
	selector         string
	SiteIsUTF8Encode bool
	profixURL        string
}

//go:embed html/index.html
var embedFile embed.FS

//go:embed html/img/logo.png
var logoFile []byte
var globalData = make([]HotspotModel, 0)
var indextmpl = &template.Template{}

func decodeToGBK(text string) (string, error) {
	dst := make([]byte, len(text)*2)
	tr := simplifiedchinese.GB18030.NewDecoder()
	nDst, _, err := tr.Transform(dst, []byte(text), true)
	if err != nil {
		return text, err
	}
	return string(dst[:nDst]), nil
}

func getHttpBody(weburl string) ([]byte, error) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	request, _ := http.NewRequest(http.MethodGet, weburl, nil)
	response, err := client.Do(request)
	if err != nil {
		return nil, errors.New("NewRequest: " + err.Error())
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return nil, errors.New(fmt.Sprintf("status code error: %d %s", response.StatusCode, response.Status))
	}
	byteBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, errors.New("read body failed, " + err.Error())
	}
	return byteBody, nil
}

func getGoquryDocument(weburl string) (*goquery.Document, error) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	request, _ := http.NewRequest(http.MethodGet, weburl, nil)
	request.Header.Add("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9")
	u, _ := url.Parse(weburl)
	request.Header.Add("Host", u.Host)
	request.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/90.0.4430.212 Safari/537.36")
	response, err := client.Do(request)
	if err != nil {
		return nil, errors.New("NewRequest: " + err.Error())
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return nil, errors.New(fmt.Sprintf("status code error: %d %s", response.StatusCode, response.Status))
	}
	doc, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		return nil, errors.New("goquery new document failed, " + err.Error())
	}
	return doc, nil
}

func EscapeStructHTML(model interface{}) ([]byte, error) {
	bf := bytes.NewBuffer([]byte{})
	jsonEncoder := json.NewEncoder(bf)
	jsonEncoder.SetEscapeHTML(false)
	err := jsonEncoder.Encode(model)
	if err != nil {
		return nil, errors.New("Encode result failed" + err.Error())
	}
	return bf.Bytes(), nil
}

func writeData(fileName string, content []JsonModel) {
	exists := false
	for _, v := range globalData {
		if v.FileName == fileName {
			v.Content = content
			exists = true
			break
		}
	}
	if !exists {
		var curr = HotspotModel{FileName: fileName, Content: content}
		globalData = append(globalData, curr)
	}
}
func getIndexTemplate() (*template.Template, error) {
	funcMap := template.FuncMap{
		// The name "inc" is what the function will be called in the template text.
		"inc": func(i int) int {
			return i + 1
		},
	}
	byteHtml, err := embedFile.ReadFile("html/index.html")
	if err != nil {
		return nil, err
	}
	tmpl := template.New("index").Funcs(funcMap)
	if err != nil {
		return nil, err
	}
	tmpl, err = tmpl.Parse(string(byteHtml))
	if err != nil {
		return nil, err
	}
	return tmpl, nil
}

/************** Get Data **************/

func parse_zhihu_rb() {
	//https://www.zhihu.com/hot this url need cookies
	weburl := "https://www.zhihu.com/api/v3/feed/topstory/hot-lists/total?limit=50&desktop=true"
	fileName := "zhihu.json"
	byteBody, err := getHttpBody(weburl)
	if err != nil {
		log.Println("getHttpBody failed: ", err)
		return
	}
	result := make([]JsonModel, 0)
	var zhihuhot ZhihuHot
	if err := json.Unmarshal(byteBody, &zhihuhot); err != nil {
		log.Println("Unmarshal failed: ", err)
		return
	}
	for _, item := range zhihuhot.Data {
		title := item.Target.Title
		url := item.Target.Url
		// https://www.zhihu.com/questions/451966718
		// https://www.zhihu.com/question/451966718
		//https://api.zhihu.com/questions/451966718
		if strings.Contains(url, "api.zhihu.com/questions") {
			url = strings.Replace(url, "api.zhihu.com/questions", "www.zhihu.com/question", -1)
		}
		model := JsonModel{title, url}
		result = append(result, model)
	}
	writeData(fileName, result)
}

func parse_website(model HotSiteModel) {
	doc, err := getGoquryDocument(model.WebURL)
	if err != nil {
		log.Println("getGoquryDocument failed: ", err)
		return
	}
	result := make([]JsonModel, 0)
	doc.Find(model.selector).Each(func(i int, s *goquery.Selection) {
		//foreach item found
		title := s.Text()
		host_url, exists := s.Attr("href")
		if exists {
			if model.SiteIsUTF8Encode {
				title, _ = decodeToGBK(title)
			}
			//过滤微博的广告
			if strings.Index(host_url, "javascript:void(0)") != -1 {
				return
			}
			model := JsonModel{title, model.profixURL + host_url}
			result = append(result, model)
		}
	})
	writeData(model.FileName, result)
}

func GetHotSiteModel() []HotSiteModel {
	sites := []HotSiteModel{
		{WebURL: "http://top.baidu.com/buzz?b=341", FileName: "baidusj.json", selector: "a.list-title",
			SiteIsUTF8Encode: true, profixURL: ""},
		{WebURL: "http://top.baidu.com/buzz?b=1", FileName: "baidurd.json", selector: "a.list-title",
			SiteIsUTF8Encode: true, profixURL: ""},
		{WebURL: "https://s.weibo.com/top/summary?cate=realtimehot", FileName: "weibo.json", selector: "td.td-02 a",
			SiteIsUTF8Encode: false, profixURL: "https://s.weibo.com"},
		{WebURL: "http://tieba.baidu.com/hottopic/browse/topicList?res_type=1", FileName: "tieba.json", selector: "a.topic-text",
			SiteIsUTF8Encode: false, profixURL: ""},
		{WebURL: "https://www.douban.com/group/explore", FileName: "douban.json", selector: "div.channel-item h3 a",
			SiteIsUTF8Encode: false, profixURL: ""},
	}
	return sites
}

func GetHotspot() {
	timer_duration := os.Getenv("HOTSPOT_TIMER_DURATION")
	duration, err := strconv.Atoi(timer_duration)
	if err != nil {
		duration = 10
	}
	sites := GetHotSiteModel()
	for {
		parse_zhihu_rb()
		for _, model := range sites {
			parse_website(model)
		}
		//rum every 10 Minute
		time.Sleep(time.Duration(duration) * time.Minute)
	}
}

/************** Http **************/

func handlerHome(w http.ResponseWriter, r *http.Request) {
	err := indextmpl.Execute(w, globalData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// w.Write(byteHtml)
}
func handlerLogo(w http.ResponseWriter, r *http.Request) {
	w.Write(logoFile)
}

func handlerHotspot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	data, err := EscapeStructHTML(globalData)
	if err != nil {
	}
	escapedData := []HotspotModel{}
	err = json.Unmarshal(data, &escapedData)
	if err != nil {
	}
	jsonEncoder := json.NewEncoder(w)
	jsonEncoder.SetEscapeHTML(false)
	_ = jsonEncoder.Encode(escapedData)
}

/************** main func **************/
func main() {
	httpPort := os.Getenv("HOTSPOT_HTTP_PORT")
	port, err := strconv.Atoi(httpPort)
	if err != nil {
		port = 80
	}
	httpPort = ":" + strconv.Itoa(port)
	//get data from website
	go GetHotspot()
	//start http server
	indextmpl, err = getIndexTemplate()
	if err != nil {
		log.Fatal("ListenAndServe failed: ", err)
		return
	}
	http.HandleFunc("/img/logo.png", handlerLogo)
	http.HandleFunc("/favicon.ico", handlerLogo)
	http.HandleFunc("/hotspot", handlerHotspot)
	http.HandleFunc("/", handlerHome)
	err = http.ListenAndServe(httpPort, nil)
	if err != nil {
		log.Fatal("ListenAndServe failed: ", err)
		return
	}
	log.Println("ListenAndServe: ", httpPort)
}
