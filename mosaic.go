package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	_ "image/jpeg"
	"io/ioutil"
	"math"
	"os"
	"sync"
)

type DB struct {
	// 瓷片的名字为键, rgb 为值
	tiles map[string][3]float64
	mutex sync.Mutex
}

// 生成瓷片
func tilesDB() (db DB, err error) {
	db.tiles = make(map[string][3]float64)

	// 读取图片文件夹信息
	fileInfos, err := ioutil.ReadDir("./tiles")
	if err == nil {

		for _, fileInfo := range fileInfos {
			name := fileInfo.Name()
			img, err := getImageByName("./tiles/" + name)
			if err == nil {
				// 计算图片的平均颜色
				db.tiles[name] = averageColor(img)
			} else {
				fmt.Println(err, "获取图片失败")
			}
		}

	} else {
		fmt.Println(err, "读取图片文件夹失败")
	}

	return
}

// 图片的平均颜色
func averageColor(img image.Image) [3]float64 {
	bounds := img.Bounds()
	rSum, gSum, bSum := 0.0, 0.0, 0.0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			rSum, gSum, bSum = rSum+float64(r), gSum+float64(g), bSum+float64(b)
		}
	}
	totalPixels := float64(bounds.Max.X * bounds.Max.Y)

	return [3]float64{rSum / totalPixels, gSum / totalPixels, bSum / totalPixels}
}

// 瓷片缩小
func imgZoomout(img image.Image, width int) image.Image {
	rImg := image.NewNRGBA(image.Rect(0, 0, width, width))
	if img == nil || width > img.Bounds().Dx() || width > img.Bounds().Dy() {
		fmt.Println("没有符合的照片")
		return rImg.SubImage(rImg.Bounds())
	}

	ratio := img.Bounds().Dx() / width
	for x := 0; x < width; x++ {
		for y := 0; y < width; y++ {
			r, g, b, a := img.At(x*ratio, y*ratio).RGBA()
			r = r >> 8
			g = g >> 8
			b = b >> 8
			a = a >> 8
			rImg.SetNRGBA(x, y, color.NRGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: uint8(a)})
		}
	}

	return rImg.SubImage(rImg.Bounds())
}

// 找出颜色最接近的瓷片
func (db *DB) nearest(c color.Color) image.Image {
	r, g, b, _ := c.RGBA()
	cf := [3]float64{float64(r), float64(g), float64(b)}

	// 找到差距最小的瓷片
	db.mutex.Lock()
	var imgName string
	smallest := 1000000.0
	for name, tile := range db.tiles {
		dist := distance(cf, tile)
		if dist < smallest {
			imgName, smallest = name, dist
		}
	}
	delete(db.tiles, imgName)
	db.mutex.Unlock()

	img, _ := getImageByName("./tiles/" + imgName)

	return img
}

// 克隆数据库
func (db *DB) cloneDB() DB {
	dbC := DB{}

	dbC.tiles = make(map[string][3]float64)
	for k, v := range db.tiles {
		dbC.tiles[k] = v
	}

	return dbC
}

// 替换
func (db *DB) exchange(img image.Image, bounds image.Rectangle, tileSize int) image.Image {
	newImg := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y += tileSize {
		for x := bounds.Min.X; x < bounds.Max.X; x += tileSize {
			tile := imgZoomout(db.nearest(img.At(x, y)), tileSize)
			draw.Draw(newImg, image.Rect(x, y, x+tileSize, y+tileSize), tile, image.Point{0, 0}, draw.Src)
		}
	}

	width := newImg.Bounds().Dx()
	height := newImg.Bounds().Dy()
	newImg.Rect = image.Rect(0, 0, width, height)

	return newImg.SubImage(newImg.Bounds())
}

// exchange 并发版
func (db *DB) send(img image.Image, rect image.Rectangle, tileSize int, c chan image.Image) {
	go func() {
		c <- db.exchange(img, rect, tileSize)
	}()
}

// 差距
func distance(p1 [3]float64, p2 [3]float64) float64 {
	return math.Sqrt(sq(p2[0]-p1[0]) + sq(p2[1]-p1[1]) + sq(p2[2]-p1[2]))
}

// 平方
func sq(n float64) float64 {
	return n * n
}

// 通过名字获取图片
func getImageByName(name string) (img image.Image, err error) {
	file, err := os.Open(name)
	if err == nil {
		img, _, err = image.Decode(file)
		if err != nil {
			fmt.Println(name, err, ": 解码图片失败")
		}
	} else {
		fmt.Println(name, err, ": 读取图片失败")
	}

	file.Close()

	return
}

// img 转 base64
func img2base64(img image.Image) string {
	options := &jpeg.Options{100}
	buf := new(bytes.Buffer)
	jpeg.Encode(buf, img, options)

	str := base64.StdEncoding.EncodeToString(buf.Bytes())

	return str
}

// 一张图片分成4张小图
//  0 | 1
// --------
//  2 | 3
func diviFourImg(img image.Image) [4]image.Image {
	x := img.Bounds().Max.X / 2
	y := img.Bounds().Max.Y / 2
	rect := image.Rect(0, 0, x, y)

	var newImgs [4]image.Image
	for i := 0; i < 4; i++ {
		var point image.Point
		switch i {
		case 0:
			point = image.Point{0, 0}
		case 1:
			point = image.Point{x, 0}
		case 2:
			point = image.Point{0, y}
		case 3:
			point = image.Point{x, y}
		}

		newImg := image.NewRGBA(rect)
		draw.Draw(newImg, newImg.Bounds(), img, point, draw.Src)
		newImgs[i] = newImg.SubImage(newImg.Bounds())
	}

	return newImgs
}

// 保存图片
func saveImg(img image.Image, name string) error {
	imgFile, _ := os.Create(name)
	err := jpeg.Encode(imgFile, img, &jpeg.Options{100})
	imgFile.Close()

	return err
}
