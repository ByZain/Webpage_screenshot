package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"

	"github.com/Luxurioust/excelize"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/device"
)

var emulateChoose string
var de string
var wg sync.WaitGroup
var mu sync.Mutex
var fileName string

// PCfolderName 为文件夹名称，保存pc端截图
var PCfolderName string

// PhonefolderName 为文件夹名称，保存手机端截图
var PhonefolderName string
var targetChan chan string = make(chan string)
var imgNameChan chan string = make(chan string)

// KeyWords 存放关键字
var KeyWords string
var sleepTime int = 1

func screen() {
	defer wg.Done()
	fmt.Println("线程开启！")
	fmt.Println("开始抓取关键字......")
	// create context
	ctx, cancel := chromedp.NewContext(context.Background())

	defer cancel()

	// run
	var res string
	var b1, b2 []byte
	var highLight string = `var ele = document.getElementsByTagName("body")[0];
	var keys = "` + KeyWords + `";
	var reg = new RegExp("(" + keys.replace(/,/, "|") + ")", "g");
	ele.innerHTML = ele.innerHTML.replace(reg, "<span style='background:red;'><font color='#fff'>$1</font></span>");`

	//chromedp.UserAgent(`Mozilla/5.0 (Windows NT 6.3; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/73.0.3683.103 Safari/537.36`)
	for {
		select {
		case targetURL := <-targetChan:
			imgName := <-imgNameChan
			fmt.Println("正在抓取:"+targetURL, imgName)
			err := chromedp.Run(ctx,
				// emulate iPhone 7 landscape
				chromedp.Emulate(device.IPad),
				chromedp.Navigate(targetURL),
				chromedp.Sleep(time.Second*time.Duration(sleepTime)),
				chromedp.EvaluateAsDevTools(highLight, &res),
				chromedp.CaptureScreenshot(&b1),

				// reset
				chromedp.Emulate(device.Reset),

				// set really large viewport

				chromedp.EmulateViewport(1920, 2000),
				chromedp.Navigate(targetURL),
				chromedp.Sleep(time.Second*time.Duration(sleepTime)),
				chromedp.EvaluateAsDevTools(highLight, &res),
				chromedp.CaptureScreenshot(&b2),
			)
			mu.Lock()
			if err != nil {
				errorHTTPHandle, _ := os.OpenFile("errorHTTP.txt", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0777)
				io.WriteString(errorHTTPHandle, targetURL)
				fmt.Println(targetURL + "抓取失败！该地址保存在本程序同级目录下！")
				continue
			}
			mu.Unlock()
			getwd, _ := os.Getwd()
			filepath.Join(getwd, PCfolderName)
			filepath.Join(getwd, PhonefolderName)
			if err := ioutil.WriteFile(PhonefolderName+"/"+imgName+"_Phone.png", b1, 0777); err != nil {
				log.Fatal(err)
			}

			if err := ioutil.WriteFile(PCfolderName+"/"+imgName+"_PC.png", b2, 0777); err != nil {
				log.Fatal(err)
			}
		case <-time.After(time.Second * 1):
			fmt.Println("线程退出！")
			return
		}
	}
}

func imageName(URLString string) string {
	matchString := `[:\/\.\?+=]+`
	re := regexp.MustCompile(matchString)
	imageName := re.ReplaceAllString(URLString, "")
	return imageName
}

func makeFolder() {
	PCfolderName = time.Now().Format("20060102") + "_PC"
	PhonefolderName = time.Now().Format("20060102") + "_Phone"
	_, err := os.Stat(PCfolderName)
	if os.IsNotExist(err) {
		err := os.Mkdir(PCfolderName, 0777)
		if err != nil {
			panic(err)
		}
	}

	_, err = os.Stat(PhonefolderName)
	if os.IsNotExist(err) {
		err := os.Mkdir(PhonefolderName, 0777)
		if err != nil {
			panic(err)
		}
	}
}

// 通道生产者
func makeData(fileName string) {
	xlsx, err := excelize.OpenFile(fileName)
	if err != nil {
		panic(err)
	}

	allValueSlice, err := xlsx.GetRows("Sheet1")
	if err != nil {
		panic(err)
	}
	for i, v := range allValueSlice {
		if i == 0 {
			continue
		}
		if len(v) > 0 {
			targetChan <- v[2]
			imgNameChan <- v[0]
		}
	}
	wg.Done()
}

func main() {
	var threads int
	fmt.Println("请输入要高亮的关键字(多个关键字用|分隔)：")
	fmt.Scanln(&KeyWords)
	fmt.Println("请将文件拖入(文件名中不能存在空格)：")
	fmt.Scanln(&fileName)
	fmt.Println("请设置线程数量(数字)：")
	fmt.Scanln(&threads)
	fmt.Println("请设置等待时间(数字)：")
	fmt.Scanln(&sleepTime)
	makeFolder()
	wg.Add(1)
	go makeData(fileName)

	for i := 0; i < threads; i++ {
		wg.Add(1)
		go screen()
	}

	wg.Wait()
	fmt.Println("抓取结束！截图保存在此程序同级目录下")
	select {
	case <-time.After(time.Second * 3):

	}
}
