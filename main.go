package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	dysmsapi "github.com/alibabacloud-go/dysmsapi-20170525/v3/client"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/debug"

	"github.com/robertkrimen/otto"
	gojsonq "github.com/thedevsaddam/gojsonq/v2"
)

type DH struct {
	DHID string
}

type Badminton struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type KYY struct {
	CODE          string `json:"CODE"`
	NAME          string `json:"NAME"`
	STATE_EXPLAIN string `json:"STATE_EXPLAIN"`
	WID           string `json:"WID"`
	Disabled      bool   `json:"disabled"`
	Text          string `json:"text"`
}

type OpenRoomResponse struct {
	Code  string   `json:"code"`
	Datas OpenRoom `json:"datas"`
}

type OpenRoom struct {
	GetOpeningRoom OpenRoomObject `json:"getOpeningRoom"`
}

type OpenRoomObject struct {
	PageNumber int            `json:"pageNumber"`
	PageSize   int            `json:"pageSize"`
	TotalSize  int            `json:"totalSize"`
	Rows       []OpenRoomData `json:"rows"`
}

type OpenRoomData struct {
	BCRSXZ                string `json:"BCRSXZ"`
	CDMC                  string `json:"CDMC"`
	CGBM                  string `json:"CGBM"`
	CGBM_DISPLAY          string `json:"CGBM_DISPLAY"`
	DCFS                  string `json:"DCFS"`
	DCFS_DISPLAY          string `json:"DCFS_DISPLAY"`
	SCWSDPRS              string `json:"SCWSDPRS"`
	STATE_EXPLAIN         string `json:"STATE_EXPLAIN"`
	STATE_EXPLAIN_DISPLAY string `json:"STATE_EXPLAIN_DISPLAY"`
	WID                   string `json:"WID"`
	XMDM                  string `json:"XMDM"`
	XMDM_DISPLAY          string `json:"XMDM_DISPLAY"`
	XQDM                  string `json:"XQDM"`
	XQDM_DISPLAY          string `json:"XQDM_DISPLAY"`
	Disabled              bool   `json:"disabled"`
	Text                  string `json:"text"`
}

type UserInfo struct {
	UserId       string
	UserName     string
	Password     string
	PhoneNumber  string
	SportDate    string
	FirstTime    string
	SecondTime   string
	IfExecNow    string
	WEU          string
	MOD_AUTH_CAS string
	firstRound   bool
	secondRound  bool
}

type SMSConfig struct {
	AccessKeyID     string `json:"accessKeyId"`
	AccessKeySecret string `json:"accessKeySecret"`
	SignName        string `json:"signName"`
	TemplateCode    string `json:"templateCode"`
	RegionID        string `json:"regionId"`
}

type GoroutineInfo struct {
	// true 表示任务完成，false 表示还在运行
	FirstStatus  bool
	SecondStatus bool
	// Identification of Goroutine
	Identification int
	// 学号
	UserId   string
	UserName string
	// 日期
	ReservationDate string
	// 场次时间
	FirstReservationTime  string
	SecondReservationTime string
}

var goroutines map[int]*GoroutineInfo
var smsConfig *SMSConfig
var addLock sync.Mutex
var deleteLock sync.Mutex
var deleteLock2 sync.Mutex
var deleteChan chan bool = make(chan bool)
var deletingFlag bool = false
var idx int = 0

func loadSMSConfigFromFile() (*SMSConfig, error) {
	data, err := ioutil.ReadFile("sms_config.json")
	if os.IsNotExist(err) {
		return &SMSConfig{}, nil
	}
	if err != nil {
		return nil, err
	}

	var cfg SMSConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func saveSMSConfigToFile(cfg *SMSConfig) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile("sms_config.json", data, 0644)
}

func valueOrEnv(value, envKey string) string {
	if value != "" {
		return value
	}

	return os.Getenv(envKey)
}

func getActiveSMSConfig() *SMSConfig {
	if smsConfig == nil {
		cfg, err := loadSMSConfigFromFile()
		if err != nil {
			log.Printf("failed to load sms config: %v", err)
			smsConfig = &SMSConfig{}
		} else {
			smsConfig = cfg
		}
	}

	return &SMSConfig{
		AccessKeyID:     valueOrEnv(smsConfig.AccessKeyID, "ALIYUN_SMS_ACCESS_KEY_ID"),
		AccessKeySecret: valueOrEnv(smsConfig.AccessKeySecret, "ALIYUN_SMS_ACCESS_KEY_SECRET"),
		SignName:        valueOrEnv(smsConfig.SignName, "ALIYUN_SMS_SIGN_NAME"),
		TemplateCode:    valueOrEnv(smsConfig.TemplateCode, "ALIYUN_SMS_TEMPLATE_CODE"),
		RegionID:        valueOrEnv(smsConfig.RegionID, "ALIYUN_SMS_REGION_ID"),
	}
}

