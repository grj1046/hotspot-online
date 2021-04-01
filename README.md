# hotspot-online

各大平台热榜聚合-实时更新

### 00.简介
本项目来自于 https://github.com/OpsFans/hotspot-online
朋友写论文要用到这个，部署到CoreOS的时候各种不如意，最近在学习golang，使用golang重写了一下，golang天生亲和Docker，非常奈斯。

#### 01.采集数据 使用的 goruntine 默认每10分钟更新一次数据（可通过环境变量`HOTSPOT_TIMER_DURATION`设置时间，单位分钟。）
01. 使用http模块发送请求获得网页数据，

02. 使用 `github.com/PuerkitoBio/goquery` 包 html并清洗出自己想要的数据  

03. 使用 `golang.org/x/text/encoding/simplifiedchinese` 包来处理GB2312编码的转换

03. 本地化处理（写入到本地json文件）

#### 02.处理并返回数据 (可通过环境变量`HOTSPOT_HTTP_PORT`设置端口，默认值`80`)
接口  `/hotspot` 会将本地json文件读取并按照需求返回为json格式接口
返回格式如下:

```
[{
    FileName: '',
    Content: [{
        Name: '',
        Url: ''
    }]
}]
```

#### 03.前端展示
前端采用Bootstrap4 来展示，用`jquery.getJSON`从远程接口获取数据，来渲染页面。

### 04.添加对Docker的支持
使用的golang alpine生成的镜像有点大 366MB
改为使用Scratch作镜像，大小只有5MB 奈斯

Docker-compose.yml的模式没去研究

### 其它有用无用的东西
https://golang.org/pkg/embed/
https://harsimranmaan.medium.com/embedding-static-files-in-a-go-binary-using-go-embed-bac505f3cb9a
https://blog.cloud66.com/how-to-create-the-smallest-possible-docker-image-for-your-golang-application/