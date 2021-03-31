package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
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
		fmt.Println("goquery new document failed", err)
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
		fmt.Println("Encode result failed", err)
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
		fmt.Println("open file failed", fileName, err)
		return errors.New("open file failed " + fileName + " " + err.Error())
	}
	defer fp.Close()
	fp.WriteString(content)
	return nil
}

/************** Get Data **************/
//baidu
func parse_baidu(weburl string, fileName string) {
	doc, err := getGoquryDocument(weburl)
	if err != nil {
		fmt.Println("getGoquryDocument failed", err)
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
	err = WriteFile(result, fileName)
	if err != nil {
		fmt.Println("WriteFile failed ", err)
		return
	}
}

//知乎全站热榜
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

func parse_zhihu_rb() {
	weburl := "https://www.zhihu.com/api/v3/feed/topstory/hot-lists/total?limit=50&desktop=true"
	fileName := "zhihu.json"
	client := &http.Client{}
	request, _ := http.NewRequest(http.MethodGet, weburl, nil)
	response, err := client.Do(request)
	if err != nil {
		fmt.Println("request error. ", err)
		return
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		fmt.Println("Get Response error. ", response.Status)
		return
	}
	result := make([]JsonModel, 0)
	byteBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		fmt.Println("read body failed", err)
		return
	}
	var zhihuhot ZhihuHot
	if err := json.Unmarshal(byteBody, &zhihuhot); err != nil {
		fmt.Println("Unmarshal failed", err)
		return
	}
	for _, item := range zhihuhot.Data {
		title := item.Target.Title
		url := item.Target.Url
		model := JsonModel{title, url}
		result = append(result, model)
	}
	err = WriteFile(result, fileName)
	if err != nil {
		fmt.Println("WriteFile failed ", err)
		return
	}
}

//微博热点排行榜

//贴吧热度榜单

//V2EX热度榜单

func GetHotspot() {
	for {
		parse_baidu(baidu_ssrd, "baidurd.json")
		parse_baidu(baidu_today, "baidusj.json")
		parse_zhihu_rb()
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