func getDHID(urls string, user *UserInfo) string {
	// formValues := url.Values{}
	// formValues.Set("wid", "15093a7663fa498695608f3d52cca59d")
	// formDataStr := formValues.Encode()
	// formDataBytes := []byte(formDataStr)
	// formBytesReader := bytes.NewReader(formDataBytes)

	req, err := http.NewRequest("POST", urls,
		nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Accept", "application/json, text/javascript, */*; q=0.01")
	// request.Header.Add("Accept-Language", "zh-CN,zh;q=0.8,en-US;q=0.5,en;q=0.3")
	req.Header.Add("Connection", "keep-alive")

	cookie2 := &http.Cookie{Name: "_WEU", Value: user.WEU, HttpOnly: true}
	cookie9 := &http.Cookie{Name: "MOD_AUTH_CAS", Value: user.MOD_AUTH_CAS, HttpOnly: true}
	// no need to modify
	cookie4 := &http.Cookie{Name: "insert_cookie", Value: "28057208", HttpOnly: true}
	cookie13 := &http.Cookie{Name: "EMAP_LANG", Value: "zh"}
	req.AddCookie(cookie2)
	req.AddCookie(cookie4)
	req.AddCookie(cookie9)
	req.AddCookie(cookie13)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	byts, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		log.Fatal(err)
	}
	dh := DH{}
	json.Unmarshal(byts, &dh)

	errno := gojsonq.New().FromString(string(byts)).Find("code")
	if err != nil {
		fmt.Println("错误码：", errno)
	}

	return dh.DHID
}

func getYY(year int, month int, day int, startTime string, endTime string) (string, string, string, string) {
	var YYRQ string
	var KYYSJD string
	var YYKS string
	var YYJS string
	if day >= 10 {
		if month >= 10 {
			YYRQ = fmt.Sprintf("%d-%d-%d", year, month, day)
		} else {
			YYRQ = fmt.Sprintf("%d-0%d-%d", year, month, day)
		}
	} else {
		if month >= 10 {
			YYRQ = fmt.Sprintf("%d-%d-0%d", year, month, day)
		} else {
			YYRQ = fmt.Sprintf("%d-0%d-0%d", year, month, day)
		}
	}
	KYYSJD = fmt.Sprintf("%s:00-%s:00", startTime, endTime)
	YYKS = fmt.Sprintf("%s %s:00", YYRQ, startTime)
	YYJS = fmt.Sprintf("%s %s:00", YYRQ, endTime)
	return YYRQ, KYYSJD, YYKS, YYJS
}

func getBadmitonData(year int, month int, day int, startTime string, endTime string) []Badminton {
	var badmitons_data []Badminton
	// urls := "https://ehall.szu.edu.cn/publicapp/sys/tycgyyxt/sportVenue/getCdxx.do"
	// YYRQ, _, YYKS, YYJS := getYY(month, day, startTime, endTime)

	// formValues := url.Values{}
	// formValues.Set("YYRQ", YYRQ)
	// formValues.Set("START", YYKS)
	// formValues.Set("END", YYJS)
	// formValues.Set("CGBM", "002")
	// formValues.Set("XMDM", "005")
	// formValues.Set("TYPE", "YY_TT")
	// formValues.Set("YYTYPE", "1.0")
	// formDataStr := formValues.Encode()
	// formDataBytes := []byte(formDataStr)
	// formBytesReader := bytes.NewReader(formDataBytes)

	// req, err := http.NewRequest("POST", urls,
	// 	formBytesReader)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// req.Header.Add("Accept", "application/json, text/javascript, */*; q=0.01")
	// // request.Header.Add("Accept-Language", "zh-CN,zh;q=0.8,en-US;q=0.5,en;q=0.3")
	// req.Header.Add("Connection", "keep-alive")

	// cookie2 := &http.Cookie{Name: "_WEU", Value: _WEU, HttpOnly: true}
	// cookie9 := &http.Cookie{Name: "MOD_AUTH_CAS", Value: MOD_AUTH_CAS, HttpOnly: true}
	// // no need to modify
	// cookie4 := &http.Cookie{Name: "insert_cookie", Value: "28057208", HttpOnly: true}
	// cookie13 := &http.Cookie{Name: "EMAP_LANG", Value: "zh"}
	// req.AddCookie(cookie2)
	// req.AddCookie(cookie4)
	// req.AddCookie(cookie9)
	// req.AddCookie(cookie13)

	// resp, err := http.DefaultClient.Do(req)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// byts, err := ioutil.ReadAll(resp.Body)
	// defer resp.Body.Close()
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// fmt.Println(string(byts))
	// json.Unmarshal(byts, &badmitons_data)

	// errno := gojsonq.New().FromString(string(byts)).Find("code")
	// if err != nil {
	// 	fmt.Println("错误码：", errno)
	// }

	filePtr, err := os.Open("./badmiton.json")
	if err != nil {
		fmt.Println(err.Error())
		return nil
	}
	defer filePtr.Close()
	// 创建json解码器
	decoder := json.NewDecoder(filePtr)
	err = decoder.Decode(&badmitons_data)
	if err != nil {
		fmt.Println("decode failed", err.Error())
		return nil
	}

	return badmitons_data
}

func httpRequestDHID(urls string, dhID string, year int, month int, day int, startTime string, endTime string, user *UserInfo) bool {

	badminton := getBadmitonData(year, month, day, startTime, endTime)
	count := 0
	if len(badminton) == 0 {
		return false
	}
	for _, value := range badminton {
		if !getKyydata(value.Id, year, month, day, startTime, endTime, user) {
			fmt.Printf("没有kyy data ")
			continue
		}
		if count > 2 {
			fmt.Println("request too much, just rest.")
			return false
		}
		formValues := url.Values{}
		// formValues.Set("DHID", dhID)
		formValues.Set("DHID", "")
		formValues.Set("YYRGH", user.UserId)
		formValues.Set("CYRS", "")
		formValues.Set("YYRXM", user.UserName)
		formValues.Set("LXFS", user.PhoneNumber)
		formValues.Set("CGDM", "001")
		// 场地ID, 不固定, 需要读取JSON文件
		formValues.Set("CDWID", value.Id)
		formValues.Set("XMDM", "001")
		formValues.Set("XQWID", "1")
		// 时间段信息
		YYRQ, KYYSJD, YYKS, YYJS := getYY(year, month, day, startTime, endTime)
		formValues.Set("KYYSJD", KYYSJD)
		formValues.Set("YYRQ", YYRQ)
		formValues.Set("YYLX", "1.0")
		formValues.Set("YYKS", YYKS)
		formValues.Set("YYJS", YYJS)
		formValues.Set("PC_OR_PHONE", "pc")
		// 以下信息全固定
		formDataStr := formValues.Encode()
		formDataBytes := []byte(formDataStr)
		formBytesReader := bytes.NewReader(formDataBytes)

		req, err := http.NewRequest("POST", urls,
			formBytesReader)
		if err != nil {
			log.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Add("Accept", "application/json, text/javascript, */*; q=0.01")
		req.Header.Add("Connection", "keep-alive")

		cookie2 := &http.Cookie{Name: "_WEU", Value: user.WEU, HttpOnly: true}
		cookie9 := &http.Cookie{Name: "MOD_AUTH_CAS", Value: user.MOD_AUTH_CAS, HttpOnly: true}
		// no need to modify
		cookie4 := &http.Cookie{Name: "insert_cookie", Value: "28057208", HttpOnly: true}
		cookie13 := &http.Cookie{Name: "EMAP_LANG", Value: "zh"}
		req.AddCookie(cookie2)
		req.AddCookie(cookie4)
		req.AddCookie(cookie9)
		req.AddCookie(cookie13)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Fatal(err)
		}

		byts, err := ioutil.ReadAll(resp.Body)
		defer resp.Body.Close()
		if err != nil {
			log.Fatal(err)
		}

		if strings.Contains(string(byts), "false") {
			fmt.Println("ERROR: ", string(byts))
			count++
		} else {
			fmt.Println(string(byts), "OK!")
			return true
		}

		errno := gojsonq.New().FromString(string(byts)).Find("code")
		if err != nil {
			fmt.Println("错误码：", errno)
			return false
		}
	}

	return false
}

func sendSMSNotification(user *UserInfo, reservationTime string) error {
	if user.PhoneNumber == "" {
		return fmt.Errorf("phone number is empty for user %s", user.UserName)
	}

	activeConfig := getActiveSMSConfig()
	accessKeyID := activeConfig.AccessKeyID
	accessKeySecret := activeConfig.AccessKeySecret
	signName := activeConfig.SignName
	templateCode := activeConfig.TemplateCode
	regionID := activeConfig.RegionID
	if regionID == "" {
		regionID = "cn-hangzhou"
	}

	if accessKeyID == "" || accessKeySecret == "" || signName == "" || templateCode == "" {
		return fmt.Errorf("sms config is not complete")
	}

	config := &openapi.Config{
		AccessKeyId:     tea.String(accessKeyID),
		AccessKeySecret: tea.String(accessKeySecret),
		RegionId:        tea.String(regionID),
	}

	smsClient, err := dysmsapi.NewClient(config)
	if err != nil {
		return err
	}

	messageParams := fmt.Sprintf(`{"name":"%s","date":"%s","time":"%s"}`, user.UserName, user.SportDate, reservationTime)
	request := &dysmsapi.SendSmsRequest{
		PhoneNumbers:  tea.String(user.PhoneNumber),
		SignName:      tea.String(signName),
		TemplateCode:  tea.String(templateCode),
		TemplateParam: tea.String(messageParams),
	}

	_, err = smsClient.SendSms(request)
	return err
}

func getOpeningRoom(CDWID string, year int, month int, day int, startTime string, endTime string, user *UserInfo) bool {
	urls := "https://ehall.szu.edu.cn/qljfwapp/sys/lwSzuCgyy/modules/sportVenue/getOpeningRoom.do"
	YYRQ, _, YYKS, YYJS := getYY(year, month, day, startTime, endTime)

	formValues := url.Values{}
	formValues.Set("XMDM", "001")
	formValues.Set("YYRQ", YYRQ)
	formValues.Set("YYLX", "1.0")
	formValues.Set("KSSJ", strings.Split(YYKS, " ")[1])
	formValues.Set("JSSJ", strings.Split(YYJS, " ")[1])
	formValues.Set("XQDM", "1")
	// fmt.Println("预约开始", strings.Split(YYKS, " ")[1])
	// fmt.Println("预约结束", strings.Split(YYJS, " ")[1])
	// fmt.Println("formValues:", formValues)
	formDataStr := formValues.Encode()
	formDataBytes := []byte(formDataStr)
	formBytesReader := bytes.NewReader(formDataBytes)

	req, err := http.NewRequest("POST", urls,
		formBytesReader)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Accept", "application/json, text/javascript, */*; q=0.01")
	req.Header.Add("Connection", "keep-alive")

	cookie2 := &http.Cookie{Name: "_WEU", Value: user.WEU, HttpOnly: true}
	cookie9 := &http.Cookie{Name: "MOD_AUTH_CAS", Value: user.MOD_AUTH_CAS, HttpOnly: true}
	// no need to modify
	cookie4 := &http.Cookie{Name: "insert_cookie", Value: "28057208", HttpOnly: true}
	cookie13 := &http.Cookie{Name: "EMAP_LANG", Value: "zh"}
	req.AddCookie(cookie2)
	req.AddCookie(cookie4)
	req.AddCookie(cookie9)
	req.AddCookie(cookie13)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
		return false
	}

	byts, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		log.Fatal(err)
		return false
	}
	// fmt.Println(string(byts))
	openRoomData := OpenRoomResponse{}
	json.Unmarshal(byts, &openRoomData)

	// fmt.Println("openRoomData.getOpeningRoom.rows ", openRoomData.Datas.GetOpeningRoom.Rows)
	for _, v := range openRoomData.Datas.GetOpeningRoom.Rows {
		// fmt.Println(v.WID, CDWID, v.Disabled, v.Text)
		if v.WID == CDWID && !v.Disabled && v.Text == "可预约" {
			return true
		}
	}

	return false
}

func getKyydata(CDWID string, year int, month int, day int, startTime string, endTime string, user *UserInfo) bool {
	urls := "https://ehall.szu.edu.cn/qljfwapp/sys/lwSzuCgyy/sportVenue/getTimeList.do"
	YYRQ, KYYSJD, _, _ := getYY(year, month, day, startTime, endTime)

	formValues := url.Values{}
	formValues.Set("XQ", "1")
	fmt.Println("YYRQ", YYRQ)
	formValues.Set("YYRQ", YYRQ)
	formValues.Set("XMDM", "001")
	formValues.Set("YYLX", "1.0")
	formDataStr := formValues.Encode()
	formDataBytes := []byte(formDataStr)
	formBytesReader := bytes.NewReader(formDataBytes)

	req, err := http.NewRequest("POST", urls,
		formBytesReader)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Add("Accept", "*/*")
	req.Header.Add("Connection", "keep-alive")
	req.Header.Add("Host", "ehall.szu.edu.cn")
	req.Header.Add("Origin", "https://ehall.szu.edu.cn")
	req.Header.Add("Referer", "https://ehall.szu.edu.cn/qljfwapp/sys/lwSzuCgyy/index.do")

	// fmt.Println("WEU", user.WEU)
	// fmt.Println("MOD_AUTH_CAS", user.MOD_AUTH_CAS)
	cookie2 := &http.Cookie{Name: "_WEU", Value: user.WEU}
	cookie9 := &http.Cookie{Name: "MOD_AUTH_CAS", Value: user.MOD_AUTH_CAS}
	// no need to modify
	cookie4 := &http.Cookie{Name: "asessionid", Value: "f7d75b63-1d8d-4b30-91c1-3ea268e2a296"}
	cookie5 := &http.Cookie{Name: "route", Value: "c74f3c8250d849c2cfd6230ee3f779bd"}
	cookie13 := &http.Cookie{Name: "amp.locale", Value: "undefined"}
	cookie14 := &http.Cookie{Name: "EMAP_LANG", Value: "zh"}
	req.AddCookie(cookie2)
	req.AddCookie(cookie4)
	req.AddCookie(cookie9)
	req.AddCookie(cookie13)
	req.AddCookie(cookie5)
	req.AddCookie(cookie14)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
		return false
	}

	byts, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()

	// fmt.Println("resp.Header: ", resp.Header.Values("Set-Cookie"), len(resp.Header.Values("Set-Cookie")))
	// fmt.Println()
	if len(resp.Header.Values("Set-Cookie")) == 1 {
		user.WEU = strings.Split(strings.Split(resp.Header.Values("Set-Cookie")[0], ";")[0], "=")[1]
	}

	if err != nil {
		log.Fatal(err)
		return false
	}
	//fmt.Println(string(byts))
	var kyyData []KYY
	json.Unmarshal(byts, &kyyData)

	for _, v := range kyyData {
		//fmt.Println(v.CODE, KYYSJD, v.Disabled, v.Text)
		if v.CODE == KYYSJD && !v.Disabled && v.Text == "可预约" {
			// time is suitable, and then check the CD if suitable
			if getOpeningRoom(CDWID, year, month, day, startTime, endTime, user) {
				return true
			} else {
				fmt.Println("该时间该场地已约完，尝试换该时间其他场地中")
				return false
			}
		}
	}

	errno := gojsonq.New().FromString(string(byts)).Find("code")
	if err != nil {
		fmt.Println("错误码：", errno)
		return false
	}

	return false
}

