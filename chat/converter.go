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
	//bytesData := bytes.NewReader([]byte(str))
	log.Println(format)
	switch format {
	case "png":
		log.Println("png")
		im, err := png.Decode(bytesData)
		if err != nil {
			return err
		}
		// e := base64.NewEncoder(base64.StdEncoding, dest)
		// _, err := e.Write(bytesData)
		// if err != nil {
		// 	log.Println(err.Error())
		// 	return err
		// }
		// err := e.Close()
		// if err != nil {
		// 	log.Println(err.Error())
		// 	return err
		// }
		// return nil
		return png.Encode(e, im)
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
	return errors.New("format only jpeg/gif/png")

	// switch strings.TrimSuffix(str[5:coI], ";base64") {
	// case "image/png":
	// 	pngI, err := png.Decode(res)
	// 	// ...
	// case "image/jpeg":
	// 	jpgI, err := jpeg.Decode(res)
	// 	// ...
	// }
}
