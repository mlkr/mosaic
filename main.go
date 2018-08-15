package main

import (
	"fmt"
	"html/template"
	"image"
	"image/draw"
	"net/http"
	"strconv"
	"time"
)

type ImgChan struct {
	seq int
	img image.Image
}

var db DB

func main() {
	mux := http.NewServeMux()

	files := http.FileServer(http.Dir("public"))
	mux.Handle("/static/", http.StripPrefix("/static/", files))

	mux.HandleFunc("/", index)
	mux.HandleFunc("/mosaic", mosaic)

	server := &http.Server{
		Addr:    "127.0.0.1:8080",
		Handler: mux,
	}

	fmt.Println("生成瓷片数据...")
	var err error
	db, err = tilesDB()
	if err != nil {
		fmt.Println("生成瓷片数据失败")
		panic(err)
	}
	fmt.Println("生成瓷片数据完成")

	fmt.Println("监听地址: ", server.Addr)
	err = server.ListenAndServe()
	fmt.Println(err)

}

func index(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("index.html")
	if err != nil {
		panic(err)
	}

	t.Execute(w, nil)
}

func mosaic(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// 取值
	r.ParseMultipartForm(10 << 20) //10M

	file, _, err := r.FormFile("image")
	if err != nil {
		fmt.Println("获取上传文件失败")
		panic(err)
	}

	img, _, err := image.Decode(file)
	if err != nil {
		fmt.Println("图片解码失败:")
		panic(err)
	}
	file.Close()

	tileSize, _ := strconv.Atoi(r.FormValue("tile_size"))

	// 替换
	dbC := db.cloneDB()

	// 图片分为 4 部分去完成
	//  0 | 1
	// --------
	//  2 | 3
	c0 := make(chan image.Image)
	c1 := make(chan image.Image)
	c2 := make(chan image.Image)
	c3 := make(chan image.Image)

	rect := img.Bounds()
	x := rect.Max.X / 2
	y := rect.Max.Y / 2

	dbC.send(img, image.Rect(0, 0, x, y), tileSize, c0)
	dbC.send(img, image.Rect(x, 0, rect.Max.X, y), tileSize, c1)
	dbC.send(img, image.Rect(0, y, x, rect.Max.Y), tileSize, c2)
	dbC.send(img, image.Rect(x, y, rect.Max.X, rect.Max.Y), tileSize, c3)

	newImgChan := receive(c0, c1, c2, c3, rect)

	// 返回数据
	end := time.Now()
	data := map[string]string{
		"original": img2base64(img),
		"mosaic":   img2base64(<-newImgChan),
		"duration": fmt.Sprintf("%v", end.Sub(start)),
	}

	t, _ := template.ParseFiles("results.html")
	t.Execute(w, data)
}

// 合并 chan 中的图像
func receive(c0, c1, c2, c3 chan image.Image, rect image.Rectangle) chan image.Image {
	c := make(chan image.Image)
	go func() {
		newImg := image.NewRGBA(rect)
		x := rect.Max.X / 2
		y := rect.Max.Y / 2

		for i := 0; i < 4; i++ {
			select {
			case img0 := <-c0:
				draw.Draw(newImg, image.Rect(0, 0, x, y), img0, image.Point{0, 0}, draw.Src)
			case img1 := <-c1:
				draw.Draw(newImg, image.Rect(x, 0, rect.Max.X, y), img1, image.Point{0, 0}, draw.Src)
			case img2 := <-c2:
				draw.Draw(newImg, image.Rect(0, y, x, rect.Max.Y), img2, image.Point{0, 0}, draw.Src)
			case img3 := <-c3:
				draw.Draw(newImg, image.Rect(x, y, rect.Max.X, rect.Max.Y), img3, image.Point{0, 0}, draw.Src)
			}
		}

		c <- newImg
	}()

	return c
}