func execRub(user *UserInfo, goroutineID int) bool {
	dhID := getDHID("https://ehall.szu.edu.cn/qljfwapp/sys/lwSzuCgyy/sportVenue/getOrderNum.do", user)

	// date
	var year int
	var month int
	var day int
	arr := strings.Split(user.SportDate, "-")
	year, _ = strconv.Atoi(arr[0])
	month, _ = strconv.Atoi(arr[1])
	day, _ = strconv.Atoi(arr[2])

	// first time
	var startTime int
	timeArr := strings.Split(user.FirstTime, ":")
	startTime, _ = strconv.Atoi(timeArr[0])
	endTime := strconv.Itoa(startTime + 1)
	if len(endTime) == 1 {
		endTime = "0" + endTime
	}

	// second time
	var timeArr2 []string
	var endTime2 string
	if user.SecondTime != "00:00" {
		timeArr2 = strings.Split(user.SecondTime, ":")
		startTime2, _ := strconv.Atoi(timeArr2[0])
		endTime2 = strconv.Itoa(startTime2 + 1)
		if len(endTime2) == 1 {
			endTime2 = "0" + endTime2
		}
	}

	waitGroup := sync.WaitGroup{}
	waitGroup.Add(2)

	firstSMSent := false
	secondSMSent := false

	go func() {
		for !user.firstRound {
			success := httpRequestDHID("https://ehall.szu.edu.cn/qljfwapp/sys/lwSzuCgyy/sportVenue/insertVenueBookingInfo.do",
				dhID, year, month, day, timeArr[0], endTime, user)
			// return err
			user.firstRound = success
			if success && !firstSMSent {
				if err := sendSMSNotification(user, user.FirstTime); err != nil {
					log.Printf("send sms for first slot failed: %v", err)
				}
				firstSMSent = true
			}
			if !user.firstRound {
				// 被通知需要关闭
				if goroutines[goroutineID].FirstStatus {
					user.firstRound = true
				}
				fmt.Println("第一轮尝试中...")
				// 3
				time.Sleep(3 * time.Second)
			}
		}
		waitGroup.Done()
		addLock.Lock()
		goroutines[goroutineID].FirstStatus = true
		addLock.Unlock()
		fmt.Println("first round success.")
	}()

	dhID2 := getDHID("https://ehall.szu.edu.cn/qljfwapp/sys/lwSzuCgyy/sportVenue/getOrderNum.do", user)
	for !user.secondRound {
		success := httpRequestDHID("https://ehall.szu.edu.cn/qljfwapp/sys/lwSzuCgyy/sportVenue/insertVenueBookingInfo.do",
			dhID2, year, month, day, timeArr2[0], endTime2, user)
		user.secondRound = success
		if success && !secondSMSent && user.SecondTime != "00:00" {
			if err := sendSMSNotification(user, user.SecondTime); err != nil {
				log.Printf("send sms for second slot failed: %v", err)
			}
			secondSMSent = true
		}
		if !success {
			// 被通知需要关闭
			if goroutines[goroutineID].SecondStatus {
				user.secondRound = true
			}
			fmt.Println("第二轮尝试中...")
			// 3
			time.Sleep(3 * time.Second)
		}
	}
	waitGroup.Done()
	addLock.Lock()
	goroutines[goroutineID].SecondStatus = true
	addLock.Unlock()
	fmt.Println("second round success.")

	waitGroup.Wait()

	delete(goroutines, goroutineID)

	if deletingFlag {
		// 通知删除完成
		deleteLock.Lock()
		deleteChan <- true
		deleteLock.Unlock()
	}

	return true
}

