package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/text/encoding/simplifiedchinese"
)

const jsonPath = "json"

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
	client := &http.Client{}
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
	client := &http.Client{}
	request, _ := http.NewRequest(http.MethodGet, weburl, nil)
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

func EscapeStructHTML(model interface{}) (string, error) {
	bf := bytes.NewBuffer([]byte{})
	jsonEncoder := json.NewEncoder(bf)
	jsonEncoder.SetEscapeHTML(false)
	err := jsonEncoder.Encode(model)
	if err != nil {
		return "", errors.New("Encode result failed" + err.Error())
	}
	return bf.String(), nil
}

func WriteFile(model interface{}, fileName string) error {
	content, err := EscapeStructHTML(model)
	if err != nil {
		return errors.New("EscapeStructHTML failed" + err.Error())
	}
	fp, err := os.OpenFile(path.Join(jsonPath, fileName), os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return errors.New("open file failed " + fileName + " " + err.Error())
	}
	defer fp.Close()
	fp.WriteString(content)
	return nil
}

/************** Get Data **************/
//百度今日热点事件排行榜
//百度实时热点排行榜
//微博热点排行榜
//贴吧热度榜单
//V2EX热度榜单
//知乎全站热榜

func parse_zhihu_rb() {
	//https://www.zhihu.com/hot this url need cookies
	weburl := "https://www.zhihu.com/api/v3/feed/topstory/hot-lists/total?limit=50&desktop=true"
	fileName := "zhihu.json"
	byteBody, err := getHttpBody(weburl)
	if err != nil {
		log.Println("getHttpBody failed", err)
		return
	}
	result := make([]JsonModel, 0)
	var zhihuhot ZhihuHot
	if err := json.Unmarshal(byteBody, &zhihuhot); err != nil {
		log.Println("Unmarshal failed", err)
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
	err = WriteFile(result, fileName)
	if err != nil {
		log.Println("WriteFile failed ", err)
		return
	}
}

func parse_website(model HotSiteModel) {
	doc, err := getGoquryDocument(model.WebURL)
	if err != nil {
		log.Println("getGoquryDocument failed", err)
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
	err = WriteFile(result, model.FileName)
	if err != nil {
		log.Println("WriteFile failed ", err)
		return
	}
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
		{WebURL: "https://www.v2ex.com/?tab=hot", FileName: "vsite.json", selector: "span.item_title a",
			SiteIsUTF8Encode: false, profixURL: "https://www.v2ex.com"},
	}
	return sites
}

func GetHotspot() {
	sites := GetHotSiteModel()
	for {
		parse_zhihu_rb()
		for _, model := range sites {
			parse_website(model)
		}
		//rum every 10 Minute
		time.Sleep(time.Duration(1) * time.Minute)
	}
}

/************** Http **************/
func handlerHome(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hi there, I love %s!", r.URL.Path[1:])
}

func readJsonFile(fileName string) ([]JsonModel, error) {
	content, err := ioutil.ReadFile(path.Join(jsonPath, fileName))
	if err != nil {
		return nil, errors.New("read file failed" + err.Error())
	}

	result := make([]JsonModel, 0)
	if err := json.Unmarshal(content, &result); err != nil {
		return nil, errors.New("Unmarshal failed" + err.Error())
	}
	return result, nil
}

func handlerHotspot(w http.ResponseWriter, r *http.Request) {
	result := make([]HotspotModel, 0)
	//zhihu
	jsonContent, _ := readJsonFile("zhihu.json")
	tmp := HotspotModel{FileName: "zhihu.json", Content: jsonContent}
	result = append(result, tmp)
	//other
	sites := GetHotSiteModel()
	for _, model := range sites {
		jsonContent, _ := readJsonFile(model.FileName)
		tmp := HotspotModel{FileName: model.FileName, Content: jsonContent}
		result = append(result, tmp)
	}
	w.Header().Set("Content-Type", "application/json")
	jsonEncoder := json.NewEncoder(w)
	jsonEncoder.SetEscapeHTML(false)
	err := jsonEncoder.Encode(result)
	if err != nil {
		log.Fatal("Encode result failed" + err.Error())
	}
}

/************** main func **************/
func main() {
	//Create json Folder
	_, err := os.Stat(jsonPath)
	if err != nil {
		os.Mkdir(jsonPath, os.ModePerm)
	}
	//get data from website
	go GetHotspot()
	//start http server
	http.HandleFunc("/hotspot", handlerHotspot)
	http.HandleFunc("/", handlerHome)
	http.ListenAndServe(":83", nil)
}
