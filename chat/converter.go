package chat

import (
	"bytes"
	"encoding/base64"
	"errors"
	"image/gif"
	"image/jpeg"
	"image/png"
	"log"
	"os"
	"strings"
)

func convertString(str string, format string, dest *os.File) error {

	b64data := str[strings.IndexByte(str, ',')+1:]
	bytesArray, err := base64.StdEncoding.DecodeString(b64data)
	if err != nil {
		return err
	}
	bytesData := bytes.NewReader(bytesArray)
	log.Println(format)
	switch format {
	case "png":
		log.Println("png")
		im, err := png.Decode(bytesData)
		if err != nil {
			return err
		}
		return png.Encode(dest, im)
	case "jpeg":
		log.Println("jpeg")
		var opt jpeg.Options
		opt.Quality = 80
		im, err := jpeg.Decode(bytesData)
		if err != nil {
			return err
		}
		return jpeg.Encode(dest, im, &opt)
	case "gif":
		log.Println("gif")
		var opt gif.Options
		im, err := gif.Decode(bytesData)
		if err != nil {
			return err
		}
		return gif.Encode(dest, im, &opt)
	}
	dest.Close()
	return errors.New("format only jpeg/gif/png")
}