func callJavascript(password, salt string) string {
	filePath := "./encrypt.js"

	bytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		panic(err)
	}

	vm := otto.New()

	_, err = vm.Run(string(bytes))
	if err != nil {
		panic(err)
	}

	value, err := vm.Call("encryptAES", nil, password, salt)
	if err != nil {
		panic(err)
	}
	// fmt.Println(value.String())
	return value.String()
}

func process(w http.ResponseWriter, r *http.Request) {
	t := template.Must(template.ParseFiles("./templates/tmpl.html"))

	dataEncoded, err := ioutil.ReadFile("users")
	if err != nil {
		panic(err)
	}
	var usersDecode []*UserInfo
	json.Unmarshal(dataEncoded, &usersDecode)
	config := getActiveSMSConfig()
	configMessage := r.URL.Query().Get("config")

	if r.Method != http.MethodPost {
		t.Execute(w, struct {
			Result        bool
			Message       string
			UserInfo      []*UserInfo
			Config        *SMSConfig
			ConfigMessage string
		}{false, "", usersDecode, config, configMessage})
		return
	}

	user := UserInfo{
		UserId:      r.FormValue("user_id"),
		UserName:    r.FormValue("user_name"),
		Password:    r.FormValue("password"),
		PhoneNumber: r.FormValue("phone_number"),
		SportDate:   r.FormValue("sportDate"),
		FirstTime:   r.FormValue("firstTime"),
		SecondTime:  r.FormValue("secondTime"),
		IfExecNow:   r.FormValue("ifExecuteNow"),
	}

	fmt.Println(user)

	var result = false

	if user.UserId != "" && user.UserName != "" && user.Password != "" {
		tempId := 0
		addLock.Lock()
		// 添加goroutine信息
		newGoroutine := GoroutineInfo{
			FirstStatus:           false,
			SecondStatus:          false,
			Identification:        idx,
			UserId:                user.UserId,
			UserName:              user.UserName,
			ReservationDate:       user.SportDate,
			FirstReservationTime:  user.FirstTime,
			SecondReservationTime: user.SecondTime,
		}
		tempId = idx
		goroutines[idx] = &newGoroutine
		idx++
		addLock.Unlock()

		result = startRub(&user, tempId)
	}

	if result {
		t.Execute(w, struct {
			Result        bool
			Message       string
			UserInfo      []*UserInfo
			Config        *SMSConfig
			ConfigMessage string
		}{result, "成功", usersDecode, config, configMessage})
	} else {
		t.Execute(w, struct {
			Result        bool
			Message       string
			UserInfo      []*UserInfo
			Config        *SMSConfig
			ConfigMessage string
		}{result, "失败", usersDecode, config, configMessage})
	}
}

func updateSMSConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	updated := &SMSConfig{
		AccessKeyID:     strings.TrimSpace(r.FormValue("sms_access_key_id")),
		AccessKeySecret: strings.TrimSpace(r.FormValue("sms_access_key_secret")),
		SignName:        strings.TrimSpace(r.FormValue("sms_sign_name")),
		TemplateCode:    strings.TrimSpace(r.FormValue("sms_template_code")),
		RegionID:        strings.TrimSpace(r.FormValue("sms_region_id")),
	}

	smsConfig = updated
	if updated.RegionID == "" {
		smsConfig.RegionID = "cn-hangzhou"
	}

	if err := saveSMSConfigToFile(updated); err != nil {
		log.Printf("failed to save sms config: %v", err)
	}

	http.Redirect(w, r, "/?config=saved", http.StatusSeeOther)
}

func add(w http.ResponseWriter, r *http.Request) {
	t := template.Must(template.ParseFiles("./templates/add.html"))

	if r.Method != http.MethodPost {
		dataEncoded, err := ioutil.ReadFile("users")
		if err != nil {
			panic(err)
		}
		var usersDecode []*UserInfo
		json.Unmarshal(dataEncoded, &usersDecode)
		if len(usersDecode) == 0 {
			log.Println("Users information are nil")
			t.Execute(w, nil)
		} else {
			fmt.Printf("Have %d users\n", len(usersDecode))
			t.Execute(w, struct {
				ErrorHave bool
				Already   []*UserInfo
			}{false, usersDecode})
		}
	}

	// get the already information
	alreadyDataEncode, err := ioutil.ReadFile("users")
	if err != nil {
		panic(err)
	}
	var alreadyUsersDecode []*UserInfo
	json.Unmarshal(alreadyDataEncode, &alreadyUsersDecode)

	newUser := UserInfo{
		UserId:      r.FormValue("user_id"),
		UserName:    r.FormValue("user_name"),
		Password:    r.FormValue("password"),
		PhoneNumber: r.FormValue("phone_number"),
	}

	// when a link to /add, it will take a POST method, skip that
	if newUser.UserId == "" {
		fmt.Println("SKIP")
		return
	}

	for _, v := range alreadyUsersDecode {
		if v.UserId == newUser.UserId {
			fmt.Println("already have this user")
			t.Execute(w, struct {
				ErrorHave bool
				Already   []*UserInfo
			}{true, alreadyUsersDecode})
			return
		}
	}

	fmt.Println("newUser", newUser)

	alreadyUsersDecode = append(alreadyUsersDecode, &newUser)
	data, err := json.Marshal(alreadyUsersDecode)
	if err != nil {
		panic(err)
	}
	// add the new information to the file
	err = ioutil.WriteFile("users", data, 0644)
	if err != nil {
		panic(err)
	}

	t.Execute(w, struct {
		ErrorHave bool
		Already   []*UserInfo
	}{false, alreadyUsersDecode})
}

