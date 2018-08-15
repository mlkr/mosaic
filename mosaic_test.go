package main

import (
	"fmt"
	"image"
	"testing"
)

func TestTilesDB(t *testing.T) {
	db, err := tilesDB()
	if err != nil {
		t.Fatal(err)
	}

	if len(db.tiles) == 0 {
		t.Fatal("生成的瓷片数据为空")
	}
}

func TestAverageColor(t *testing.T) {
	img, _ := getImageByName("./tiles/0002cbf69906a68fe2d8e7ef9e7c995e3bab0439.jpg")
	color := averageColor(img)
	fmt.Println(color)
}

func TestImgZoomout(t *testing.T) {
	img, _ := getImageByName("./tiles/0a0baafce7cb2fe062796c1ffd1d1bb38fdf82bb.jpg")

	width := 50
	newImg := imgZoomout(img, width)
	if newImg.Bounds().Dx() != width {
		t.Fatal(newImg.Bounds().Dx(), "缩小后的宽度不对")
	}
	if newImg.Bounds().Dy() != width {
		t.Fatal(newImg.Bounds().Dy(), "缩小后的高度不对")
	}

	saveImg(newImg, "newImg.jpg")
}

func TestNearest(t *testing.T) {
	db, err := tilesDB()

	img, err := getImageByName("./tiles/0a0baafce7cb2fe062796c1ffd1d1bb38fdf82bb.jpg")
	if err != nil {
		t.Fatal(err, "获取图片失败")
	}

	tilesLen := len(db.tiles)
	newImg := db.nearest(img.At(0, 0))
	saveImg(newImg, "newImg.jpg")

	newImg2 := db.nearest(img.At(0, 0))
	saveImg(newImg2, "newImg2.jpg")

	if tilesLen == len(db.tiles) {
		t.Fatal("数据库中的瓷片没有减少")
	}

}

func TestCloneDB(t *testing.T) {
	db, _ := tilesDB()
	dbC := db.cloneDB()

	if len(db.tiles) != len(dbC.tiles) {
		t.Fatal("克隆数据库与原数据库不相等")

		if len(dbC.tiles) == 0 {
			t.Fatal("克隆数据库为空")
		}
	}

	var name string
	for k, _ := range db.tiles {
		name = k
		break
	}
	delete(db.tiles, name)
	if len(db.tiles) == len(dbC.tiles) {
		t.Fatal("克隆数据库失败")
	}
}

func TestSaveImg(t *testing.T) {
	img, err := getImageByName("./tiles/0a0baafce7cb2fe062796c1ffd1d1bb38fdf82bb.jpg")
	if err != nil {
		t.Fatal(err, "获取图片失败")
	}

	err = saveImg(img, "abcd.jpg")
	if err != nil {
		t.Fatal(err, "保存图片失败")
	}
}

func TestDiviFourPic(t *testing.T) {
	img, _ := getImageByName("./tiles/0a0baafce7cb2fe062796c1ffd1d1bb38fdf82bb.jpg")
	imgs := diviFourImg(img)

	for i := range imgs {
		saveImg(imgs[i], fmt.Sprintf("%d.jpg", i))
	}
}

func TestExchange(t *testing.T) {
	db, _ := tilesDB()

	img, _ := getImageByName("cat.jpg")

	rect := img.Bounds()
	x := rect.Max.X / 2
	y := rect.Max.Y / 2
	img0 := db.exchange(img, image.Rect(0, 0, x, y), 10)
	img1 := db.exchange(img, image.Rect(x, 0, rect.Max.X, y), 10)
	img2 := db.exchange(img, image.Rect(0, y, x, rect.Max.Y), 10)
	img3 := db.exchange(img, image.Rect(x, y, rect.Max.X, rect.Max.Y), 10)

	saveImg(img0, "img0.jpg")
	saveImg(img1, "img1.jpg")
	saveImg(img2, "img2.jpg")
	saveImg(img3, "img3.jpg")

	if img3.Bounds().Min.X != 0 || img3.Bounds().Min.Y != 0 {
		t.Fatal("返回的图片 bounds 错误", img3.Bounds())
	}

}
