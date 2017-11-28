package chat

import (
	"encoding/base64"
	"errors"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"
	"strings"
)

func convertString(str string, format string, dest *os.File) error {

	imageReader := base64.NewDecoder(base64.StdEncoding, strings.NewReader(str))
	pngImage, _, err := image.Decode(imageReader)
	if err != nil {
		return err
	}
	defer dest.Close()
	switch format {
	case "png":
		return png.Encode(dest, pngImage)
	case "jpeg":
		return jpeg.Encode(dest, pngImage, &jpeg.Options{Quality: 80})
	case "gif":
		return gif.Encode(dest, pngImage, &gif.Options{})
	}
	return errors.New("format only jpeg/gif/png")
	// bytesData := bytes.NewReader([]byte(str))
	// log.Println(format)
	// switch format {
	// case "png":
	// 	log.Println("png")
	// 	im, err := png.Decode(bytesData)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	return png.Encode(dest, im)
	// case "jpeg":
	// 	log.Println("jpeg")
	// 	var opt jpeg.Options
	// 	opt.Quality = 80
	// 	im, err := jpeg.Decode(bytesData)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	return jpeg.Encode(dest, im, &opt)
	// case "gif":
	// 	log.Println("gif")
	// 	var opt gif.Options
	// 	im, err := gif.Decode(bytesData)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	return gif.Encode(dest, im, &opt)
	// }
	// return errors.New("format only jpeg/gif/png")

	// switch strings.TrimSuffix(str[5:coI], ";base64") {
	// case "image/png":
	// 	pngI, err := png.Decode(res)
	// 	// ...
	// case "image/jpeg":
	// 	jpgI, err := jpeg.Decode(res)
	// 	// ...
	// }
}