func getTheToken(user *UserInfo) {
	writer, err := os.OpenFile("collector.log", os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		panic(err)
	}

	// create a new collector
	c := colly.NewCollector(colly.Debugger(&debug.LogDebugger{Output: writer}), colly.MaxDepth(2))

	// attributes
	var lt string
	var dllt string
	var execution string
	var _eventId string
	// var rmShown string
	var pwdDefaultEncryptSalt string

	// Find and visit all links
	c.OnHTML("form#pwdFromId", func(e *colly.HTMLElement) {
		selection := e.DOM

		ltTemp, ok := selection.Find("input[name=lt]").Attr("value")
		if !ok {
			fmt.Println("Not OK in lt")
			return
		}
		lt = ltTemp
		dlltTemp, ok := selection.Find("input[name=dllt]").Attr("value")
		if !ok {
			fmt.Println("Not OK in dllt")
			return
		}
		dllt = dlltTemp
		executionTemp, ok := selection.Find("input[name=execution]").Attr("value")
		if !ok {
			fmt.Println("Not OK in execution")
			return
		}
		execution = executionTemp
		_eventIdTemp, ok := selection.Find("input[name=_eventId]").Attr("value")
		if !ok {
			fmt.Println("Not OK in _eventId")
			return
		}
		_eventId = _eventIdTemp
		//rmShownTemp, ok := selection.Find("input[name=rmShown]").Attr("value")
		//if !ok {
		//	fmt.Println("Not OK in rmShown")
		//	return
		//}
		//rmShown = rmShownTemp
		pwdDefaultEncryptSaltTemp, ok := selection.Find("input#pwdEncryptSalt").Attr("value")
		if !ok {
			fmt.Println("Not OK in pwdDefaultEncryptSalt")
			return
		}
		pwdDefaultEncryptSalt = pwdDefaultEncryptSaltTemp

	})

	ehallUrl := "https://authserver.szu.edu.cn/authserver/login?service=https%3A%2F%2Fehall.szu.edu.cn%3A443%2Fqljfwapp%2Fsys%2FlwSzuCgyy%2Findex.do%23%2FsportVenue"
	c.Request("GET", ehallUrl, nil, nil, nil)
	c.Wait()

	// get the encrypt password
	password := callJavascript(user.Password, pwdDefaultEncryptSalt)
	// fmt.Println("Start Login...", lt, dllt, execution, _eventId, rmShown, pwdDefaultEncryptSalt, password)
	fmt.Println()

	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("Content-Type", "application/x-www-form-urlencoded")
	})

	firstDone := false

	c.OnResponse(func(r *colly.Response) {
		// 这里有两个WEU，直接使用最后一个试试
		log.Println("Set-Cookie ", r.Headers.Values("Set-Cookie"))
		fmt.Println()

		cookies := r.Request.Headers.Get("Cookie")
		log.Println("Cookie: ", cookies)
		fmt.Println()

		modAuthCas := strings.Split(cookies[strings.Index(cookies, "MOD_AUTH_CAS"):], "=")
		if len(modAuthCas) == 2 && modAuthCas[0] == "MOD_AUTH_CAS" {
			user.MOD_AUTH_CAS = modAuthCas[1]
			fmt.Println("Set the MOD_AUTH_CAS: ", user.MOD_AUTH_CAS)
			fmt.Println()
			fmt.Println("Headers", r.Headers.Values("Set-Cookie"), len(r.Headers.Values("Set-Cookie")))
			fmt.Println("Testing", r.Headers.Values("Set-Cookie")[0])
			fmt.Println()
			if !firstDone {
				user.WEU = strings.Split(strings.Split(r.Headers.Values("Set-Cookie")[0], ";")[0], "=")[1]
				fmt.Println("get the first WEU: ", user.WEU)
				firstDone = true
			}
		}
	})

	// login post
	err = c.Post("https://authserver.szu.edu.cn/authserver/login?service=https%3A%2F%2Fehall.szu.edu.cn%3A443%2Fqljfwapp%2Fsys%2FlwSzuCgyy%2Findex.do%23%2FsportVenue",
		map[string]string{"username": user.UserId, "password": password, "lt": lt, "dllt": dllt, "execution": execution, "_eventId": _eventId})
	if err != nil {
		log.Fatal(err)
	}

	c.Wait()

	// get the temp final WEU
	tempFinalDone := false
	configUrl := "https://ehall.szu.edu.cn/qljfwapp/sys/lwSzuCgyy/index.do"

	c.OnRequest(func(r *colly.Request) {
		r.URL.Host = "ehall.szu.edu.cn"
		r.Headers.Del("Cookie")
		cookieString := "_WEU=" + user.WEU + ";" + "MOD_AUTH_CAS=" + user.MOD_AUTH_CAS
		fmt.Println("0 cookieString: ", cookieString)
		r.Headers.Add("Cookie", cookieString)
	})

	c.OnResponse(func(r *colly.Response) {
		fmt.Println("Temp Final CONFIG", r.Headers.Values("Set-Cookie"), len(r.Headers.Values("Set-Cookie")))

		if len(r.Headers.Values("Set-Cookie")) == 1 {
			fmt.Println("Temp Final CONFIG", r.Headers.Values("Set-Cookie")[0])
			if !tempFinalDone {
				user.WEU = strings.Split(strings.Split(r.Headers.Values("Set-Cookie")[0], ";")[0], "=")[1]
				fmt.Println("get the temp final WEU: ", user.WEU)
				fmt.Println()
				tempFinalDone = true
			}
		} else {
			fmt.Println("Temp Final CONFIG", r.Headers.Values("Set-Cookie")[0])
			fmt.Println("Temp Final CONFIG", r.Headers.Values("Set-Cookie")[1])
			fmt.Println(strings.Split(r.Headers.Values("Set-Cookie")[1], ";"), len(strings.Split(r.Headers.Values("Set-Cookie")[1], ";")))

			if !tempFinalDone {
				user.WEU = strings.Split(strings.Split(r.Headers.Values("Set-Cookie")[1], ";")[0], "=")[1]
				fmt.Println("get the temp final WEU: ", user.WEU)
				fmt.Println()
				tempFinalDone = true
			}
		}

	})

	c.Request("GET", configUrl, nil, nil, nil)

	c.Wait()

	fmt.Println(c)

	// time.Sleep(30 * time.Second)
}

func startRub(user *UserInfo, goroutineID int) bool {
	if user.SecondTime == "00:00" {
		user.secondRound = true
	}

	if user.IfExecNow != "" {
		fmt.Println("抢票中...")
		getTheToken(user)
		result := execRub(user, goroutineID)
		fmt.Println("抢票结束...")
		return result
	}

	// 每天的执行时间
	hour := 12
	minute := 29
	second := 56
	now := time.Now()
	next := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, second, 0, now.Location())
	if now.After(next) {
		// 如果当前已经是预设的时间之后，计算下一个时刻(明天这个点)
		next = next.Add(24 * time.Hour)
	}
	duration := next.Sub(now)
	// 创建定时器
	timer := time.NewTicker(duration)
	defer timer.Stop()
	// 定时调用
	for {
		select {
		case <-timer.C:
			fmt.Println("开始抢票...")
			getTheToken(user)
			execRub(user, goroutineID)
			fmt.Println("抢票结束...")
			return true
		}
	}
}

func stop(w http.ResponseWriter, r *http.Request) {
	t := template.Must(template.ParseFiles("./templates/stopGoroutine.html"))

	if r.Method != http.MethodPost {
		goroutinesInfo := make([]GoroutineInfo, 0)

		for _, v := range goroutines {
			goroutinesInfo = append(goroutinesInfo, *v)
		}

		t.Execute(w, struct {
			Infos []GoroutineInfo
		}{goroutinesInfo})
		return
	}

	// fmt.Println("identification", r.FormValue("identification"))

	// when a link to /stop, it will take a POST method, skip that
	if r.FormValue("user_id") == "" {
		fmt.Println("stop skip")
		return
	}

	id, err := strconv.Atoi(r.FormValue("identification"))
	if err != nil {
		log.Fatal(err)
	}

	goroutines[id].FirstStatus = true
	goroutines[id].SecondStatus = true

	// 等待删除完成
	deleteLock2.Lock()
	deletingFlag = true
	select {
	case <-deleteChan:
	}
	deletingFlag = false
	deleteLock2.Unlock()

	goroutinesInfo := make([]GoroutineInfo, 0)

	for _, v := range goroutines {
		goroutinesInfo = append(goroutinesInfo, *v)
	}

	t.Execute(w, struct {
		Infos []GoroutineInfo
	}{goroutinesInfo})
}

func main() {
	// user := UserInfo{
	// 	UserId:     "2210274049",
	// 	UserName:   "莫昌康",
	// 	Password:   "09010013",
	// 	SportDate:  "2024-09-17",
	// 	FirstTime:  "20:00",
	// 	SecondTime: "21:00",
	// 	IfExecNow:  "1",
	// }
	// getTheToken(&user)
	// startRub(&user)
	goroutines = make(map[int]*GoroutineInfo)

	server := http.Server{
		Addr: "127.0.0.1:8080",
	}
	http.HandleFunc("/", process)
	http.HandleFunc("/add", add)
	http.HandleFunc("/config", updateSMSConfig)
	http.HandleFunc("/stop", stop)

	log.Println("Listen at http://127.0.0.1:8080")
	server.ListenAndServe()
}
